package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/Waziup/wazigate-edge/mqtt"
)

var Downstream mqtt.Server

var retries = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	20 * time.Second,
	60 * time.Second,
}

func (cloud *Cloud) endSync() {
	if cloud.Client != nil {
		cloud.Client.Disconnect()
		cloud.Client = nil
	}
	cloud.counter++
}

func (cloud *Cloud) beginSync(counter int) {

	nretry := 0

	cloud.setStatus(0, "Beginning sync...")

	retry := func() {

		duration := retries[nretry]
		cloud.setStatus(0, fmt.Sprintf("Waiting %ds before retry after error.\n%s", duration/time.Second, cloud.StatusText))
		log.Printf("[UP   ] Waiting %ds before retry.\n", duration/time.Second)
		time.Sleep(duration)

		nretry++
		if nretry == len(retries) {
			nretry = len(retries) - 1
		}
	}

	for !cloud.Paused && cloud.counter == counter {

		if !cloud.initialSync() {
			retry()
			continue
		}

		if cloud.Paused || cloud.counter != counter {
			break
		}

		if !cloud.persistentSync() {
			retry()
			continue
		}
	}

	cloud.setStatus(0, "Disconnected.")
}

////////////////////////////////////////////////////////////////////////////////

type v2Sensor struct {
	ID   string `json:"id" bson:"id"`
	Name string `json:"name" bson:"name"`
}

type v2Actuator struct {
	ID   string `json:"id" bson:"id"`
	Name string `json:"name" bson:"name"`
}

type v2Device struct {
	Name      string       `json:"name" bson:"name"`
	ID        string       `json:"id" bson:"_id"`
	Sensors   []v2Sensor   `json:"sensors" bson:"sensors"`
	Actuators []v2Actuator `json:"actuators" bson:"actuators"`
}

type v2Gateway struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

func (cloud *Cloud) initialSync() bool {

	cloud.setStatus(0, "Connecting to server for initial sync...")

	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		cloud.Credentials.Username,
		cloud.Credentials.Token,
	}
	// Get Authentication Token
	//
	body, _ := json.Marshal(credentials)
	addr := cloud.getRESTAddr()
	log.Printf("[UP   ] Dialing REST %q...", addr)
	resp, err := http.Post(addr+"/auth/token", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[UP   ] Err %s", err.Error())
		cloud.setStatus(-1, fmt.Sprintf("Unable to connect.\n%s", err.Error()))
		return false
	}

	body, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		log.Printf("[UP   ] Err %s %q", resp.Status, err)
		cloud.setStatus(resp.StatusCode, fmt.Sprintf("REST failed: %s.\n%s", resp.Status, err.Error()))
		return false
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[UP   ] Err %s %q", resp.Status, string(body))
		cloud.setStatus(resp.StatusCode, fmt.Sprintf("Authentication failed: %s.\n%s", resp.Status, body))
		return false
	}

	auth := "Bearer " + string(body)
	log.Println("[UP   ] Authentication successfull.")

	// Call /gateways

	var device Device
	DBDevices.FindId(GetLocalID()).One(&device)

	var gateway = v2Gateway{
		ID:         GetLocalID(),
		Name:       device.Name,
		Visibility: "public",
	}

	body, _ = json.Marshal(gateway)
	req, err := http.NewRequest(http.MethodPost, addr+"/gateways", bytes.NewReader(body))
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[UP   ] Err %s %q", resp.Status, err)
		cloud.setStatus(resp.StatusCode, fmt.Sprintf("REST failed: %s.\n%s", resp.Status, err.Error()))
		return false
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[UP   ] Err %s %q", resp.Status, err)
		cloud.setStatus(resp.StatusCode, fmt.Sprintf("REST failed: %s.\n%s", resp.Status, err.Error()))
		return false
	}
	switch resp.StatusCode {
	case http.StatusOK:
		log.Printf("[UP   ] Gateway was declared at cloud.")
	default:
		log.Printf("[UP   ] Err %s %q", resp.Status, string(body))
	}

	// Get all devices from this Gateway and update all.
	//

	iter := DBDevices.Find(nil).Iter()
	for iter.Next(&device) {
		req, err := http.NewRequest(http.MethodGet, addr+"/devices/"+device.ID, nil)
		req.Header.Set("Authorization", auth)
		resp, err = http.DefaultClient.Do(req)

		if err != nil {
			log.Printf("[UP   ] Err %s %q", resp.Status, err)
			cloud.setStatus(resp.StatusCode, fmt.Sprintf("REST failed: %s.\n%s", resp.Status, err.Error()))
			iter.Close()
			return false
		}
		switch resp.StatusCode {
		case http.StatusNotFound:
			log.Printf("[UP   ] Device %q not found. Will be pushed.", device.ID)
			resp.Body.Close()

			// -- Sync for v2 only!
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
			// --

			data, _ := json.Marshal(syncDev)
			req2, _ := http.NewRequest(http.MethodPost, addr+"/devices", bytes.NewReader(data))
			req2.Header.Set("Authorization", auth)
			req2.Header.Set("Content-Type", "application/json")
			resp2, err := http.DefaultClient.Do(req2)
			if err != nil {
				log.Printf("[UP   ] Err %s %q", resp.Status, err)
				cloud.setStatus(resp.StatusCode, fmt.Sprintf("REST failed: %s.\n%s", resp.Status, err.Error()))
				return false
			}
			if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusNoContent {
				body, _ := ioutil.ReadAll(resp2.Body)
				log.Printf("[UP   ] Err Device %q sync failed: %s", device.ID, resp2.Status)
				log.Printf("[UP   ] Err Body: %q", body)
				cloud.setStatus(resp2.StatusCode, fmt.Sprintf("Can not sync device %q\n%s", device.Name, resp2.Status))
				resp2.Body.Close()
				iter.Close()
				return false
			}

		case http.StatusOK:
			log.Printf("[UP   ] Device %q found. Checking for updates.", device.ID)

			/*
				decoder := json.NewDecoder(resp.Body)
				var device2 Device
				err := decoder.Decode(&device2)
				resp.Body.Close()

				if err != nil {
					log.Printf("[UP   ] Err %s %q", resp.Status, err)
					cloud.setStatus(resp, = fmt.Sprintf("REST failed: %s.\n%s", resp.Status, err.Error())
					iter.Close()
					return
				}

				log.Printf("%#v", device2)
			*/

		default:
			log.Printf("[UP   ] Err Unexpected status %d for device %q", resp.StatusCode, device.ID)
		}
		resp.Body.Close()
	}
	iter.Close()
	return true
}

func (cloud *Cloud) persistentSync() bool {

	cloud.setStatus(0, "Connecting to server for persistent sync...")

	addr := cloud.getMQTTAddr()
	log.Printf("[UP   ] Dialing MQTT %q...\n", addr)
	auth := &mqtt.ConnectAuth{
		Username: cloud.Credentials.Username,
		Password: cloud.Credentials.Token,
	}
	client, err := mqtt.Dial(addr, GetLocalID(), false, auth, nil)
	cloud.Client = client

	if err != nil {
		log.Printf("[UP   ] Error: %v\n", err)
		cloud.setStatus(701, fmt.Sprintf("MQTT connection failed.\n%s", err.Error()))
		return false
	}

	log.Printf("[UP   ] Connected.\n")
	cloud.setStatus(200, "MQTT connected for persistent sync.")
	cloud.Queue.ServeWriter(client)

	if DBDevices != nil {
		var device Device
		// Subscribe to all actuators
		devices := DBDevices.Find(nil).Iter()
		for devices.Next(&device) {
			client.Subscribe("devices/"+device.ID+"/actuators/#", 0)
		}
		devices.Close()
	}

	//

	for msg := range client.Message() {
		log.Printf("[UP   ] Recieved \"%s\" QoS:%d len:%d\n", msg.Topic, msg.QoS, len(msg.Data))

		if Downstream != nil {
			Downstream.Publish(client, msg)
		}
	}

	// if len(client.Pending) != 0 {
	//	for _, pkt := range client.Pending {
	//		cloud.Queue.WritePacket(pkt)
	//	}
	// }

	cloud.setStatus(0, "Disconnected.")
	log.Printf("[UP   ] Disconnected: %v\n", client.Error)
	cloud.Client = nil
	return true
}
