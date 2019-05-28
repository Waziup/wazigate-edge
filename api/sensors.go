package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Waziup/waziup-edge/tools"
	"github.com/globalsign/mgo/bson"
	routing "github.com/julienschmidt/httprouter"
)

// Sensor represents a Waziup sensor
type Sensor struct {
	ID    string      `json:"id" bson:"id"`
	Name  string      `json:"name" bson:"name"`
	Time  time.Time   `json:"time" bson:"time"`
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
	err := DBDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(bson.M{
		"sensors": bson.M{
			"$elemMatch": bson.M{
				"id": sensorID,
			},
		},
	}).One(&device)

	if err != nil || len(device.Sensors) == 0 {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}

	sensor := device.Sensors[0]
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

	data, _ := json.Marshal(device.Sensors)
	resp.Write(data)
}

func postDeviceSensor(resp http.ResponseWriter, req *http.Request, deviceID string) {
	var err error

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var sensor Sensor
	if err = getReqSensor(req, &sensor); err != nil {
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$push": bson.M{
			"sensors": &sensor,
		},
	})
	if err != nil {
		http.Error(resp, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DB   ] created sensor %s/%s\n", deviceID, sensor.ID)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'"'})
	resp.Write([]byte(sensor.ID))
	resp.Write([]byte{'"'})
}

func deleteDeviceSensor(resp http.ResponseWriter, deviceID string, sensorID string) {

	if DBDevices == nil || DBSensorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err1 := DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$pull": bson.M{
			"sensors": bson.M{
				"id": sensorID,
			},
		},
	})
	info, err2 := DBSensorValues.RemoveAll(bson.M{
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

	log.Printf("[DB   ] removed sensor %s/%s\n", deviceID, sensorID)
	log.Printf("[DB   ] removed %d values from %s/%s\n", info.Removed, deviceID, sensorID)

	resp.Write([]byte("true"))
}

////////////////////

func getReqSensor(req *http.Request, sensor *Sensor) error {
	body, err := tools.ReadAll(req.Body)
	if err != nil {
		return err
	}
	sensor.Time = time.Now()
	err = json.Unmarshal(body, &sensor)
	if err != nil {
		return err
	}
	if sensor.ID == "" {
		sensor.ID = bson.NewObjectId().String()
	}
	return nil
}
