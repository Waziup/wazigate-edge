package api

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"testing"
	"time"
)

// buildExportZip creates an archive with the same layout as exportTree().
func buildExportZip(t *testing.T, files map[string][][]string) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	// directory entries like `zip -r` creates them
	for _, dir := range []string{"exportTree/", "exportTree/dev1/", "exportTree/dev1/temp/"} {
		if _, err := w.Create(dir); err != nil {
			t.Fatal(err)
		}
	}
	for name, rows := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		cw := csv.NewWriter(f)
		if err := cw.WriteAll(rows); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestParseImportTree(t *testing.T) {
	archive := buildExportZip(t, map[string][][]string{
		"exportTree/devices.csv": {
			{"dev1", "Device 1", "2024-01-01 10:00:00 +0100 CET", "2024-01-01 10:00:00 +0100 CET", `{"codec":"application/x-xlpp"}`},
			{"dev2", "Device 2", "", "", "null"},
		},
		// rows are padded to 7 columns by the export
		"exportTree/dev1/sensors.csv": {
			{"temp", "Temperature", "2024-01-01T10:00:00+01:00", "2024-01-01T10:00:00+01:00", `{"kind":"AirThermometer"}`, "", ""},
		},
		"exportTree/dev1/actuators.csv": {
			{"relay", "Relay", "2024-01-01T10:00:00+01:00", "2024-01-01T10:00:00+01:00", "null", "", ""},
		},
		// deliberately out of order
		"exportTree/dev1/temp/values.csv": {
			{"2024-01-01T10:02:00+01:00", "21.5"},
			{"2024-01-01T10:01:00+01:00", "20"},
			{"2024-01-01T10:03:00+01:00", `"text"`},
			{"2024-01-01T10:04:00+01:00", "not json"},
		},
		"exportTree/dev1/relay/values.csv": {
			{"2024-01-01T10:05:00+01:00", "true"},
		},
		// dev2 has empty sensors.csv and no actuators.csv
		"exportTree/dev2/sensors.csv": {},
	})

	devices, err := ParseImportTree(archive)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	dev1 := devices[0]
	if dev1.ID != "dev1" || dev1.Name != "Device 1" {
		t.Errorf("unexpected device: %+v", dev1)
	}
	if dev1.Meta["codec"] != "application/x-xlpp" {
		t.Errorf("device meta not parsed: %v", dev1.Meta)
	}
	if len(dev1.Sensors) != 1 || len(dev1.Actuators) != 1 {
		t.Fatalf("expected 1 sensor and 1 actuator, got %d / %d", len(dev1.Sensors), len(dev1.Actuators))
	}

	temp := dev1.Sensors[0]
	if temp.ID != "temp" || temp.Name != "Temperature" || temp.Meta["kind"] != "AirThermometer" {
		t.Errorf("unexpected sensor: %+v", temp)
	}
	if len(temp.Values) != 4 {
		t.Fatalf("expected 4 values, got %d", len(temp.Values))
	}
	wantValues := []interface{}{float64(20), 21.5, "text", "not json"}
	for i, v := range temp.Values {
		if v.Value != wantValues[i] {
			t.Errorf("value %d: got %#v, want %#v", i, v.Value, wantValues[i])
		}
		if i > 0 && !temp.Values[i-1].Time.Before(v.Time) {
			t.Errorf("values not sorted by time at %d", i)
		}
	}
	wantTime, _ := time.Parse(time.RFC3339, "2024-01-01T09:01:00Z")
	if !temp.Values[0].Time.Equal(wantTime) {
		t.Errorf("time: got %v, want %v", temp.Values[0].Time, wantTime)
	}

	relay := dev1.Actuators[0]
	if relay.Meta != nil {
		t.Errorf("expected nil meta for 'null', got %v", relay.Meta)
	}
	if len(relay.Values) != 1 || relay.Values[0].Value != true {
		t.Errorf("unexpected actuator values: %+v", relay.Values)
	}

	dev2 := devices[1]
	if dev2.Meta != nil || len(dev2.Sensors) != 0 || len(dev2.Actuators) != 0 {
		t.Errorf("unexpected device 2: %+v", dev2)
	}
}

func TestParseImportTreeWithoutPrefix(t *testing.T) {
	archive := buildExportZip(t, map[string][][]string{
		"devices.csv":          {{"dev1", "Device 1"}},
		"dev1/sensors.csv":     {{"s1", "Sensor 1"}},
		"dev1/s1/values.csv":   {{"2024-01-01T10:00:00Z", "1"}},
		"dev1/actuators.csv":   {},
		"unrelated/readme.txt": {{"ignored"}},
	})
	devices, err := ParseImportTree(archive)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || len(devices[0].Sensors) != 1 || len(devices[0].Sensors[0].Values) != 1 {
		t.Fatalf("unexpected result: %+v", devices)
	}
}

func TestParseImportTreeErrors(t *testing.T) {
	cases := map[string]map[string][][]string{
		"no devices.csv": {"foo.csv": {{"a"}}},
		"bad meta":       {"devices.csv": {{"dev1", "Device 1", "", "", "{broken"}}},
		"bad time": {
			"devices.csv":        {{"dev1", "Device 1"}},
			"dev1/sensors.csv":   {{"s1", "Sensor 1"}},
			"dev1/s1/values.csv": {{"yesterday", "1"}},
		},
	}
	for name, files := range cases {
		if _, err := ParseImportTree(buildExportZip(t, files)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
