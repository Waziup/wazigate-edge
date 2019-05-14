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

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	err := DBDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(bson.M{
		"actuators": bson.M{
			"$elemMatch": bson.M{
				"id": actuatorID,
			},
		},
	}).One(&device)

	if err != nil || len(device.Actuators) == 0 {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}

	value := device.Actuators[0].Value
	data, _ := json.Marshal(value)
	resp.Write(data)
}

func getActuatorValues(resp http.ResponseWriter, deviceID string, actuatorID string, query *Query) {

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var value ActuatorValue
	iter := DBActuatorValues.Find(bson.M{"deviceId": deviceID, "actuatorId": actuatorID}).Iter()
	serveIter(resp, iter, &value)
}

////////////////////

func postActuatorValue(resp http.ResponseWriter, req *http.Request, deviceID string, actuatorID string) {

	val, err := getReqValue(req)
	if err != nil {
		http.Error(resp, "Bad Request - "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	value := ActuatorValue{
		ID:         newID(val.Time),
		Value:      val.Value,
		DeviceID:   deviceID,
		ActuatorID: actuatorID,
	}
	err = DBActuatorValues.Insert(&value)

	if err != nil {
		http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Write([]byte{'"'})
	resp.Write([]byte(value.ID.Hex()))
	resp.Write([]byte{'"'})
}

func postActuatorValues(resp http.ResponseWriter, req *http.Request, deviceID string, actuatorID string) {

	vals, err := getReqValues(req)
	if err != nil {
		http.Error(resp, "Bad Request - "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	values := make([]ActuatorValue, len(vals))
	interf := make([]interface{}, len(vals))

	for i, v := range vals {
		values[i] = ActuatorValue{
			ID:         newID(v.Time),
			DeviceID:   deviceID,
			ActuatorID: actuatorID,
			Value:      v.Value,
		}
		interf[i] = values[i]
	}

	err = DBActuatorValues.Insert(interf...)

	if err != nil {
		http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Write([]byte("true"))
}
