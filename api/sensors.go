package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/tools"
	"github.com/globalsign/mgo/bson"
	routing "github.com/julienschmidt/httprouter"
)

// Sensor represents a Waziup sensor
type Sensor struct {
	ID       string      `json:"id" bson:"id"`
	Name     string      `json:"name" bson:"name"`
	Modified time.Time   `json:"modified" bson:"modified"`
	Created  time.Time   `json:"created" bson:"created"`
	Time     time.Time   `json:"time" bson:"time"`
	Value    interface{} `json:"value" bson:"value"`
}

// GetDeviceSensor implements GET /devices/{deviceID}/sensors/{sensorID}
func GetDeviceSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceSensor(resp, params.ByName("device_id"), params.ByName("sensor_id"))
}

// GetSensor implements GET /sensors/{sensorID}
func GetSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceSensor(resp, edge.LocalID(), params.ByName("sensor_id"))
}

// GetDeviceSensors implements GET /devices/{deviceID}/sensors
func GetDeviceSensors(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceSensors(resp, params.ByName("device_id"))
}

// GetSensors implements GET /sensors
func GetSensors(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceSensors(resp, edge.LocalID())
}

// PostDeviceSensor implements POST /devices/{deviceID}/sensors
func PostDeviceSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceSensor(resp, req, params.ByName("device_id"))
}

// PostSensor implements POST /sensors
func PostSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceSensor(resp, req, edge.LocalID())
}

// DeleteDeviceSensor implements DELETE /devices/{deviceID}/sensors/{sensorID}
func DeleteDeviceSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	deleteDeviceSensor(resp, params.ByName("device_id"), params.ByName("sensor_id"))
}

// DeleteSensor implements DELETE /sensors/{sensorID}
func DeleteSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	deleteDeviceSensor(resp, edge.LocalID(), params.ByName("sensor_id"))
}

// PostDeviceSensorName implements POST /devices/{deviceID}/sensors/{sensorID}/name
func PostDeviceSensorName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceSensorName(resp, req, params.ByName("device_id"), params.ByName("sensor_id"))
}

// PostSensorName implements POST /sensors/{sensorID}/name
func PostSensorName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceSensorName(resp, req, edge.LocalID(), params.ByName("sensor_id"))
}

////////////////////

func getDeviceSensor(resp http.ResponseWriter, deviceID string, sensorID string) {

	sensor, err := edge.GetSensor(deviceID, sensorID)
	if err != nil {
		serveError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(sensor)
	resp.Write(data)
}

func getDeviceSensors(resp http.ResponseWriter, deviceID string) {

	device, err := edge.GetDevice(deviceID)
	if err != nil {
		serveError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(device.Sensors)
	resp.Write(data)
}

func postDeviceSensor(resp http.ResponseWriter, req *http.Request, deviceID string) {

	var sensor edge.Sensor
	if err := getReqSensor(req, &sensor); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := edge.PostSensor(deviceID, &sensor); err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[DB   ] Sensor %s/%s created.\n", deviceID, sensor.ID)
	clouds.FlagSensor(deviceID, sensor.ID, noTime)

	resp.Write([]byte(sensor.ID))
}

func postDeviceSensorName(resp http.ResponseWriter, req *http.Request, deviceID string, sensorID string) {

	body, err := tools.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	var name string
	contentType := req.Header.Get("Content-Type")
	if strings.HasSuffix(contentType, "application/json") {
		err = json.Unmarshal(body, &name)
		if err != nil {
			http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		name = string(body)
	}

	err = edge.SetSensorName(deviceID, sensorID, name)
	if err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[DB   ] Sensor %s/%s name changed: %q", deviceID, sensorID, name)
}

func deleteDeviceSensor(resp http.ResponseWriter, deviceID string, sensorID string) {

	points, err := edge.DeleteSensor(deviceID, sensorID)
	if err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[DB   ] Sensor %s/%s removed. (%d values)\n", deviceID, sensorID, points)
}

////////////////////

func getReqSensor(req *http.Request, sensor *edge.Sensor) error {
	body, err := tools.ReadAll(req.Body)
	if err != nil {
		return err
	}

	now := time.Now()
	sensor.Time = now
	sensor.Modified = now
	sensor.Created = now

	err = json.Unmarshal(body, &sensor)
	if err != nil {
		return err
	}

	if sensor.ID == "" {
		sensor.ID = bson.NewObjectId().Hex()
	}
	return nil
}
