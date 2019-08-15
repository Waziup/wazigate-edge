package clouds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/edge"
)

func (cloud *Cloud) nextSensor() (entity, *remote) {

	for true {
		cloud.remoteMutex.Lock()
		if len(cloud.remote) != 0 {
			cloud.remoteMutex.Unlock()
			break
		}
		cloud.remoteMutex.Unlock()
		cloud.setStatus(0, "Queue drained. Cloud is up-to-date.")
		<-cloud.sigDirty
	}

	var nextEntity entity
	var rem *remote

	cloud.remoteMutex.Lock()

	// loop all dirty sensors/devices to see which one to sync next
	for entity, r := range cloud.remote {
		// unexisting devices have the highest priority
		if entity.sensorID == "" && entity.actuatorID == "" {
			nextEntity = entity
			rem = r
			break
		}
		// unexisting sensors & actuators have the bext priority
		if !r.exists {
			nextEntity = entity
			rem = r
			continue
		}
		if rem == nil {
			nextEntity = entity
			rem = r
			continue
		}
		// oldest values first
		if r.time.Before(rem.time) {
			nextEntity = entity
			rem = r
		}
	}

	cloud.remoteMutex.Unlock()

	return nextEntity, rem
}

func (cloud *Cloud) persistentSync() int {

	ent, rem := cloud.nextSensor()
	status := cloud.processEntity(ent, rem)
	switch status {
	case http.StatusBadRequest, http.StatusNotFound:
		log.Printf("[UP   ] Entity removed from sync queue due to an error.")
		cloud.remoteMutex.Lock()
		delete(cloud.remote, ent)
		cloud.remoteMutex.Unlock()
		return 0
	}
	return status
}

////////////////////////////////////////////////////////////////////////////////

func (cloud *Cloud) processEntity(ent entity, rem *remote) (status int) {

	if ent.sensorID == "" && ent.actuatorID == "" {
		// sync an unexisting device
		log.Printf("[UP   ] Pushing device %s ...", ent.deviceID)
		device, err := edge.GetDevice(ent.deviceID)
		if err != nil {
			log.Printf("[Err  ] %s", err.Error())
			return -1
		}
		status = cloud.postDevice(device)

		if isOk(status) {
			cloud.remoteMutex.Lock()
			delete(cloud.remote, ent)
			log.Printf("[UP   ] Device pushed.")
			for _, sensor := range device.Sensors {
				if sensor.Value != nil {
					cloud.remote[entity{device.ID, sensor.ID, ""}] = &remote{noTime, true}
				}
			}
			for _, actuator := range device.Actuators {
				if actuator.Value != nil {
					cloud.remote[entity{device.ID, "", actuator.ID}] = &remote{noTime, true}
				}
			}
			cloud.remoteMutex.Unlock()
		}
		return

	}

	if !rem.exists {

		if ent.sensorID != "" {
			// sync an unexisting sensor
			log.Printf("[UP   ] Pushing sensor %s/%s ...", ent.deviceID, ent.sensorID)
			sensor, err := edge.GetSensor(ent.deviceID, ent.sensorID)
			if err != nil {
				log.Printf("[Err   ] Err %s", err.Error())
				return -1
			}
			status = cloud.postSensor(ent.deviceID, sensor)
			if isOk(status) {
				rem.exists = true
				log.Printf("[UP   ] Sensor pushed.")
			}
		} else {
			// sync an unexisting actuator
			log.Printf("[UP   ] Pushing actuator %s/%s ...", ent.deviceID, ent.actuatorID)
			actuator, err := edge.GetActuator(ent.deviceID, ent.actuatorID)
			if err != nil {
				log.Printf("[Err   ] Err %s", err.Error())
				return -1
			}
			status = cloud.postActuator(ent.deviceID, actuator)
			if isOk(status) {
				rem.exists = true
				log.Printf("[UP   ] Actuator pushed.")
			}
		}
		return
	}

	log.Printf("[UP   ] Pushing values %s/%s ...", ent.deviceID, ent.sensorID)

	query := &edge.Query{
		From:  rem.time,
		Size:  1024 * 1024,
		Limit: 30000,
	}
	values := edge.GetSensorValues(ent.deviceID, ent.sensorID, query)
	status, numVal, lastTime := cloud.postValues(ent.deviceID, ent.sensorID, values)
	if isOk(status) {
		if numVal == 0 {
			log.Printf("[UP   ] Values are now up-to-date.")
			cloud.remoteMutex.Lock()
			delete(cloud.remote, ent)
			cloud.remoteMutex.Unlock()
			return
		}

		log.Printf("[UP   ] Pushed %d values until %s.", numVal, lastTime.UTC())
		rem.time = lastTime
		return
	}
	return
}

func (cloud *Cloud) postDevice(device *edge.Device) int {
	var syncDev v2Device
	syncDev.ID = device.ID
	syncDev.Name = device.Name
	syncDev.Sensors = make([]v2Sensor, len(device.Sensors))
	for i, sensor := range device.Sensors {
		syncDev.Sensors[i].ID = sensor.ID
		syncDev.Sensors[i].Name = sensor.Name
	}
	syncDev.Actuators = make([]v2Actuator, len(device.Actuators))
	for i, actuator := range device.Actuators {
		syncDev.Actuators[i].ID = actuator.ID
		syncDev.Actuators[i].Name = actuator.Name
	}

	addr := cloud.getRESTAddr()

	body, _ := json.Marshal(syncDev)
	resp := fetch(addr+"/devices", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": cloud.auth,
		},
		body: bytes.NewReader(body),
	})
	if !resp.ok {
		cloud.setStatus(resp.status, fmt.Sprintf("Unable to push device. %s\n%s", resp.statusText, strings.TrimSpace(resp.text())))
	}
	return resp.status
}

func (cloud *Cloud) postSensor(deviceID string, sensor *edge.Sensor) int {
	var syncSensor v2Sensor
	syncSensor.ID = sensor.ID
	syncSensor.Name = sensor.Name

	addr := cloud.getRESTAddr()

	body, _ := json.Marshal(syncSensor)
	resp := fetch(addr+"/devices/"+deviceID+"/sensors", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": cloud.auth,
		},
		body: bytes.NewReader(body),
	})
	if !resp.ok {
		cloud.setStatus(resp.status, fmt.Sprintf("Unable to push sensor. %s\n%s", resp.statusText, strings.TrimSpace(resp.text())))
	}
	return resp.status
}

func (cloud *Cloud) postActuator(deviceID string, actuator *edge.Actuator) int {
	var syncActuator v2Actuator
	syncActuator.ID = actuator.ID
	syncActuator.Name = actuator.Name

	addr := cloud.getRESTAddr()

	body, _ := json.Marshal(syncActuator)
	resp := fetch(addr+"/devices/"+deviceID+"/actuators", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": cloud.auth,
		},
		body: bytes.NewReader(body),
	})
	if !resp.ok {
		cloud.setStatus(resp.status, fmt.Sprintf("Unable to push actuator. %s\n%s", resp.statusText, strings.TrimSpace(resp.text())))
	}
	return resp.status
}

func (cloud *Cloud) postValues(deviceID string, sensorID string, values edge.ValueIterator) (int, int, time.Time) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	value, err := values.Next()
	if err == io.EOF {
		return http.StatusNoContent, 0, noTime
	}

	var lastTime time.Time
	var value2 struct {
		Value interface{} `json:"value"`
		Time  time.Time   `json:"timestamp"`
	}

	numValues := 0
	for ; err == nil; value, err = values.Next() {
		numValues++
		lastTime = value.Time
		value2.Time = value.Time
		value2.Value = value.Value
		encoder.Encode(value2)
	}

	addr := cloud.getRESTAddr()

	resp := fetch(addr+"/devices/"+deviceID+"/sensors/"+sensorID+"/values", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": cloud.auth,
		},
		body: &buf,
	})
	if !resp.ok {
		cloud.setStatus(resp.status, fmt.Sprintf("Unable to push values. %s\n%s", resp.statusText, strings.TrimSpace(resp.text())))
		return resp.status, -1, lastTime
	}
	return resp.status, numValues, lastTime
}
