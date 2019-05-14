package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	routing "github.com/julienschmidt/httprouter"
)

////////////////////

// Device represents a Waziup Device
type Device struct {
	Name      string      `json:"name" bson:"name"`
	ID        string      `json:"id" bson:"_id"`
	Sensors   []*Sensor   `json:"sensors" bson:"sensors"`
	Actuators []*Actuator `json:"actuators" bson:"actuators"`
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
	var err error

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	if err = getReqDevice(req, &device); err != nil {
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = DBDevices.Insert(&device)
	if err != nil {
		http.Error(resp, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(device.Sensors) != 0 && DBSensorValues != nil {
		sensors := make([]interface{}, len(device.Sensors))
		for i, sensor := range device.Sensors {
			sensors[i] = &SensorValue{
				ID:       newID(sensor.Time),
				DeviceID: device.ID,
				SensorID: sensor.ID,
				Value:    sensor.Value,
			}
		}
		DBSensorValues.Insert(sensors...)
	}

	log.Printf("[DB   ] created device %s\n", device.ID)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'"'})
	resp.Write([]byte(device.ID))
	resp.Write([]byte{'"'})
}

////////////////////

func deleteDevice(resp http.ResponseWriter, deviceID string) {

	if DBDevices == nil || DBSensorValues == nil || DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err := DBDevices.RemoveId(deviceID)
	info1, _ := DBSensorValues.RemoveAll(bson.M{"deviceId": deviceID})
	info2, _ := DBActuatorValues.RemoveAll(bson.M{"deviceId": deviceID})
	log.Printf("[DB   ] removed device %s\n", deviceID)
	log.Printf("[DB   ] removed %d values from %s/* sensors\n", info1.Removed, deviceID)
	log.Printf("[DB   ] removed %d values from %s/* actuators\n", info2.Removed, deviceID)

	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(resp, "null", http.StatusNotFound)
			return
		}

		http.Error(resp, "Database Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Write([]byte("null"))
}

////////////////////

func getReqDevice(req *http.Request, device *Device) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &device)
	if err != nil {
		return err
	}
	if device.ID == "" {
		device.ID = bson.NewObjectId().Hex()
	}
	now := time.Now()
	var noTime time.Time
	if device.Sensors != nil {
		for _, sensor := range device.Sensors {
			if sensor.Time == noTime {
				sensor.Time = now
			}
		}
	}
	if device.Actuators != nil {
		for _, sensor := range device.Actuators {
			if sensor.Time == noTime {
				sensor.Time = now
			}
		}
	}
	return nil
}
