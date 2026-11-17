package api

// Import is the counterpart of exportTree() in export.go.
// It reads the ZIP archive produced by GET /exporttree and writes the devices,
// sensors, actuators and their values into this gateway.
//
// Expected archive layout (as created by exportTree):
//
//	exportTree/devices.csv                          id,name,created,modified,meta(json)
//	exportTree/<deviceID>/sensors.csv               id,name,created,modified,meta(json),,
//	exportTree/<deviceID>/actuators.csv             id,name,created,modified,meta(json),,
//	exportTree/<deviceID>/<probeID>/values.csv      time(RFC3339),value(json)
//
// The "exportTree/" prefix is optional, the root is wherever devices.csv is found.

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/tools"
	routing "github.com/julienschmidt/httprouter"
)

// ImportMode defines how existing devices on this gateway are treated.
type ImportMode string

const (
	// ImportMerge keeps existing devices, adds missing sensors/actuators and
	// adds only values that are not already present (same second + same value).
	// Importing the same archive twice is therefore idempotent.
	ImportMerge ImportMode = "merge"
	// ImportReplace deletes existing devices with the same ID (including all
	// their values) before importing them. The gateway's own device is never
	// deleted, it is merged instead.
	ImportReplace ImportMode = "replace"
)

// ImportProbe is a sensor or actuator read from the export archive.
type ImportProbe struct {
	ID     string
	Name   string
	Meta   edge.Meta
	Values []edge.Value
}

// ImportDevice is a device read from the export archive.
type ImportDevice struct {
	ID        string
	Name      string
	Meta      edge.Meta
	Sensors   []*ImportProbe
	Actuators []*ImportProbe
}

// ImportReport summarizes what an import did.
type ImportReport struct {
	Mode             ImportMode `json:"mode"`
	DevicesCreated   int        `json:"devicesCreated"`
	DevicesMerged    int        `json:"devicesMerged"`
	DevicesReplaced  int        `json:"devicesReplaced"`
	SensorsCreated   int        `json:"sensorsCreated"`
	ActuatorsCreated int        `json:"actuatorsCreated"`
	ValuesImported   int        `json:"valuesImported"`
	ValuesSkipped    int        `json:"valuesSkipped"`
	Errors           []string   `json:"errors"`
}

func (r *ImportReport) addError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	log.Printf("[IMPRT] Error: %s", msg)
	r.Errors = append(r.Errors, msg)
}

////////////////////////////////////////////////////////////////////////////////
// HTTP handler

// PostImportTree implements POST /importtree?mode=merge|replace
//
// The body is either the raw ZIP file (Content-Type: application/zip) or a
// multipart/form-data upload with the ZIP in the field "file".
//
// Example:
//
//	curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/zip" \
//	     --data-binary @exportTree.zip "http://localhost/importtree?mode=merge"
func PostImportTree(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	mode := ImportMode(req.URL.Query().Get("mode"))
	if mode == "" {
		mode = ImportMerge
	}
	if mode != ImportMerge && mode != ImportReplace {
		http.Error(resp, "bad request: mode must be 'merge' or 'replace'", http.StatusBadRequest)
		return
	}

	var archive *zip.Reader
	var err error

	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		file, header, err := req.FormFile("file")
		if err != nil {
			http.Error(resp, "bad request: missing form field 'file': "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()
		archive, err = zip.NewReader(file, header.Size)
		if err != nil {
			http.Error(resp, "bad request: not a zip file: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		// Serve() in main.go has already buffered the whole POST body in memory.
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		archive, err = zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			http.Error(resp, "bad request: not a zip file: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	devices, err := ParseImportTree(archive)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	report := ImportTree(devices, mode)

	resp.Header().Set("Content-Type", "application/json")
	json.NewEncoder(resp).Encode(report)

	// Serve() republishes every successful POST body via MQTT (topic "importtree").
	// Publish the small report instead of the whole zip archive.
	tools.SetRequestBody(req, report)
}

////////////////////////////////////////////////////////////////////////////////
// Parsing (no database access, can be unit tested)

// ParseImportTree reads an archive created by exportTree() into memory.
func ParseImportTree(archive *zip.Reader) ([]*ImportDevice, error) {

	// Index all files by their normalized path.
	files := make(map[string]*zip.File, len(archive.File))
	root := ""
	rootFound := false
	for _, f := range archive.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if strings.HasSuffix(name, "/") {
			continue // directory entry
		}
		files[name] = f
		if path.Base(name) == "devices.csv" {
			dir := path.Dir(name)
			if dir == "." {
				dir = ""
			}
			// Prefer the top most devices.csv
			if !rootFound || len(dir) < len(root) {
				root = dir
				rootFound = true
			}
		}
	}
	if !rootFound {
		return nil, fmt.Errorf("archive contains no devices.csv")
	}

	join := func(elem ...string) string {
		if root != "" {
			elem = append([]string{root}, elem...)
		}
		return path.Join(elem...)
	}

	deviceRows, err := readZipCsv(files, join("devices.csv"))
	if err != nil {
		return nil, err
	}

	devices := make([]*ImportDevice, 0, len(deviceRows))
	for i, row := range deviceRows {
		if len(row) < 2 || row[0] == "" {
			return nil, fmt.Errorf("devices.csv line %d: expected at least id and name", i+1)
		}
		device := &ImportDevice{
			ID:   row[0],
			Name: row[1],
		}
		if device.Meta, err = parseImportMeta(row, 4); err != nil {
			return nil, fmt.Errorf("devices.csv line %d: %v", i+1, err)
		}

		if device.Sensors, err = parseImportProbes(files, join, device.ID, "sensors.csv"); err != nil {
			return nil, err
		}
		if device.Actuators, err = parseImportProbes(files, join, device.ID, "actuators.csv"); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}

	return devices, nil
}

func parseImportProbes(files map[string]*zip.File, join func(...string) string, deviceID string, csvName string) ([]*ImportProbe, error) {

	file := join(deviceID, csvName)
	if _, ok := files[file]; !ok {
		return nil, nil // device without sensors / actuators
	}
	rows, err := readZipCsv(files, file)
	if err != nil {
		return nil, err
	}

	probes := make([]*ImportProbe, 0, len(rows))
	for i, row := range rows {
		if len(row) < 2 || row[0] == "" {
			return nil, fmt.Errorf("%s line %d: expected at least id and name", file, i+1)
		}
		probe := &ImportProbe{
			ID:   row[0],
			Name: row[1],
		}
		if probe.Meta, err = parseImportMeta(row, 4); err != nil {
			return nil, fmt.Errorf("%s line %d: %v", file, i+1, err)
		}

		valuesFile := join(deviceID, probe.ID, "values.csv")
		if _, ok := files[valuesFile]; ok {
			valueRows, err := readZipCsv(files, valuesFile)
			if err != nil {
				return nil, err
			}
			probe.Values = make([]edge.Value, 0, len(valueRows))
			for j, vrow := range valueRows {
				if len(vrow) < 2 {
					return nil, fmt.Errorf("%s line %d: expected time and value", valuesFile, j+1)
				}
				t, err := time.Parse(time.RFC3339, vrow[0])
				if err != nil {
					return nil, fmt.Errorf("%s line %d: %v", valuesFile, j+1, err)
				}
				probe.Values = append(probe.Values, edge.Value{
					Time:  t,
					Value: parseImportValue(vrow[1]),
				})
			}
			// edge.Post*Values() sets the probe's current value to the last
			// element, so the values must be in chronological order.
			sort.SliceStable(probe.Values, func(a, b int) bool {
				return probe.Values[a].Time.Before(probe.Values[b].Time)
			})
		}
		probes = append(probes, probe)
	}
	return probes, nil
}

func readZipCsv(files map[string]*zip.File, name string) ([][]string, error) {
	f, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("archive is missing %s", name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("can not open %s: %v", name, err)
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	reader.FieldsPerRecord = -1 // export writes padded rows
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("can not parse %s: %v", name, err)
	}
	return rows, nil
}

// parseImportMeta reads the JSON meta column. Empty or "null" means no meta.
func parseImportMeta(row []string, col int) (edge.Meta, error) {
	if len(row) <= col || row[col] == "" || row[col] == "null" {
		return nil, nil
	}
	var meta edge.Meta
	if err := json.Unmarshal([]byte(row[col]), &meta); err != nil {
		return nil, fmt.Errorf("invalid meta JSON: %v", err)
	}
	return meta, nil
}

// parseImportValue reads the JSON encoded value column. Values that are not
// valid JSON (e.g. hand written test data) are imported as plain strings.
func parseImportValue(s string) interface{} {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	return v
}

////////////////////////////////////////////////////////////////////////////////
// Writing to the database

// ImportTree writes the parsed devices into this gateway.
// Errors of single devices/probes are collected in the report and do not stop the import.
func ImportTree(devices []*ImportDevice, mode ImportMode) *ImportReport {

	report := &ImportReport{Mode: mode, Errors: []string{}}
	localID := edge.LocalID()

	for _, device := range devices {

		existing, err := edge.GetDevice(device.ID)
		if err != nil && !isNotFound(err) {
			report.addError("device %s: %v", device.ID, err)
			continue
		}
		exists := err == nil
		replaced := false

		// Never delete the gateway's own device, always merge it.
		if exists && mode == ImportReplace && device.ID != localID {
			_, numS, numA, err := edge.DeleteDevice(device.ID)
			if err != nil {
				report.addError("device %s: can not delete existing device: %v", device.ID, err)
				continue
			}
			log.Printf("[IMPRT] Removed device %s for replacement (%d sensor values, %d actuator values).", device.ID, numS, numA)
			clouds.FlagDevice(device.ID, clouds.ActionDelete, nil)
			exists = false
			replaced = true
		}

		if exists {
			report.DevicesMerged++
			importMergeDevice(report, device, existing)
			continue
		}

		if err := importCreateDevice(device); err != nil {
			report.addError("device %s: %v", device.ID, err)
			continue
		}
		if replaced {
			report.DevicesReplaced++
		} else {
			report.DevicesCreated++
		}
		report.SensorsCreated += len(device.Sensors)
		report.ActuatorsCreated += len(device.Actuators)

		for _, sensor := range device.Sensors {
			importSensorValues(report, device.ID, sensor)
		}
		for _, actuator := range device.Actuators {
			importActuatorValues(report, device.ID, actuator)
		}
	}

	log.Printf("[IMPRT] Import finished: %d created, %d merged, %d replaced, %d values imported, %d skipped, %d errors.",
		report.DevicesCreated, report.DevicesMerged, report.DevicesReplaced,
		report.ValuesImported, report.ValuesSkipped, len(report.Errors))
	return report
}

func isNotFound(err error) bool {
	codeErr, ok := err.(edge.CodeError)
	return ok && codeErr.Code == http.StatusNotFound
}

// importCreateDevice creates a new device including all its sensors and actuators (without values).
func importCreateDevice(device *ImportDevice) error {

	newDevice := &edge.Device{
		ID:        device.ID,
		Name:      device.Name,
		Meta:      device.Meta,
		Sensors:   make([]*edge.Sensor, 0, len(device.Sensors)),
		Actuators: make([]*edge.Actuator, 0, len(device.Actuators)),
	}
	for _, s := range device.Sensors {
		newDevice.Sensors = append(newDevice.Sensors, &edge.Sensor{ID: s.ID, Name: s.Name, Meta: s.Meta})
	}
	for _, a := range device.Actuators {
		newDevice.Actuators = append(newDevice.Actuators, &edge.Actuator{ID: a.ID, Name: a.Name, Meta: a.Meta})
	}

	if err := edge.PostDevices(newDevice); err != nil {
		return err
	}
	log.Printf("[IMPRT] Created device %s (%d sensors, %d actuators).", device.ID, len(device.Sensors), len(device.Actuators))
	clouds.FlagDevice(device.ID, clouds.ActionCreate, newDevice.Meta)
	return nil
}

// importMergeDevice adds missing sensors/actuators and missing values to an existing device.
// Name and meta of existing entities are left untouched.
func importMergeDevice(report *ImportReport, device *ImportDevice, existing *edge.Device) {

	existingSensors := make(map[string]bool, len(existing.Sensors))
	for _, s := range existing.Sensors {
		existingSensors[s.ID] = true
	}
	existingActuators := make(map[string]bool, len(existing.Actuators))
	for _, a := range existing.Actuators {
		existingActuators[a.ID] = true
	}

	for _, sensor := range device.Sensors {
		if !existingSensors[sensor.ID] {
			newSensor := &edge.Sensor{ID: sensor.ID, Name: sensor.Name, Meta: sensor.Meta}
			if err := edge.PostSensor(device.ID, newSensor); err != nil {
				report.addError("sensor %s/%s: %v", device.ID, sensor.ID, err)
				continue
			}
			clouds.FlagSensor(device.ID, sensor.ID, clouds.ActionCreate, noTime, newSensor.Meta)
			report.SensorsCreated++
		} else {
			sensor.Values = importDropDuplicates(report, sensor.Values,
				edge.GetSensorValues(device.ID, sensor.ID, importValuesQuery(sensor.Values)))
		}
		importSensorValues(report, device.ID, sensor)
	}

	for _, actuator := range device.Actuators {
		if !existingActuators[actuator.ID] {
			newActuator := &edge.Actuator{ID: actuator.ID, Name: actuator.Name, Meta: actuator.Meta}
			if err := edge.PostActuator(device.ID, newActuator); err != nil {
				report.addError("actuator %s/%s: %v", device.ID, actuator.ID, err)
				continue
			}
			clouds.FlagActuator(device.ID, actuator.ID, clouds.ActionCreate, noTime, newActuator.Meta)
			report.ActuatorsCreated++
		} else {
			actuator.Values = importDropDuplicates(report, actuator.Values,
				edge.GetActuatorValues(device.ID, actuator.ID, importValuesQuery(actuator.Values)))
		}
		importActuatorValues(report, device.ID, actuator)
	}
}

// importValuesQuery covers the time range of the (sorted) values to import.
func importValuesQuery(values []edge.Value) *edge.ValuesQuery {
	if len(values) == 0 {
		return &edge.ValuesQuery{}
	}
	return &edge.ValuesQuery{
		From: values[0].Time.Add(-time.Second),
		To:   values[len(values)-1].Time.Add(time.Second),
	}
}

// importDropDuplicates removes values that already exist on this gateway.
// The export only keeps whole seconds, so values are compared by second + JSON value.
func importDropDuplicates(report *ImportReport, values []edge.Value, iter edge.ValueIterator) []edge.Value {
	defer iter.Close()
	if len(values) == 0 {
		return values
	}

	key := func(v edge.Value) string {
		data, _ := json.Marshal(v.Value)
		return strconv.FormatInt(v.Time.Unix(), 10) + "|" + string(data)
	}

	present := make(map[string]int)
	for {
		v, err := iter.Next()
		if err != nil {
			break // io.EOF or db error, either way we are done
		}
		present[key(v)]++
	}
	if len(present) == 0 {
		return values
	}

	filtered := make([]edge.Value, 0, len(values))
	for _, v := range values {
		k := key(v)
		if present[k] > 0 {
			present[k]--
			report.ValuesSkipped++
			continue
		}
		filtered = append(filtered, v)
	}
	return filtered
}

func importSensorValues(report *ImportReport, deviceID string, sensor *ImportProbe) {
	if len(sensor.Values) == 0 {
		return
	}
	meta, err := edge.PostSensorValues(deviceID, sensor.ID, sensor.Values)
	if err != nil {
		report.addError("sensor %s/%s: can not import values: %v", deviceID, sensor.ID, err)
		return
	}
	report.ValuesImported += len(sensor.Values)
	log.Printf("[IMPRT] %d values for sensor %s/%s.", len(sensor.Values), deviceID, sensor.ID)
	clouds.FlagSensor(deviceID, sensor.ID, clouds.ActionSync, sensor.Values[0].Time, meta)
}

func importActuatorValues(report *ImportReport, deviceID string, actuator *ImportProbe) {
	if len(actuator.Values) == 0 {
		return
	}
	meta, err := edge.PostActuatorValues(deviceID, actuator.ID, actuator.Values)
	if err != nil {
		report.addError("actuator %s/%s: can not import values: %v", deviceID, actuator.ID, err)
		return
	}
	report.ValuesImported += len(actuator.Values)
	log.Printf("[IMPRT] %d values for actuator %s/%s.", len(actuator.Values), deviceID, actuator.ID)
	clouds.FlagActuator(deviceID, actuator.ID, clouds.ActionSync, actuator.Values[0].Time, meta)
}
