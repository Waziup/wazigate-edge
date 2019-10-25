package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/Waziup/wazigate-edge/edge"

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

	var query edge.Query
	if err := query.Parse(req); err != "" {
		http.Error(resp, "bad request: "+err, http.StatusBadRequest)
		return
	}
	getSensorValues(resp, params.ByName("device_id"), params.ByName("sensor_id"), &query)
}

// GetSensorValues implements GET /sensors/{sensorID}/values
func GetSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query edge.Query
	if err := query.Parse(req); err != "" {
		http.Error(resp, "bad request: "+err, http.StatusBadRequest)
		return
	}
	getSensorValues(resp, edge.LocalID(), params.ByName("sensor_id"), &query)
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

	postSensorValue(resp, req, edge.LocalID(), params.ByName("sensor_id"))
}

// PostSensorValues implements POST /sensors/{sensorID}/values
func PostSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValues(resp, req, edge.LocalID(), params.ByName("sensor_id"))
}

////////////////////

func getLastSensorValue(resp http.ResponseWriter, deviceID string, sensorID string) {

	sensor, err := edge.GetSensor(deviceID, sensorID)
	if err != nil {
		serveError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(sensor.Value)
	resp.Write(data)
}

func getSensorValues(resp http.ResponseWriter, deviceID string, sensorID string, query *edge.Query) {

	values := edge.GetSensorValues(deviceID, sensorID, query)
	encoder := json.NewEncoder(resp)

	value, err := values.Next()
	if err != nil && err != io.EOF {
		serveError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'['})
	for err == nil {
		encoder.Encode(value)
		value, err = values.Next()
		if err == nil {
			resp.Write([]byte{','})
		}
	}
	resp.Write([]byte{']'})
}

////////////////////

func postSensorValue(resp http.ResponseWriter, req *http.Request, deviceID string, sensorID string) {

	val, err := getReqValue(req)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	meta, err := edge.PostSensorValue(deviceID, sensorID, val)
	if err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[DB   ] 1 value for %s/%s.\n", deviceID, sensorID)

	clouds.FlagSensor(deviceID, sensorID, clouds.ActionSync, val.Time, meta)
}

func postSensorValues(resp http.ResponseWriter, req *http.Request, deviceID string, sensorID string) {

	vals, err := getReqValues(req)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(vals) != 0 {
		meta, err := edge.PostSensorValues(deviceID, sensorID, vals)
		if err != nil {
			serveError(resp, err)
			return
		}

		clouds.FlagSensor(deviceID, sensorID, clouds.ActionSync, vals[0].Time, meta)
	}

	log.Printf("[DB   ] %d values for %s/%s.\n", len(vals), deviceID, sensorID)
}
