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

// if A is less prioritized then B to sync
// returns false if they are equal
func less(entA *Entity, statusA *Status, entB *Entity, statusB *Status) bool {

	if entB.Sensor == "" && entB.Actuator == "" {
		// it's a device
		if entA.Sensor != "" || entB.Actuator != "" {
			// devices have a higher priority then sensors and actuators
			return true
		}
		// highest action wins (create > modify > sync)
		return statusA.Action < statusB.Action
	}

	if statusA.Action != statusB.Action {
		// highest action wins
		return statusA.Action < statusB.Action
	}

	if statusB.Wakeup.Before(statusA.Wakeup) {
		// first one to wakeup wins
		return true
	}

	return false
}

func (cloud *Cloud) nextEntity() (Entity, *Status, time.Time) {

	var entity Entity
	var status *Status

	now := time.Now()
	wakeup := now.Add(time.Hour)

	cloud.StatusMutex.Lock()

	// loop all dirty sensors/devices to see which one to sync next
	for e, s := range cloud.Status {

		if s.Action&ActionError != 0 {
			// we do not sync entities that have an error
			continue
		}

		if s.Wakeup.After(now) {
			// we do not sync entities that are sleeping now
			if s.Wakeup.Before(wakeup) {
				wakeup = s.Wakeup
			}
			continue
		}

		if s.Action == 0 {
			// a no-action entity? delete it!
			delete(cloud.Status, e)
			continue
		}

		if status == nil {
			entity = e
			status = s
			continue
		}

		// unexisting entities have the highest priority
		if s.Action&ActionCreate != 0 {

			if e.Sensor == "" && e.Actuator == "" {
				// unexisting devices
				entity = e
				status = s
				break
			}

			// unexisting sensor or actuator
			entity = e
			status = s
			continue
		}

		if status == nil {
			entity = e
			status = s
			continue
		}

		// both entities can sync now -> the one with the oldest unsynced values makes the race
		if s.Wakeup.Before(now) && status.Wakeup.Before(now) && s.Remote.Before(status.Remote) {
			entity = e
			status = s
			continue
		}

		// the one that synces next makes the race
		if s.Wakeup.Before(status.Wakeup) {
			entity = e
			status = s
			continue
		}
	}

	if status != nil {
		s := &Status{} // duplicate the status to
		*s = *status   // avoid race contidtion
		status = s     // if we leave statusMutex now
	}

	cloud.StatusMutex.Unlock()

	return entity, status, wakeup
}

func (cloud *Cloud) persistentSync() (int, error) {

	ent, status, wakeup := cloud.nextEntity()
	for status == nil && !cloud.Pausing {
		now := time.Now()
		timer := time.NewTimer(wakeup.Sub(now))
		select {
		case ent := <-cloud.sigDirty:
			log.Printf("[UP   ] Wakeup on flagged entity %q.", ent)
		case <-timer.C:
			log.Printf("[UP   ] Wakeup on timer.")
		}
		ent, status, wakeup = cloud.nextEntity()
	}
	if cloud.Pausing {
		return 0, nil
	}

	code, err := cloud.processEntity(ent, status)
	isRetry := false
PROCESS:
	if err != nil {
		switch {
		case code == 0:
			cloud.Printf("Network Error\n%s", code, err.Error())
		case code == -1:
			cloud.Printf("Internal Error\n%s", code, err.Error())
			cloud.flag(ent, ActionError, noTime, nil)
		case code == http.StatusUnauthorized || code == http.StatusForbidden:
			// on permission error: re-authenticate (new token) and try again
			if !isRetry && isOk(cloud.authenticate()) {
				isRetry = true
				code, err = cloud.processEntity(ent, status)
				goto PROCESS
			}
			cloud.Printf("Permission Error\n%s", code, err.Error())
			cloud.flag(ent, ActionError, noTime, nil)
		case code >= 500 && code < 600:
			cloud.Printf("Server Error %d\n%s", code, code, err.Error())
			cloud.flag(ent, ActionError, noTime, nil)

			// cloud.statusMutex.Lock()
			// delete(cloud.Status, ent)
			// cloud.statusMutex.Unlock()
		case code >= 400 && code < 500:
			cloud.Printf("Synchronization Error %d\n%s", code, code, err.Error())

			cloud.flag(ent, ActionError, noTime, nil)

			// cloud.statusMutex.Lock()
			// delete(cloud.Status, ent)
			// cloud.statusMutex.Unlock()
		}
	}
	return code, err
}

////////////////////////////////////////////////////////////////////////////////

func (cloud *Cloud) processEntity(ent Entity, status *Status) (int, error) {

	if status.Action&ActionDelete != 0 {
		cloud.flag(ent, -ActionDelete, noTime, nil)
		return 204, nil
	}

	if status.Action&ActionCreate != 0 {

		if ent.Sensor == "" && ent.Actuator == "" {
			// sync an unexisting device
			// log.Printf("[UP   ] Pushing device %s ...", ent.Device)
			device, err := edge.GetDevice(ent.Device)
			if err != nil {
				return -1, fmt.Errorf("Internal Error\n%s", err.Error())
			}
			code, err := cloud.postDevice(device)
			if err == nil {
				cloud.flag(Entity{device.ID, "", ""}, -ActionCreate, noTime, nil)
				// cloud.statusMutex.Lock()
				// delete(cloud.Status, ent)
				// log.Printf("[UP   ] Device pushed.")
				for _, sensor := range device.Sensors {
					if sensor.Value != nil {
						cloud.flag(Entity{device.ID, sensor.ID, ""}, ActionSync, noTime, nil)
					}
				}
				for _, actuator := range device.Actuators {
					if actuator.Value != nil {
						cloud.flag(Entity{device.ID, "", actuator.ID}, ActionSync, noTime, nil)
					}
				}
				// cloud.statusMutex.Unlock()
			}
			return code, err
		}

		if ent.Sensor != "" {
			// sync an unexisting sensor
			// log.Printf("[UP   ] Pushing sensor %s/%s ...", ent.Device, ent.Sensor)
			sensor, err := edge.GetSensor(ent.Device, ent.Sensor)
			if err != nil {
				return -1, fmt.Errorf("Internal Error\n%s", err.Error())
			}
			code, err := cloud.postSensor(ent.Device, sensor)
			if err == nil {
				cloud.flag(ent, -ActionCreate, noTime, nil)
				// log.Printf("[UP   ] Sensor pushed.")
			}
			return code, err
		}

		// sync an unexisting actuator
		log.Printf("[UP   ] Pushing actuator %s/%s ...", ent.Device, ent.Actuator)
		actuator, err := edge.GetActuator(ent.Device, ent.Actuator)
		if err != nil {
			return -1, fmt.Errorf("Internal Error\n%s", err.Error())
		}
		code, err := cloud.postActuator(ent.Device, actuator)
		if err == nil {
			cloud.flag(ent, -ActionCreate, noTime, nil)
			// log.Printf("[UP   ] Actuator pushed successfull.")
		}
		return code, err
	}

	if status.Action&ActionModify != 0 {

		if ent.Sensor == "" && ent.Actuator == "" {
			name, err := edge.GetDeviceName(ent.Device)
			if err != nil {
				return -1, fmt.Errorf("Internal Error\n%s", err.Error())
			}
			code, err := cloud.postDeviceName(ent.Device, name)
			if err == nil {
				cloud.flag(ent, -ActionModify, noTime, nil)
			}
			return code, err
		}

		if ent.Sensor != "" {
			sensor, err := edge.GetSensor(ent.Device, ent.Sensor)
			if err != nil {
				return -1, fmt.Errorf("Internal Error\n%s", err.Error())
			}
			code, err := cloud.postSensorName(ent.Device, ent.Sensor, sensor.Name)
			if err == nil {
				cloud.flag(ent, -ActionModify, noTime, nil)
			}
			return code, err
		}

		actuator, err := edge.GetActuator(ent.Device, ent.Actuator)
		if err != nil {
			return -1, fmt.Errorf("Internal Error\n%s", err.Error())
		}
		code, err := cloud.postActuatorName(ent.Device, ent.Actuator, actuator.Name)
		if err == nil {
			cloud.flag(ent, -ActionModify, noTime, nil)
		}
		return code, err
	}

	if status.Action&ActionSync != 0 {
		// log.Printf("[UP   ] Pushing values %s/%s ...", ent.Device, ent.Sensor)

		query := &edge.ValuesQuery{
			From:  status.Remote,
			Size:  1024 * 1024,
			Limit: 30000,
		}
		values := edge.GetSensorValues(ent.Device, ent.Sensor, query)

		remote, n, code, err := cloud.postValues(ent.Device, ent.Sensor, values)
		if err == nil {
			if n == 0 {
				// log.Printf("[UP   ] Values are now up-to-date.")
				cloud.flag(ent, -ActionSync, noTime, nil)
			} else {
				// log.Printf("[UP   ] Pushed %d values successfull until %s.", n, remote.UTC())
				cloud.flag(ent, 0, remote.Add(time.Second), nil)
			}
		}
		return code, err
	}

	return 0, nil
}

func (cloud *Cloud) postDevice(device *edge.Device) (int, error) {
	var syncDev v2Device
	syncDev.ID = device.ID
	syncDev.Name = device.Name
	syncDev.Gateway = edge.LocalID()
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
		err := fmt.Errorf("Unable to push device.\nStatus: %s\n%s", resp.statusText, strings.TrimSpace(resp.text()))
		return resp.status, err
	}
	return resp.status, nil
}

func (cloud *Cloud) postDeviceName(deviceID string, name string) (int, error) {

	addr := cloud.getRESTAddr()

	resp := fetch(addr+"/devices/"+deviceID+"/name", fetchInit{
		method: http.MethodPut,
		headers: map[string]string{
			"Content-Type":  "text/plain; charset=utf-8",
			"Authorization": cloud.auth,
		},
		body: bytes.NewReader([]byte(name)),
	})

	if !resp.ok {
		err := fmt.Errorf("Unable to change device name.\nStatus: %s\n%s", resp.statusText, strings.TrimSpace(resp.text()))
		return resp.status, err
	}

	if deviceID == edge.LocalID() {
		resp := fetch(addr+"/gateways/"+deviceID+"/name", fetchInit{
			method: http.MethodPut,
			headers: map[string]string{
				"Content-Type":  "text/plain; charset=utf-8",
				"Authorization": cloud.auth,
			},
			body: bytes.NewReader([]byte(name)),
		})

		if !resp.ok {
			err := fmt.Errorf("Unable to change device name.\nStatus: %s\n%s", resp.statusText, strings.TrimSpace(resp.text()))
			return resp.status, err
		}
	}

	return resp.status, nil
}

func (cloud *Cloud) postSensor(deviceID string, sensor *edge.Sensor) (int, error) {
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
		err := fmt.Errorf("Unable to push sensor.\nStatus: %s\n%s", resp.statusText, strings.TrimSpace(resp.text()))
		return resp.status, err
	}
	return resp.status, nil
}

func (cloud *Cloud) postSensorName(deviceID string, sensorID string, name string) (int, error) {

	addr := cloud.getRESTAddr()

	resp := fetch(addr+"/devices/"+deviceID+"/sensors/"+sensorID+"/name", fetchInit{
		method: http.MethodPut,
		headers: map[string]string{
			"Content-Type":  "text/plain; charset=utf-8",
			"Authorization": cloud.auth,
		},
		body: bytes.NewReader([]byte(name)),
	})

	if !resp.ok {
		err := fmt.Errorf("Unable to change sensor name.\nStatus: %s\n%s", resp.statusText, strings.TrimSpace(resp.text()))
		return resp.status, err
	}
	return resp.status, nil
}

func (cloud *Cloud) postActuator(deviceID string, actuator *edge.Actuator) (int, error) {
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
		err := fmt.Errorf("Unable to push actuator.\nStatus: %s\n%s", resp.statusText, strings.TrimSpace(resp.text()))
		return resp.status, err
	}
	return resp.status, nil
}

func (cloud *Cloud) postActuatorName(deviceID string, actuatorID string, name string) (int, error) {

	addr := cloud.getRESTAddr()

	resp := fetch(addr+"/devic/"+deviceID+"/actuators/"+actuatorID+"/name", fetchInit{
		method: http.MethodPut,
		headers: map[string]string{
			"Content-Type":  "text/plain; charset=utf-8",
			"Authorization": cloud.auth,
		},
		body: bytes.NewReader([]byte(name)),
	})

	if !resp.ok {
		err := fmt.Errorf("Unable to change actuator name.\nStatus: %s\n%s", resp.statusText, strings.TrimSpace(resp.text()))
		return resp.status, err
	}
	return resp.status, nil
}

func (cloud *Cloud) postValues(deviceID string, sensorID string, values edge.ValueIterator) (time.Time, int, int, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	value, err := values.Next()
	if err == io.EOF {
		return noTime, 0, 204, nil
	}

	var value2 struct {
		Value interface{} `json:"value"`
		Time  time.Time   `json:"timestamp"`
	}

	buf.Write([]byte{'['})

	n := 0
	var remote time.Time
	for ; err == nil; value, err = values.Next() {
		if n != 0 {
			buf.Write([]byte{','})
		}
		value2.Time = value.Time
		remote = value.Time
		value2.Value = value.Value
		encoder.Encode(value2)
		n++
	}

	buf.Write([]byte{']'})

	addr := cloud.getRESTAddr()

	resp := fetch(addr+"/devices/"+deviceID+"/sensors/"+sensorID+"/values", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type":  "application/json; charset=UTF-8",
			"Authorization": cloud.auth,
		},
		body: &buf,
	})
	if !resp.ok {
		err := fmt.Errorf("Unable to push values.\nStatus: %s\n%s", resp.statusText, strings.TrimSpace(resp.text()))
		return noTime, 0, resp.status, err
	}
	return remote, n, resp.status, nil
}
