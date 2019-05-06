package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Waziup/waziup-edge/api"
	"github.com/globalsign/mgo"
)

func TestWaziupEdge(t *testing.T) {
	cleanup := initTests(t)
	defer cleanup()

	post(t, "/devices", "", "[]")
}

func post(t *testing.T, path string, body string, expect string) {
	t.Log("POST", path)
	reader := bytes.NewReader([]byte(body))
	resp, err := http.Post("http://127.0.0.1"+path, "application/json", reader)
	if err != nil {
		t.Fatalf("failed to post %q\n%v\n%v", path, body, err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to post %q\n%v\n%v", path, body, err)
	}
	r := strings.TrimSpace(string(data))
	expect = strings.TrimSpace(expect)
	if r != expect {
		t.Fatalf("unexpected result %q\n- expected:\n%v\n- got:\n%v", path, expect, r)
	}
}

func initTests(t *testing.T) func() {

	if testing.Short() {
		t.Skip("skipping full test in short mode.")
	}

	if api.DBSensorValues == nil || api.DBActuatorValues == nil || api.DBDevices == nil {

		os.RemoveAll("test/db")
		os.MkdirAll("test/db", 0666)

		checkMongoVersion(t)
		mongo := runMongoBackground(t)
		cleanup := func() {
			mongo.Process.Kill()
			os.RemoveAll("test/db")
		}
		dialMongo(t)
		return cleanup
	}

	cleanup := func() {}
	return cleanup
}

func checkMongoVersion(t *testing.T) {
	t.Log("$ mongod --version")
	mongo := exec.Command("mongod", "--version")
	out, err := mongo.CombinedOutput()
	if err != nil {
		t.Fatalf("can not get mongod version. mongod is required for this test.\n%v", err)
	}
	i := bytes.IndexByte(out, '\n')
	if i == -1 {
		t.Fatalf("mongod unusual output:\n%v", string(out))
	}
	t.Log(string(out[:i]))
}

func runMongoBackground(t *testing.T) *exec.Cmd {
	t.Log("$ mongod --dbpath ./test/db --port 27019")
	mongo := exec.Command("mongod", "--dbpath", "./test/db", "--port", "27019")
	err := mongo.Start()
	if err != nil {
		t.Fatalf("mongod is required for this test.\n%v", err)
	}
	return mongo
}

func dialMongo(t *testing.T) {
	db, err := mgo.Dial("mongodb://127.0.0.1:27019/?connect=direct")
	if err != nil {
		t.Fatalf("can not connect to mongodb.\n%v", err)
	}

	api.DBSensorValues = db.DB("waziup").C("sensor_values")
	api.DBActuatorValues = db.DB("waziup").C("actuator_values")
	api.DBDevices = db.DB("waziup").C("devices")
}
