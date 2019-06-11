package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/Waziup/wazigate-edge/tools"
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
	Modified  time.Time   `json:"modified" bson:"modified"`
	Created   time.Time   `json:"created" bson:"created"`
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

// PostDeviceName implements POST /devices/{deviceID}/name
func PostDeviceName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceName(resp, req, params.ByName("device_id"))
}

// PostCurrentDeviceName implements POST /device/name
func PostCurrentDeviceName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postDeviceName(resp, req, GetLocalID())
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
		sensors := make([]interface{}, 0, len(device.Sensors))
		for _, sensor := range device.Sensors {
			if sensor.Value != nil {
				sensors = append(sensors, &SensorValue{
					ID:       newID(sensor.Time),
					DeviceID: device.ID,
					SensorID: sensor.ID,
					Value:    sensor.Value,
				})
			}
		}
		DBSensorValues.Insert(sensors...)
	}

	if len(device.Actuators) != 0 && DBActuatorValues != nil {
		actuators := make([]interface{}, 0, len(device.Actuators))
		for _, actuator := range device.Actuators {
			if actuator.Value != nil {
				actuators = append(actuators, &ActuatorValue{
					ID:         newID(actuator.Time),
					DeviceID:   device.ID,
					ActuatorID: actuator.ID,
					Value:      actuator.Value,
				})
			}
		}
		DBActuatorValues.Insert(actuators...)
	}

	log.Printf("[DB   ] created device %s\n", device.ID)

	CloudsMutex.RLock()
	for _, cloud := range Clouds {
		topic := "devices/" + device.ID + "/actuators/#"
		pkt := mqtt.Subscribe(0, []mqtt.TopicSubscription{
			mqtt.TopicSubscription{
				Name: topic,
			},
		})
		cloud.Queue.WritePacket(pkt)
	}
	CloudsMutex.RUnlock()

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'"'})
	resp.Write([]byte(device.ID))
	resp.Write([]byte{'"'})
}

////////////////////

func postDeviceName(resp http.ResponseWriter, req *http.Request, deviceID string) {
	body, err := tools.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}
	var name string
	err = json.Unmarshal(body, &name)
	if err != nil {
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	err = DBDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$set": bson.M{
			"modified": time.Now(),
			"name":     name,
		},
	})

	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(resp, "Device not found.", http.StatusNotFound)
			return
		}
		http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Write([]byte("true"))
}

////////////////////

func deleteDevice(resp http.ResponseWriter, deviceID string) {

	if DBDevices == nil || DBSensorValues == nil || DBActuatorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	devices := DBDevices.FindId(deviceID).Iter()
	for devices.Next(&device) {
		CloudsMutex.RLock()
		for _, cloud := range Clouds {
			topic := "devices/" + device.ID + "/actuators/#"
			pkt := mqtt.Unsubscribe(0, []string{topic})
			cloud.Queue.WritePacket(pkt)
		}
		CloudsMutex.RUnlock()
	}
	devices.Close()

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
