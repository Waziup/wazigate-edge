package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/globalsign/mgo/bson"
	routing "github.com/julienschmidt/httprouter"
)

// Actuator represents a Waziup actuator
type Actuator struct {
	ID    string      `json:"id" bson:"id"`
	Name  string      `json:"name" bson:"name"`
	Time  time.Time   `json:"time" bson:"time"`
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

	actuator := device.Actuators[0]
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

	data, _ := json.Marshal(device.Actuators)
	resp.Write(data)
}

func postDeviceActuator(resp http.ResponseWriter, req *http.Request, deviceID string) {
	var err error

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var actuator Actuator
	if err = getReqActuator(req, &actuator); err != nil {
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$push": bson.M{
			"actuators": &actuator,
		},
	})
	if err != nil {
		http.Error(resp, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DB   ] created actuator %s/%s\n", deviceID, actuator.ID)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'"'})
	resp.Write([]byte(actuator.ID))
	resp.Write([]byte{'"'})
}

func deleteDeviceActuator(resp http.ResponseWriter, deviceID string, actuatorID string) {

	if DBDevices == nil || DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err1 := DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$pull": bson.M{
			"actuators": bson.M{
				"id": actuatorID,
			},
		},
	})
	info, err2 := DBActuatorValues.RemoveAll(bson.M{
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

	log.Printf("[DB   ] removed actuator %s/%s\n", deviceID, actuatorID)
	log.Printf("[DB   ] removed %d values from %s/%s\n", info.Removed, deviceID, actuatorID)

	resp.Write([]byte("true"))
}

////////////////////

func getReqActuator(req *http.Request, actuator *Actuator) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	actuator.Time = time.Now()
	err = json.Unmarshal(body, &actuator)
	if err != nil {
		return err
	}
	if actuator.ID == "" {
		actuator.ID = bson.NewObjectId().String()
	}
	return nil
}
