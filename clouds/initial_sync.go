package clouds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Waziup/wazigate-edge/edge"
)

var noTime = time.Time{}

func (cloud *Cloud) initialSync() bool {
	cloud.setStatus(0, "Connecting to server for initial sync...")

	// 1.
	// Get Auth token from /auth

	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		cloud.Credentials.Username,
		cloud.Credentials.Token,
	}

	addr := cloud.getRESTAddr()
	log.Printf("[UP   ] Dialing REST %q...", addr)

	body, _ := json.Marshal(credentials)
	resp := fetch(addr+"/auth/token", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
		body: bytes.NewReader(body),
	})
	if !resp.ok {
		cloud.setStatus(resp.status, fmt.Sprintf("Unable to connect.\n%s", resp.statusText))
		return false
	}

	token := resp.text()
	if len(token) == 0 {
		cloud.setStatus(0, "Unable to connect.\nRecieved invalid token.")
		log.Printf("[UP   ] Err Token %s", token)
		return false
	}
	cloud.auth = "Bearer " + token
	log.Println("[UP   ] Authentication successfull.")

	// 2.
	// Call /gateways

	log.Printf("[UP   ] Pushing gateway to the cloud ...")

	localDevice, err := edge.GetDevice(edge.LocalID())
	if err != nil {
		cloud.setStatus(0, "Internal Error.\nCan not get local device.")
		log.Printf("[UP   ] Err %s", err.Error())
		return false
	}

	var gateway = v2Gateway{
		ID:         localDevice.ID,
		Name:       localDevice.Name,
		Visibility: "public",
	}

	body, _ = json.Marshal(gateway)
	resp = fetch(addr+"/gateways", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": cloud.auth,
		},
		body: bytes.NewReader(body),
	})
	if resp.ok {
		log.Printf("[UP   ] Gateway pushed.")
	} else {
		log.Printf("[UP   ] Err Can not register gateway: %s (%s)", resp.statusText, resp.text())
	}

	// 3.
	// Get all devices from this gateway and compare them with the cloud

	// log.Printf("[UP   ] Registering devices at the cloud ...")

	devices := edge.GetDevices()
	for device, err := devices.Next(); err == nil; device, err = devices.Next() {
		log.Printf("[UP   ] Checking device %q ...", device.ID)

		resp = fetch(addr+"/devices/"+device.ID, fetchInit{
			method: http.MethodGet,
			headers: map[string]string{
				"Authorization": cloud.auth,
			},
		})
		switch resp.status {
		case http.StatusNotFound:
			log.Printf("[UP   ] Device %q not found.", device.ID)
			cloud.remote[entity{device.ID, ""}] = &remote{noTime, false}
			// if !cloud.postDevice(device) {
			// 	return false
			// }
		case http.StatusOK:
			var device2 v2Device
			if err := resp.json(&device2); err != nil {
				cloud.setStatus(0, fmt.Sprintf("Communication Error.\nCan not unmarshal response: %s", err.Error()))
				return false
			}

		SENSORS:
			for _, sensor := range device.Sensors {
				if sensor.Value == nil {
					sensor.Time = noTime
				}
				for _, s := range device2.Sensors {
					if s.ID == sensor.ID {
						if s.Value != nil {
							if s.Value.Time == noTime {
								s.Value.Time = s.Value.TimeReceived
							}
							if !s.Value.Time.Before(sensor.Time) {
								log.Printf("[UP   ] Sensor %q up do date.", sensor.ID)
							} else {
								log.Printf("[UP   ] Sensor %q outdated! Last value %v.", sensor.ID, s.Value.Time)
								cloud.remote[entity{device.ID, sensor.ID}] = &remote{s.Value.Time, true}
							}
						} else {
							if sensor.Value != nil {
								log.Printf("[UP   ] Sensor %q outdated! No values.", sensor.ID)
								cloud.remote[entity{device.ID, sensor.ID}] = &remote{noTime, true}
							}
							log.Printf("[UP   ] Sensor %q up do date. No values.", sensor.ID)
						}
						continue SENSORS
					}
				}
				log.Printf("[UP   ] Sensor %q does not exist.", sensor.ID)
				cloud.remote[entity{device.ID, sensor.ID}] = &remote{noTime, false}
				// if !cloud.postSensor(device.ID, sensor) {
				// 	return false
				// }
				// log.Printf("[UP   ] Sensor pushed.")
			}

		default:
			cloud.setStatus(0, fmt.Sprintf("Communication Error.\nResponse %d %s: %s", resp.status, resp.statusText, resp.text()))
			return false
		}
	}

	return true
}
