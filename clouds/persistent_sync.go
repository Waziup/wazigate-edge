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

		<-cloud.sigDirty
	}

	var nextEntity entity
	var nextSensor *remote

	cloud.remoteMutex.Lock()

	// loop all dirty sensors/devices to see which one to sync next
	for entity, rem := range cloud.remote {
		// unexisting devices have the highest priority
		if entity.sensorID == "" {
			nextEntity = entity
			nextSensor = rem
			break
		}
		// unexisting sensors have the bext priority
		if !rem.exists {
			nextEntity = entity
			nextSensor = rem
			continue
		}
		if nextSensor == nil {
			nextEntity = entity
			nextSensor = rem
			continue
		}
		// oldest sensors first
		if rem.time.Before(nextSensor.time) {
			nextEntity = entity
			nextSensor = rem
		}
	}

	cloud.remoteMutex.Unlock()

	return nextEntity, nextSensor
}

func (cloud *Cloud) persistentSync() bool {

	for !cloud.Paused {

		nextEntity, nextSensor := cloud.nextSensor()

		// sync it!
		if nextEntity.sensorID == "" {

			// sync an unexisting device
			log.Printf("[UP   ] Pushing device %q ...", nextEntity.deviceID)
			device, err := edge.GetDevice(nextEntity.deviceID)
			if err != nil {
				log.Printf("[UP   ] Err %s", err.Error())
				continue
			}
			ok := cloud.postDevice(device)

			cloud.remoteMutex.Lock()
			delete(cloud.remote, nextEntity)
			if !ok {
				cloud.remoteMutex.Unlock()
				return false
			}
			for _, sensor := range device.Sensors {
				if sensor.Value != nil {
					cloud.remote[entity{device.ID, sensor.ID}] = &remote{noTime, true}
				}
			}
			cloud.remoteMutex.Unlock()

			log.Printf("[UP   ] Device pushed.")
			continue

		} else if !nextSensor.exists {

			// sync an unexisting sensor
			log.Printf("[UP   ] Pushing sensor %q/%q ...", nextEntity.deviceID, nextEntity.sensorID)
			sensor, err := edge.GetSensor(nextEntity.deviceID, nextEntity.sensorID)
			if err != nil {
				log.Printf("[UP   ] Err %s", err.Error())
				cloud.remoteMutex.Lock()
				delete(cloud.remote, nextEntity)
				cloud.remoteMutex.Unlock()
				log.Printf("[UP   ] Sync for this entity has been stopped due to an error.")
				continue
			}
			if !cloud.postSensor(nextEntity.deviceID, sensor) {
				cloud.remoteMutex.Lock()
				delete(cloud.remote, nextEntity)
				cloud.remoteMutex.Unlock()
				log.Printf("[UP   ] Sync for this entity has been stopped due to an error.")
				return false
			}
			nextSensor.exists = true
			log.Printf("[UP   ] Sensor pushed.")
			continue

		} else {
			log.Printf("[UP   ] Pushing values %q/%q ...", nextEntity.deviceID, nextEntity.sensorID)

			query := &edge.Query{
				From:  nextSensor.time,
				Size:  1024 * 1024,
				Limit: 30000,
			}
			values := edge.GetSensorValues(nextEntity.deviceID, nextEntity.sensorID, query)
			numVal, lastTime := cloud.postValues(nextEntity.deviceID, nextEntity.sensorID, values)
			if numVal == -1 {
				cloud.remoteMutex.Lock()
				delete(cloud.remote, nextEntity)
				log.Printf("[UP   ] Sync for this entity has been stopped due to an error.")
				cloud.remoteMutex.Unlock()
				return false
			}
			if numVal == 0 {
				log.Printf("[UP   ] Values are now up-to-date.")
				cloud.remoteMutex.Lock()
				delete(cloud.remote, nextEntity)
				cloud.remoteMutex.Unlock()
				continue
			}
			log.Printf("[UP   ] Pushed %d values until %s.", numVal, lastTime.UTC())
			nextSensor.time = lastTime
			continue
		}
	}

	return true
}

////////////////////////////////////////////////////////////////////////////////

func (cloud *Cloud) postDevice(device *edge.Device) bool {
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
	return resp.ok
}

func (cloud *Cloud) postSensor(deviceID string, sensor *edge.Sensor) bool {
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
	return resp.ok
}

func (cloud *Cloud) postValues(deviceID string, sensorID string, values edge.ValueIterator) (int, time.Time) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	value, err := values.Next()
	if err == io.EOF {
		return 0, noTime
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
		return -1, lastTime
	}
	return numValues, lastTime
}
