package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/globalsign/mgo/bson"
	routing "github.com/julienschmidt/httprouter"
)

// Sensor represents a Waziup sensor
type Sensor struct {
	ID    string      `json:"id" bson:"-"`
	Name  string      `json:"name" bson:"name"`
	Value interface{} `json:"value" bson:"value"`
}

// GetDeviceSensor implements GET /devices/{deviceID}/sensors/{sensorID}
func GetDeviceSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceSensor(resp, params.ByName("device_id"), params.ByName("sensor_id"))
}

// GetSensor implements GET /sensors/{sensorID}
func GetSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceSensor(resp, GetLocalID(), params.ByName("sensor_id"))
}

// GetDeviceSensors implements GET /devices/{deviceID}/sensors
func GetDeviceSensors(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceSensors(resp, params.ByName("device_id"))
}

// GetSensors implements GET /sensors
func GetSensors(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceSensors(resp, GetLocalID())
}

// PostDeviceSensor implements POST /devices/{deviceID}/sensors
func PostDeviceSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceSensor(resp, req, params.ByName("device_id"))
}

// PostSensor implements POST /sensors
func PostSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceSensor(resp, req, GetLocalID())
}

// DeleteDeviceSensor implements DELETE /devices/{deviceID}/sensors/{sensorID}
func DeleteDeviceSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	deleteDeviceSensor(resp, params.ByName("device_id"), params.ByName("sensor_id"))
}

// DeleteSensor implements DELETE /sensors/{sensorID}
func DeleteSensor(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	deleteDeviceSensor(resp, GetLocalID(), params.ByName("sensor_id"))
}

////////////////////

func getDeviceSensor(resp http.ResponseWriter, deviceID string, sensorID string) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	err := DBDevices.Find(bson.M{"_id": deviceID}).Select(bson.M{"sensors": sensorID}).One(&device)
	if err != nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	sensor := device.Sensors[sensorID]
	if sensor == nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	sensor.ID = sensorID
	data, _ := json.Marshal(sensor)
	resp.Write(data)
}

func getDeviceSensors(resp http.ResponseWriter, deviceID string) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	err := DBDevices.Find(bson.M{"_id": deviceID}).Select(bson.M{"sensors": 1}).One(&device)
	if err != nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	sensors := device.Sensors
	for id, sensor := range sensors {
		sensor.ID = id
	}
	data, _ := json.Marshal(sensors)
	resp.Write(data)
}

func postDeviceSensor(resp http.ResponseWriter, req *http.Request, deviceID string) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	sensor, err := getReqSensor(req)

	err = DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"sensors": bson.M{
			"$set": bson.M{
				sensor.ID: sensor,
			},
		},
	})
	if err != nil {
		http.Error(resp, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "text/plain")
	resp.Write([]byte(sensor.ID))
}

func deleteDeviceSensor(resp http.ResponseWriter, deviceID string, sensorID string) {

	if DBDevices == nil || DBSensorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err1 := DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"sensors": bson.M{
			"$unset": sensorID,
		},
	})
	err2 := DBSensorValues.Remove(bson.M{
		"deviceId": deviceID,
		"sensorId": sensorID,
	})

	if err1 != nil || err2 != nil {
		err := err1
		if err == nil {
			err = err2
		}
		http.Error(resp, "Database Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

////////////////////

func getReqSensor(req *http.Request) (*Sensor, error) {
	var sensor Sensor
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &sensor)
	if err != nil {
		return nil, err
	}
	if sensor.ID == "" {
		sensor.ID = bson.NewObjectId().String()
	}
	return &sensor, nil
}
