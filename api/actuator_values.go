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

// GetDeviceActuatorValue implements GET /devices/{deviceID}/actuators/{actuatorID}/value
func GetDeviceActuatorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getLastActuatorValue(resp, params.ByName("device_id"), params.ByName("actuator_id"))
}

// GetActuatorValue implements GET /actuators/{actuatorID}/value
func GetActuatorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getLastActuatorValue(resp, GetLocalID(), params.ByName("actuator_id"))
}

// GetDeviceActuatorValues implements GET /devices/{deviceID}/actuators/{actuatorID}/values
func GetDeviceActuatorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query edge.Query
	if err := query.Parse(req); err != "" {
		http.Error(resp, "bad request: "+err, http.StatusBadRequest)
		return
	}
	getActuatorValues(resp, params.ByName("device_id"), params.ByName("actuator_id"), &query)
}

// GetActuatorValues implements GET /actuators/{actuatorID}/values
func GetActuatorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query edge.Query
	if err := query.Parse(req); err != "" {
		http.Error(resp, "bad request: "+err, http.StatusBadRequest)
		return
	}
	getActuatorValues(resp, edge.LocalID(), params.ByName("actuator_id"), &query)
}

// PostDeviceActuatorValue implements POST /devices/{deviceID}/actuators/{actuatorID}/value
func PostDeviceActuatorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postActuatorValue(resp, req, params.ByName("device_id"), params.ByName("actuator_id"))
}

// PostDeviceActuatorValues implements POST /devices/{deviceID}/actuators/{actuatorID}/values
func PostDeviceActuatorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postActuatorValues(resp, req, params.ByName("device_id"), params.ByName("actuator_id"))
}

// PostActuatorValue implements POST /actuators/{actuatorID}/value
func PostActuatorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postActuatorValue(resp, req, edge.LocalID(), params.ByName("actuator_id"))
}

// PostActuatorValues implements POST /actuators/{actuatorID}/values
func PostActuatorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postActuatorValues(resp, req, edge.LocalID(), params.ByName("actuator_id"))
}

////////////////////

func getLastActuatorValue(resp http.ResponseWriter, deviceID string, actuatorID string) {

	actuator, err := edge.GetActuator(deviceID, actuatorID)
	if err != nil {
		serveError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(actuator.Value)
	resp.Write(data)
}

func getActuatorValues(resp http.ResponseWriter, deviceID string, actuatorID string, query *edge.Query) {

	values := edge.GetActuatorValues(deviceID, actuatorID, query)
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

func postActuatorValue(resp http.ResponseWriter, req *http.Request, deviceID string, actuatorID string) {

	val, err := getReqValue(req)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = edge.PostActuatorValue(deviceID, actuatorID, val)
	if err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[DB   ] 1 value for %s/%s.\n", deviceID, actuatorID)

	clouds.FlagActuator(deviceID, actuatorID, val.Time)
}

func postActuatorValues(resp http.ResponseWriter, req *http.Request, deviceID string, actuatorID string) {

	vals, err := getReqValues(req)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(vals) != 0 {
		err = edge.PostActuatorValues(deviceID, actuatorID, vals)
		if err != nil {
			serveError(resp, err)
			return
		}

		clouds.FlagActuator(deviceID, actuatorID, vals[0].Time)
	}

	log.Printf("[DB   ] %d values for %s/%s.\n", len(vals), deviceID, actuatorID)
}
