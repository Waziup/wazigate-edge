package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/globalsign/mgo/bson"
	routing "github.com/julienschmidt/httprouter"
)

// Actuator represents a Waziup actuator
type Actuator struct {
	ID    string      `json:"id" bson:"-"`
	Name  string      `json:"name" bson:"name"`
	Value interface{} `json:"value" bson:"value"`
}

// GetDeviceActuator implements GET /devices/{deviceID}/actuators/{actuatorID}
func GetDeviceActuator(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceActuator(resp, params.ByName("device_id"), params.ByName("actuator_id"))
}

// GetActuator implements GET /actuators/{actuatorID}
func GetActuator(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceActuator(resp, GetLocalID(), params.ByName("actuator_id"))
}

// GetDeviceActuators implements GET /devices/{deviceID}/actuators
func GetDeviceActuators(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceActuators(resp, params.ByName("device_id"))
}

// GetActuators implements GET /actuators
func GetActuators(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceActuators(resp, GetLocalID())
}

// PostDeviceActuator implements POST /devices/{deviceID}/actuators
func PostDeviceActuator(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceActuator(resp, req, params.ByName("device_id"))
}

// PostActuator implements POST /actuators
func PostActuator(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceActuator(resp, req, GetLocalID())
}

// DeleteDeviceActuator implements DELETE /devices/{deviceID}/actuators/{actuatorID}
func DeleteDeviceActuator(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	deleteDeviceActuator(resp, params.ByName("device_id"), params.ByName("actuator_id"))
}

// DeleteActuator implements DELETE /actuators/{actuatorID}
func DeleteActuator(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	deleteDeviceActuator(resp, GetLocalID(), params.ByName("actuator_id"))
}

////////////////////

func getDeviceActuator(resp http.ResponseWriter, deviceID string, actuatorID string) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	err := DBDevices.Find(bson.M{"_id": deviceID}).Select(bson.M{"actuators": actuatorID}).One(&device)
	if err != nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	actuator := device.Actuators[actuatorID]
	if actuator == nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	data, _ := json.Marshal(actuator)
	resp.Write(data)
}

func getDeviceActuators(resp http.ResponseWriter, deviceID string) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	err := DBDevices.Find(bson.M{"_id": deviceID}).Select(bson.M{"actuators": 1}).One(&device)
	if err != nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	actuators := device.Actuators
	data, _ := json.Marshal(actuators)
	resp.Write(data)
}

func postDeviceActuator(resp http.ResponseWriter, req *http.Request, deviceID string) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	actuator, err := getReqActuator(req)

	err = DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"actuators": bson.M{
			"$set": bson.M{
				actuator.ID: actuator,
			},
		},
	})
	if err != nil {
		http.Error(resp, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "text/plain")
	resp.Write([]byte(actuator.ID))
}

func deleteDeviceActuator(resp http.ResponseWriter, deviceID string, actuatorID string) {

	if DBDevices == nil || DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err1 := DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"actuators": bson.M{
			"$unset": actuatorID,
		},
	})
	err2 := DBActuatorValues.Remove(bson.M{
		"deviceId":   deviceID,
		"actuatorId": actuatorID,
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

func getReqActuator(req *http.Request) (*Actuator, error) {
	var actuator Actuator
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &actuator)
	if err != nil {
		return nil, err
	}
	if actuator.ID == "" {
		actuator.ID = bson.NewObjectId().String()
	}
	return &actuator, nil
}
