package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Waziup/wazigate-edge/api"
)

var time1, _ = time.Parse(api.TimeFormat, "2019-05-13T11:35:24.002Z")
var time2, _ = time.Parse(api.TimeFormat, "2019-05-13T11:36:24.002Z")

var device1 = &api.Device{
	ID: "5cd92df34b9f6126f840f0b1",
	Sensors: []*api.Sensor{
		{
			ID:    "df34b9f612",
			Name:  "tempsensor1",
			Time:  time1,
			Value: 7,
		},
		{
			ID:    "6f840f0b1",
			Name:  "tempsensor2",
			Time:  time2,
			Value: "65",
		},
	},
	Actuators: []*api.Actuator{},
	/* Actuators: []*api.Actuator{
		&api.Actuator{
			ID:    "cd92df3",
			Name:  "actua1",
			Value: 7.4,
		},
	}, */
}

func TestRestAPI(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping full test in short mode.")
	}

	// checkMongoVersion(t)
	// mongo := runMongoBackground(t)
	// defer mongo.Process.Kill()
	// edge := runEdgeBackground(t)
	// defer edge.Process.Kill()

	t.Log("Test: check if there are no devics right now")
	get(t, "/devices", http.StatusOK, []*api.Device{})

	defer delete(t, "/devices/"+device1.ID, 0, nil)

	t.Log("Test: insert one device")
	post(t, "/devices", device1, http.StatusOK, nil)

	t.Log("Test: unexisting device")
	get(t, "/devices/notfound", http.StatusNotFound, nil)

	t.Log("Test: check if the inserted device exists")
	get(t, "/devices", http.StatusOK, []*api.Device{device1})

	t.Log("Test: check sensors")
	sensor := device1.Sensors[0]
	get(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID, http.StatusOK, sensor)
	sensor = device1.Sensors[1]
	get(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID, http.StatusOK, sensor)

	t.Log("Test: unexisting sensor")
	get(t, "/devices/"+device1.ID+"/sensors/notfound", http.StatusNotFound, nil)

	t.Log("Test: get sensor value")
	value := sensor.Value
	get(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID+"/value", http.StatusOK, value)

	t.Log("Test: set sensor value")
	post(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID+"/value", 8, http.StatusOK, nil)

	t.Log("Test: get sensor values")
	resp := get(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID+"/values", http.StatusOK, nil)
	values, ok := resp.([]interface{})
	if !ok || len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}

	t.Log("Test: delete a sensor")
	delete(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID, http.StatusOK, nil)
	get(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID, http.StatusNotFound, nil)
	resp = get(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID+"/values", http.StatusOK, nil)
	values, ok = resp.([]interface{})
	if !ok || len(values) != 0 {
		t.Fatalf("expected 0 values, got %d", len(values))
	}

	t.Log("Test: delete this device")
	delete(t, "/devices/"+device1.ID, http.StatusOK, nil)

	t.Log("Test: check if there are no devics right now")
	get(t, "/devices", http.StatusOK, []*api.Device{})
}

func TestMQTTAPI(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping full test in short mode.")
	}

	t.Log("Test: insert one device")
	post(t, "/devices", device1, http.StatusOK, nil)

	defer delete(t, "/devices/"+device1.ID, 0, nil)

	t.Log("Test: subscribe to topic")
	sensor := device1.Sensors[0]
	cmd, subs := subscribe(t, "devices/"+device1.ID+"/sensors/"+sensor.ID+"/value")
	defer cmd.Process.Kill()

	time.Sleep(2 * time.Second)

	post(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID+"/value", "Hello :)", http.StatusOK, nil)
	post(t, "/devices/"+device1.ID+"/sensors/"+sensor.ID+"/value", 465, http.StatusOK, nil)

	line, err := subs.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil || line != "\"Hello :)\"" {
		t.Fatalf("expected \"Hello :)\", got %v (%v)", line, err)
	}
	line, err = subs.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil || line != "465" {
		t.Fatalf("expected 465, got %v (%v)", line, err)
	}

	publishT(t, "devices/"+device1.ID+"/sensors/"+sensor.ID+"/value", "\"WooHoo\"")
	line, err = subs.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil || line != "\"WooHoo\"" {
		t.Fatalf("expected \"WooHoo\", got %v (%v)", line, err)
	}
}

////////////////////////////////////////////////////////////////////////////////

func post(t *testing.T, path string, body interface{}, expectedCode int, expect interface{}) {
	t.Log("POST", path)
	data, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	reader := bytes.NewReader([]byte(data))
	resp, err := http.Post("http://127.0.0.1"+path, "application/json", reader)
	if err != nil {
		t.Fatalf("failed %q\n%v\n%v", path, body, err)
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed %q\n%v\n%v", path, body, err)
	}
	if resp.StatusCode != expectedCode {
		t.Fatalf("unexpected status %q\n- expected:\n%d\n- got:\n%d\n%s", path, expectedCode, resp.StatusCode, data)
	}
	var response interface{}
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("non json response %q\n%s", path, data)
	}
	if expect != nil {
		if !deepEqual(response, expect) {
			t.Fatalf("unexpected result %q\n- expected:\n%#v\n- got:\n%#v", path, expect, response)
		}
	}
}

func deepEqual(a, b interface{}) bool {
	aBin, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	bBin, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}
	var ai, bi interface{}
	err = json.Unmarshal(aBin, &ai)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bBin, &bi)
	if err != nil {
		panic(err)
	}
	return reflect.DeepEqual(ai, bi)
}

func get(t *testing.T, path string, expectedCode int, expect interface{}) interface{} {
	t.Log("GET", path)
	resp, err := http.Get("http://127.0.0.1" + path)
	if err != nil {
		t.Fatalf("failed %q\n%v", path, err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed %q\n%v", path, err)
	}
	if resp.StatusCode != expectedCode {
		t.Fatalf("unexpected status %q\n- expected:\n%d\n- got:\n%d\n%s", path, expectedCode, resp.StatusCode, data)
	}
	var response interface{}
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("non json response %q\n%s", path, data)
	}
	if expect != nil {
		if !deepEqual(response, expect) {
			t.Fatalf("unexpected result %q\n- expected:\n%#v\n- got:\n%#v", path, expect, response)
		}
	}
	return response
}

func delete(t *testing.T, path string, expectedCode int, expect interface{}) {
	t.Log("DELETE", path)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodDelete, "http://127.0.0.1"+path, nil)
	if err != nil {
		t.Fatalf("failed %q\n%v", path, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed %q\n%v", path, err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed %q\n%v", path, err)
	}
	if expectedCode != 0 && resp.StatusCode != expectedCode {
		t.Fatalf("unexpected status %q\n- expected:\n%d\n- got:\n%d\n%s", path, expectedCode, resp.StatusCode, data)
	}
	var response interface{}
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("non json response %q\n%s", path, data)
	}
	if expect != nil {
		if !deepEqual(response, expect) {
			t.Fatalf("unexpected result %q\n- expected:\n%#v\n- got:\n%#v", path, expect, response)
		}
	}
}

// func checkMongoVersion(t *testing.T) {
// 	t.Log("$ mongod --version")
// 	mongo := exec.Command("mongod", "--version")
// 	out, err := mongo.CombinedOutput()
// 	if err != nil {
// 		t.Fatalf("can not get mongod version. mongod is required for this test.\n%v", err)
// 	}
// 	i := bytes.IndexByte(out, '\n')
// 	if i == -1 {
// 		t.Fatalf("mongod unusual output:\n%v", string(out))
// 	}
// 	t.Log(string(out[:i]))
// }

// func runMongoBackground(t *testing.T) *exec.Cmd {
// 	t.Log("$ mongod --dbpath ./test/db --port 27019")
// 	mongo := exec.Command("mongod", "--dbpath", "./test/db", "--port", "27019")
// 	mongo.Stdout = os.Stdout
// 	mongo.Stderr = os.Stderr
// 	err := mongo.Start()
// 	if err != nil {
// 		t.Fatalf("mongod is required for this test.\n%v", err)
// 	}
// 	return mongo
// }

// func runEdgeBackground(t *testing.T) *exec.Cmd {
// 	t.Log("$ wazigate-edge")
// 	edge := exec.Command("wazigate-edge")
// 	edge.Stdout = os.Stdout
// 	edge.Stderr = os.Stderr
// 	err := edge.Start()
// 	if err != nil {
// 		t.Fatalf("wazigate-edge.exe is required for this test.\n%v", err)
// 	}
// 	return edge
// }

func subscribe(t *testing.T, topic string) (*exec.Cmd, *bufio.Reader) {
	t.Log("SUBSCRIBE", topic)
	subscriber := exec.Command("mosquitto_sub", "--protocol-version", "mqttv31", "--topic", topic)
	out, err := subscriber.StdoutPipe()
	if err != nil {
		t.Fatal("mosquitto_sub StdoutPipe", err)
	}
	err = subscriber.Start()
	if err != nil {
		t.Fatal("mosquitto_sub", err)
	}
	return subscriber, bufio.NewReader(out)
}

func publishT(t *testing.T, topic string, msg string) {
	t.Log("PUBLISH", topic, msg)
	publisher := exec.Command("mosquitto_pub", "--protocol-version", "mqttv31", "--topic", topic, "-q", "1", "-m", msg)
	err := publisher.Run()
	if err != nil {
		out, _ := publisher.CombinedOutput()
		t.Fatal("mosquitto_pub", err, out)
	}
}
