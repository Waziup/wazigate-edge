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

func (cloud *Cloud) authenticate() int {

	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		cloud.Credentials.Username,
		cloud.Credentials.Token,
	}

	addr := cloud.getRESTAddr()
	log.Printf("[UP   ] Authentication ...", addr)

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
		return resp.status
	}

	token := resp.text()
	if len(token) == 0 {
		cloud.setStatus(0, "Unable to connect.\nRecieved invalid token.")
		log.Printf("[UP   ] Err Token %s", token)
		return 0
	}
	cloud.auth = "Bearer " + token
	log.Println("[UP   ] Authentication successfull.")

	return resp.status
}

func (cloud *Cloud) initialSync() int {

	// Call /gateways

	addr := cloud.getRESTAddr()

	log.Printf("[UP   ] Pushing gateway to the cloud ...")

	localDevice, err := edge.GetDevice(edge.LocalID())
	if err != nil {
		cloud.setStatus(0, "Internal Error.\nCan not get local device.")
		log.Printf("[Err  ] %s", err.Error())
		return -1
	}

	var gateway = v2Gateway{
		ID:         localDevice.ID,
		Name:       localDevice.Name,
		Visibility: "public",
	}

	body, _ := json.Marshal(gateway)
	resp := fetch(addr+"/gateways", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": cloud.auth,
		},
		body: bytes.NewReader(body),
	})
	switch resp.status {
	case http.StatusOK:
		log.Printf("[UP   ] Gateway pushed.")
	case http.StatusUnprocessableEntity:
		log.Printf("[UP   ] Gateway already pushed.")
	default:
		return resp.status
	}

	// Get all devices from this gateway and compare them with the cloud

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
			cloud.remote[entity{device.ID, "", ""}] = &remote{noTime, false}

		case http.StatusOK:
			var device2 v2Device
			if err := resp.json(&device2); err != nil {
				cloud.setStatus(0, fmt.Sprintf("Communication Error.\nCan not unmarshal response: %s", err.Error()))
				return -1
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
								cloud.remote[entity{device.ID, sensor.ID, ""}] = &remote{s.Value.Time, true}
							}
						} else {
							if sensor.Value != nil {
								log.Printf("[UP   ] Sensor %q outdated! No values.", sensor.ID)
								cloud.remote[entity{device.ID, sensor.ID, ""}] = &remote{noTime, true}
							}
							log.Printf("[UP   ] Sensor %q up do date. No values.", sensor.ID)
						}
						continue SENSORS
					}
				}
				log.Printf("[UP   ] Sensor %q does not exist.", sensor.ID)
				cloud.remote[entity{device.ID, sensor.ID, ""}] = &remote{noTime, false}
			}

		ACTUATORS:
			for _, acuator := range device.Actuators {
				if acuator.Value == nil {
					acuator.Time = noTime
				}
				for _, s := range device2.Actuators {
					if s.ID == acuator.ID {
						if s.Value != nil {
							if s.Value.Time == noTime {
								s.Value.Time = s.Value.TimeReceived
							}
							if !s.Value.Time.Before(acuator.Time) {
								log.Printf("[UP   ] Actuator %q up do date.", acuator.ID)
							} else {
								log.Printf("[UP   ] Actuator %q outdated! Last value %v.", acuator.ID, s.Value.Time)
								cloud.remote[entity{device.ID, "", acuator.ID}] = &remote{s.Value.Time, true}
							}
						} else {
							if acuator.Value != nil {
								log.Printf("[UP   ] Actuator %q outdated! No values.", acuator.ID)
								cloud.remote[entity{device.ID, "", acuator.ID}] = &remote{noTime, true}
							}
							log.Printf("[UP   ] Actuator %q up do date. No values.", acuator.ID)
						}
						continue ACTUATORS
					}
				}
				log.Printf("[UP   ] Actuator %q does not exist.", acuator.ID)
				cloud.remote[entity{device.ID, "", acuator.ID}] = &remote{noTime, false}
			}

		default:
			cloud.setStatus(resp.status, fmt.Sprintf("Err [%d] %s: %s", resp.status, resp.statusText, resp.text()))
			return resp.status
		}
	}

	return http.StatusOK
}
