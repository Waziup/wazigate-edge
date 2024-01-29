package clouds

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/edge"
)

var noTime = time.Time{}

func (cloud *Cloud) authenticate() int {

	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		cloud.Username,
		cloud.Token,
	}

	addr := cloud.getRESTAddr()
	// log.Printf("[UP   ] Authentication as %q ...", cloud.Username)

	body, _ := json.Marshal(credentials)
	resp := fetch(addr+"/auth/token", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
		body: bytes.NewReader(body),
	})
	defer resp.Close()

	if !resp.ok {
		if resp.status <= 0 {
			cloud.Printf("Can not connect to server.\n%s", resp.status, resp.statusText)
		} else {
			cloud.Printf("Authentication failed.\n%s", resp.status, resp.statusText)
		}
		// cloud.setStatus(resp.status, fmt.Sprintf("Unable to connect.\n%s", resp.statusText))
		return resp.status
	}

	token := resp.text()
	if len(token) == 0 {
		cloud.Printf("Authentication failed.\nReceived invalid token.", 0)
		// log.Printf("[UP   ] Err Token %s", token)
		return 0
	}
	cloud.auth = "Bearer " + token
	cloud.Printf("Authentication successfull.", 200)
	// log.Println("[UP   ] Authentication successfull.")

	return resp.status
}

func (cloud *Cloud) initialSync() int {

	var resp fetchResponse

	// Call /gateways

	addr := cloud.getRESTAddr()

	localDevice, err := edge.GetDevice(edge.LocalID())
	if err != nil {
		cloud.Printf("Internal Error\nCan not get local device.\n%s", -1, err.Error())
		// cloud.setStatus(0, "Internal Error.\nCan not get local device.")
		// log.Printf("[Err  ] %s", err.Error())
	} else {

		var gateway = v2Gateway{
			ID:         localDevice.ID,
			Name:       localDevice.Name,
			Visibility: "public",
		}

		log.Printf("[UP   ] Pushing gateway %q to the cloud ...", localDevice.ID)

		body, _ := json.Marshal(gateway)
		resp := fetch(addr+"/gateways", fetchInit{
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type":  "application/json; charset=utf-8",
				"Authorization": cloud.auth,
			},
			body: bytes.NewReader(body),
		})
		defer resp.Close()

		if resp.status == http.StatusUnprocessableEntity {
			log.Printf("[UP   ] Gateway already registered.")
			// cloud.Printf("Gateway already registered.", 200)
		} else {
			if !resp.ok {
				cloud.Printf("Can not register gateway.\nStatus: %s\n%s", resp.status, resp.statusText, strings.TrimSpace(resp.text()))
				return resp.status
			}
			cloud.Printf("Gateway successfully registered.", resp.status)
		}

		cloud.Registered = true
	}

	cloud.mqttMutex.Lock()
	cloud.devices = make(map[string]struct{})
	cloud.mqttMutex.Unlock()

	// Get all devices from this gateway and compare them with the cloud
	devices := edge.GetDevices(nil)
	for device, err := devices.Next(); err == nil; device, err = devices.Next() {
		// log.Printf("[UP   ] Checking device %q ...", device.ID)

		cloud.IncludeDevice(device.ID)
		meta := device.Meta
		if meta.DoNotSync() {
			continue
		}

		resp = fetch(addr+"/devices/"+v2IdCompat(device.ID), fetchInit{
			method: http.MethodGet,
			headers: map[string]string{
				"Authorization": cloud.auth,
			},
		})

		switch resp.status {
		case http.StatusNotFound:
			// log.Printf("[UP   ] Device %q not found.", device.ID)
			cloud.flag(Entity{device.ID, "", ""}, ActionCreate, noTime, meta)
			resp.Close()

		case http.StatusOK:
			var device2 v2Device
			if err := resp.json(&device2); err != nil {
				cloud.Printf("Communication Error.\nCan not unmarshal response: %s", 500, err.Error())
				return 500
			}

		SENSORS:
			for _, sensor := range device.Sensors {
				meta := sensor.Meta
				if meta.DoNotSync() {
					continue
				}
				// if sensor.Value == nil {
				// 	sensor.Time = noTime
				// }
				for _, s := range device2.Sensors {
					if s.ID == v2IdCompat(sensor.ID) {
						if sensor.Time != nil {
							if s.Value == nil {
								cloud.flag(Entity{device.ID, sensor.ID, ""}, ActionSync, noTime, meta)
								// log.Printf("[UP   ] Sensor %q outdated! No time.", sensor.ID)
							} else if s.Value.Time.Add(time.Second).Before(*sensor.Time) {
								cloud.flag(Entity{device.ID, sensor.ID, ""}, ActionSync, s.Value.Time, meta)
								// log.Printf("[UP   ] Sensor %q outdated! Last value %v (latest: %v).", sensor.ID, s.Value.Time, sensor.Time)
							} else {
								// log.Printf("[UP   ] Sensor %q up do date.", sensor.ID)
							}
						}
						continue SENSORS
					}
				}
				// log.Printf("[UP   ] Sensor %q does not exist.", sensor.ID)
				cloud.flag(Entity{device.ID, sensor.ID, ""}, ActionCreate, noTime, meta)
				// cloud.Status[Entity{device.ID, sensor.ID, ""}] = NewStatus(ActionCreate, noTime)
			}

		ACTUATORS:
			for _, acuator := range device.Actuators {
				meta := acuator.Meta
				if meta.DoNotSync() {
					continue
				}

				for _, s := range device2.Actuators {
					if s.ID == v2IdCompat(acuator.ID) {
						if s.Value != nil {
							/*
								if s.Value.Time == noTime {
									s.Value.Time = s.Value.TimeReceived
								}
								if s.Value.Time.After(acuator.Time) {
									if acuator.Time == noTime {
										// log.Printf("[UP   ] Actuator %q outdated! Last value %v.", acuator.ID, acuator.Time)
									} else {
										// log.Printf("[UP   ] Actuator %q outdated! No values.", acuator.ID)
									}
									edge.PostActuatorValue(device.ID, acuator.ID, edge.NewValue(s.Value.Value, s.Value.Time))
								}
							*/
						}
						continue ACTUATORS
					}
				}
				// log.Printf("[UP   ] Actuator %q does not exist.", acuator.ID)

				cloud.flag(Entity{device.ID, "", acuator.ID}, ActionCreate, noTime, meta)

				// cloud.Status[Entity{device.ID, "", acuator.ID}] = NewStatus(ActionCreate, noTime)
			}

		default:
			cloud.Printf("Communication Error\nUnexpected response: %s:\n%s", 500, resp.statusText, resp.text())
			resp.Close()

			return resp.status
		}
	}

	// if _, err = cloud.client.SubscribeAll(subscriptions); err != nil {
	// 	return -1
	// }

	return http.StatusOK
}
