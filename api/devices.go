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

	var query edge.Query
	query.Parse(req.URL.Query())
	devices := edge.GetDevices(&query)
	encoder := json.NewEncoder(resp)

	device, err := devices.Next()
	if err != nil && err.Error() != "EOF" {
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

	// resp.Header().Set("Content-Type", "text/plain")
	// resp.Write([]byte(edge.LocalID()))

	tools.SendJSON(resp, edge.LocalID())
}

// GetCurrentDeviceName implements GET /device/name
func GetCurrentDeviceName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceName(resp, req, edge.LocalID())
}

// GetCurrentDeviceMeta implements GET /device/meta
func GetCurrentDeviceMeta(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceMeta(resp, req, edge.LocalID())
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

// GetDeviceName implements GET /devices/{deviceID}/name
func GetDeviceName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceName(resp, req, params.ByName("device_id"))
}

// PostDeviceName implements POST /devices/{deviceID}/name
func PostDeviceName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceName(resp, req, params.ByName("device_id"))
}

// PostCurrentDeviceName implements POST /device/name
func PostCurrentDeviceName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceName(resp, req, edge.LocalID())
}

/*---------------------------------*/

// PostCurrentDeviceID implements POST /device/id
func PostCurrentDeviceID(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	body, err := tools.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var newID string
	contentType := req.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		err = json.Unmarshal(body, &newID)
		if err != nil {
			http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		newID = string(body)
	}

	err = edge.SetDeviceID(newID)
	if err != nil {
		log.Printf("[Err  ] SetDeviceID: %s", err.Error())
		serveError(resp, err)
		return
	}
}

/*---------------------------------*/

// PostDeviceMeta implements POST /devices/{deviceID}/meta
func PostDeviceMeta(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceMeta(resp, req, params.ByName("device_id"))
}

// GetDeviceMeta implements GET /devices/{deviceID}/meta
func GetDeviceMeta(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getDeviceMeta(resp, req, params.ByName("device_id"))
}

// PostCurrentDeviceMeta implements POST /device/meta
func PostCurrentDeviceMeta(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceMeta(resp, req, edge.LocalID())
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

	log.Printf("[DB   ] Created device %s.", device.ID)

	clouds.FlagDevice(device.ID, clouds.ActionCreate, device.Meta)

	resp.Write([]byte(device.ID))
}

////////////////////

func getDeviceName(resp http.ResponseWriter, req *http.Request, deviceID string) {
	name, err := edge.GetDeviceName(deviceID)
	if err != nil {
		serveError(resp, err)
		return
	}
	resp.Header().Set("Content-Type", "text/plain")
	// resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte(name))
}

func postDeviceName(resp http.ResponseWriter, req *http.Request, deviceID string) {
	body, err := tools.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	var name string
	contentType := req.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		err = json.Unmarshal(body, &name)
		if err != nil {
			http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		name = string(body)
	}

	meta, err := edge.SetDeviceName(deviceID, name)
	if err != nil {
		serveError(resp, err)
		return
	}

	clouds.FlagDevice(deviceID, clouds.ActionModify, meta)
}

func getDeviceMeta(resp http.ResponseWriter, req *http.Request, deviceID string) {
	meta, err := edge.GetDeviceMeta(deviceID)
	if err != nil {
		serveError(resp, err)
		return
	}
	encoder := json.NewEncoder(resp)
	resp.Header().Set("Content-Type", "application/json")
	encoder.Encode(meta)
}

func postDeviceMeta(resp http.ResponseWriter, req *http.Request, deviceID string) {
	body, err := tools.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	var meta edge.Meta
	err = json.Unmarshal(body, &meta)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = edge.SetDeviceMeta(deviceID, meta)
	if err != nil {
		serveError(resp, err)
		return
	}

	clouds.FlagDevice(deviceID, clouds.ActionModify, meta)
}

////////////////////

func deleteDevice(resp http.ResponseWriter, deviceID string) {

	_, numS, numA, err := edge.DeleteDevice(deviceID)
	if err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[DB   ] Removed device %s (%d sensor values, %d actuator values).\n", deviceID, numS, numA)
	clouds.FlagDevice(deviceID, clouds.ActionDelete, nil)
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
	return nil
}
