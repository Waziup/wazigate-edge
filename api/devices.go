package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/globalsign/mgo/bson"
	routing "github.com/julienschmidt/httprouter"
)

////////////////////

// Device represents a Waziup Device
type Device struct {
	Name      string               `json:"name" bson:"name"`
	ID        string               `json:"id" bson:"_id"`
	Sensors   map[string]*Sensor   `json:"sensors" bson:"sensors"`
	Actuators map[string]*Actuator `json:"actuators" bson:"actuators"`
}

////////////////////

// GetDevices implements GET /devices
func GetDevices(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	resp.Header().Set("Content-Type", "application/json")
	iter := DBDevices.Find(nil).Iter()
	serveIter(resp, iter, &device)
}

// GetDevice implements GET /devices/{deviceID}
func GetDevice(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDevice(resp, params.ByName("device_id"))
}

// GetCurrentDevice implements GET /device
func GetCurrentDevice(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDevice(resp, GetLocalID())
}

// PostDevice implements POST /devices
func PostDevice(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDevice(resp, req)
}

// DeleteDevice implements DELETE /devices/{deviceID}
func DeleteDevice(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	deleteDevice(resp, params.ByName("device_id"))
}

// DeleteCurrentDevice implements DELETE /device
func DeleteCurrentDevice(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	deleteDevice(resp, GetLocalID())
}

////////////////////

func getDevice(resp http.ResponseWriter, deviceID string) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	query := DBDevices.FindId(deviceID)
	var device Device
	resp.Header().Set("Content-Type", "application/json")
	if err := query.One(&device); err != nil {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}
	data, _ := json.Marshal(&device)
	resp.Write(data)
}

////////////////////

func postDevice(resp http.ResponseWriter, req *http.Request) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	device, err := getReqDevice(req)

	err = DBDevices.Insert(device)
	if err != nil {
		http.Error(resp, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "text/plain")
	resp.Write([]byte(device.ID))
}

////////////////////

func deleteDevice(resp http.ResponseWriter, deviceID string) {

	if DBDevices == nil || DBSensorValues == nil || DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err1 := DBDevices.RemoveId(deviceID)
	err2 := DBSensorValues.Remove(bson.M{"deviceId": deviceID})
	err3 := DBActuatorValues.Remove(bson.M{"deviceId": deviceID})

	if err1 != nil || err2 != nil || err3 != nil {
		err := err1
		if err == nil {
			err = err2
		}
		if err == nil {
			err = err3
		}
		http.Error(resp, "Database Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

////////////////////

func getReqDevice(req *http.Request) (*Device, error) {
	var device Device
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &device)
	if err != nil {
		return nil, err
	}
	if device.ID == "" {
		device.ID = bson.NewObjectId().String()
	}
	return &device, nil
}
