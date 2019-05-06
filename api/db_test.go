package api

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/globalsign/mgo"
)

func TestDatabase(t *testing.T) {
	cleanup := initTests(t)
	cleanup()
}

func initTests(t *testing.T) func() {

	if testing.Short() {
		t.Skip("skipping full test in short mode.")
	}

	if DBSensorValues == nil || DBActuatorValues == nil || DBDevices == nil {

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

	DBSensorValues = db.DB("waziup").C("sensor_values")
	DBActuatorValues = db.DB("waziup").C("actuator_values")
	DBDevices = db.DB("waziup").C("devices")
}
