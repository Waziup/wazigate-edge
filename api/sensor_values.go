package api

import (
	"encoding/json"
	"net/http"

	"github.com/globalsign/mgo/bson"

	routing "github.com/julienschmidt/httprouter"
)

// GetDeviceSensorValue implements GET /devices/{deviceID}/sensors/{sensorID}/value
func GetDeviceSensorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getLastSensorValue(resp, params.ByName("device_id"), params.ByName("sensor_id"))
}

// GetSensorValue implements GET /sensors/{sensorID}/value
func GetSensorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getLastSensorValue(resp, GetLocalID(), params.ByName("sensor_id"))
}

// GetDeviceSensorValues implements GET /devices/{deviceID}/sensors/{sensorID}/values
func GetDeviceSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query Query
	if err := query.from(req); err != "" {
		http.Error(resp, "Bad Request - "+err, http.StatusBadRequest)
		return
	}
	getSensorValues(resp, params.ByName("device_id"), params.ByName("sensor_id"), &query)
}

// GetSensorValues implements GET /sensors/{sensorID}/values
func GetSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query Query
	if err := query.from(req); err != "" {
		http.Error(resp, "Bad Request - "+err, http.StatusBadRequest)
		return
	}
	getSensorValues(resp, GetLocalID(), params.ByName("sensor_id"), &query)
}

// PostDeviceSensorValue implements POST /devices/{deviceID}/sensors/{sensorID}/value
func PostDeviceSensorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValue(resp, req, params.ByName("device_id"), params.ByName("sensor_id"))
}

// PostDeviceSensorValues implements POST /devices/{deviceID}/sensors/{sensorID}/values
func PostDeviceSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValues(resp, req, params.ByName("device_id"), params.ByName("sensor_id"))
}

// PostSensorValue implements POST /sensors/{sensorID}/value
func PostSensorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValue(resp, req, GetLocalID(), params.ByName("sensor_id"))
}

// PostSensorValues implements POST /sensors/{sensorID}/values
func PostSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValues(resp, req, GetLocalID(), params.ByName("sensor_id"))
}

////////////////////

func getLastSensorValue(resp http.ResponseWriter, deviceID string, sensorID string) {

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var value SensorValue
	err := DBActuatorValues.Find(bson.M{"deviceID": deviceID, "sensorID": sensorID}).Sort("-_id").One(&value)
	if err != nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	data, _ := json.Marshal(value.Value)
	resp.Write(data)
}

func getSensorValues(resp http.ResponseWriter, deviceID string, sensorID string, query *Query) {

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var value SensorValue
	iter := DBActuatorValues.Find(bson.M{"deviceID": deviceID, "sensorID": sensorID}).Iter()
	serveIter(resp, iter, &value)
}

////////////////////

func postSensorValue(resp http.ResponseWriter, req *http.Request, deviceID string, sensorID string) {

	plainValue, err := getReqValue(req)
	if err != nil {
		http.Error(resp, "Bad Request - "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err = DBActuatorValues.Insert(&SensorValue{
		Value:    plainValue,
		DeviceID: deviceID,
		SensorID: sensorID,
	})

	if err != nil {
		http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func postSensorValues(resp http.ResponseWriter, req *http.Request, deviceID string, sensorID string) {

	plainValues, err := getReqValues(req)
	if err != nil {
		http.Error(resp, "Bad Request - "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	values := make([]SensorValue, len(plainValues))
	for i, plainValue := range plainValues {
		values[i].DeviceID = deviceID
		values[i].SensorID = sensorID
		values[i].Value = plainValue
	}

	interf := make([]interface{}, len(plainValues))
	for i := 0; i < len(plainValues); i++ {
		interf[i] = values[i]
	}

	err = DBActuatorValues.Insert(interf...)

	if err != nil {
		http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
		return
	}
}
