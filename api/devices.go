package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/tools"
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
	Modified  time.Time   `json:"modified" bson:"modified"`
	Created   time.Time   `json:"created" bson:"created"`
}

////////////////////

// GetDevices implements GET /devices
func GetDevices(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	devices := edge.GetDevices()
	encoder := json.NewEncoder(resp)

	device, err := devices.Next()
	if err != nil {
		serveError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'['})
	for device != nil {
		encoder.Encode(device)
		device, _ = devices.Next()
		if device != nil {
			resp.Write([]byte{','})
		}
	}
	resp.Write([]byte{']'})
}

func serveError(resp http.ResponseWriter, err error) {

	if codeErr, ok := err.(edge.CodeError); ok {
		http.Error(resp, codeErr.Text, codeErr.Code)
		return
	}

	http.Error(resp, "internal server error", 500)
}

// GetDevice implements GET /devices/{deviceID}
func GetDevice(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDevice(resp, params.ByName("device_id"))
}

// GetCurrentDevice implements GET /device
func GetCurrentDevice(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDevice(resp, edge.LocalID())
}

// GetCurrentDeviceID implements GET /device/id
func GetCurrentDeviceID(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	resp.Header().Set("Content-Type", "text/plain")
	resp.Write([]byte(edge.LocalID()))
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

	deleteDevice(resp, edge.LocalID())
}

// PostDeviceName implements POST /devices/{deviceID}/name
func PostDeviceName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceName(resp, req, params.ByName("device_id"))
}

// PostCurrentDeviceName implements POST /device/name
func PostCurrentDeviceName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceName(resp, req, edge.LocalID())
}

////////////////////

func getDevice(resp http.ResponseWriter, deviceID string) {

	device, err := edge.GetDevice(deviceID)
	if err != nil {
		serveError(resp, err)
		return
	}
	if device == nil {
		resp.WriteHeader(404)
		resp.Write([]byte("not found"))
		return
	}
	encoder := json.NewEncoder(resp)
	resp.Header().Set("Content-Type", "application/json")
	encoder.Encode(device)
}

////////////////////

func postDevice(resp http.ResponseWriter, req *http.Request) {

	var device edge.Device
	if err := getReqDevice(req, &device); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := edge.PostDevice(&device); err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[DB   ] Created device %q\n", device.ID)

	clouds.Flag(device.ID, "", noTime)

	resp.Write([]byte(device.ID))
}

////////////////////

func postDeviceName(resp http.ResponseWriter, req *http.Request, deviceID string) {
	body, err := tools.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	var name string
	contentType := req.Header.Get("Content-Type")
	if strings.HasSuffix(contentType, "application/json") {
		err = json.Unmarshal(body, &name)
		if err != nil {
			http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		name = string(body)
	}

	if err := edge.SetDeviceName(deviceID, name); err != nil {
		serveError(resp, err)
	}
}

////////////////////

func deleteDevice(resp http.ResponseWriter, deviceID string) {

	_, numS, numA, err := edge.DeleteDevice(deviceID)
	if err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[DB   ] removed device %s (%d sensor values, %d actuator values)\n", deviceID, numS, numA)
}

////////////////////

func getReqDevice(req *http.Request, device *edge.Device) error {
	body, err := tools.ReadAll(req.Body)
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
	var noTime time.Time
	now := time.Now()
	if device.Modified == noTime {
		device.Modified = now
	}

	if device.Sensors != nil {
		for _, sensor := range device.Sensors {
			if sensor.Created == noTime {
				sensor.Created = now
			}
			if sensor.Modified == noTime {
				sensor.Modified = now
			}
			if sensor.Time == noTime {
				sensor.Time = now
			}
		}
	}
	if device.Actuators != nil {
		for _, actuator := range device.Actuators {
			if actuator.Created == noTime {
				actuator.Created = now
			}
			if actuator.Modified == noTime {
				actuator.Modified = now
			}
			if actuator.Time == noTime {
				actuator.Time = now
			}
		}
	}
	return nil
}
