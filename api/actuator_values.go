package api

import (
	"encoding/json"
	"net/http"

	"github.com/globalsign/mgo/bson"

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

	var query Query
	if err := query.from(req); err != "" {
		http.Error(resp, "Bad Request - "+err, http.StatusBadRequest)
		return
	}
	getActuatorValues(resp, params.ByName("device_id"), params.ByName("actuator_id"), &query)
}

// GetActuatorValues implements GET /actuators/{actuatorID}/values
func GetActuatorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query Query
	if err := query.from(req); err != "" {
		http.Error(resp, "Bad Request - "+err, http.StatusBadRequest)
		return
	}
	getActuatorValues(resp, GetLocalID(), params.ByName("actuator_id"), &query)
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

	postActuatorValue(resp, req, GetLocalID(), params.ByName("actuator_id"))
}

// PostActuatorValues implements POST /actuators/{actuatorID}/values
func PostActuatorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postActuatorValues(resp, req, GetLocalID(), params.ByName("actuator_id"))
}

////////////////////

func getLastActuatorValue(resp http.ResponseWriter, deviceID string, actuatorID string) {

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var value ActuatorValue
	err := DBActuatorValues.Find(bson.M{"deviceID": deviceID, "actuatorID": actuatorID}).Sort("-_id").One(&value)
	if err != nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	data, _ := json.Marshal(value.Value)
	resp.Write(data)
}

func getActuatorValues(resp http.ResponseWriter, deviceID string, actuatorID string, query *Query) {

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var value ActuatorValue
	iter := DBActuatorValues.Find(bson.M{"deviceID": deviceID, "actuatorID": actuatorID}).Iter()
	serveIter(resp, iter, &value)
}

////////////////////

func postActuatorValue(resp http.ResponseWriter, req *http.Request, deviceID string, actuatorID string) {

	plainValue, err := getReqValue(req)
	if err != nil {
		http.Error(resp, "Bad Request - "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err = DBActuatorValues.Insert(&ActuatorValue{
		Value:      plainValue,
		DeviceID:   deviceID,
		ActuatorID: actuatorID,
	})

	if err != nil {
		http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func postActuatorValues(resp http.ResponseWriter, req *http.Request, deviceID string, actuatorID string) {

	plainValues, err := getReqValues(req)
	if err != nil {
		http.Error(resp, "Bad Request - "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	values := make([]ActuatorValue, len(plainValues))
	for i, plainValue := range plainValues {
		values[i].DeviceID = deviceID
		values[i].ActuatorID = actuatorID
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
