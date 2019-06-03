package api

import (
	"log"
	"net/url"
	"time"

	"github.com/Waziup/waziup-edge/mqtt"
)

var Downstream mqtt.Server
var upstream *mqtt.Queue

var retries = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	20 * time.Second,
	60 * time.Second,
}

func (cloud *Cloud) endSync() {
	if cloud.Client != nil {
		cloud.counter++
		cloud.Client.Disconnect()
		cloud.Client = nil
	}
}

func (cloud *Cloud) beginSync(counter int) {

	/*
		resp, err := http.Get("https://" + cloud.URL + "/devices")
		if err != nil {
			log.Println(err)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println(resp.Status, string(body))
		return
	*/

	nretry := 0

	addr := cloud.URL
	u, err := url.Parse("//" + cloud.URL)
	if err != nil {
		log.Println("[UP   ] Err", err)
		return
	}
	if u.Port() == "" {
		addr = u.Hostname() + ":1883" + u.RequestURI()
	}
	if addr[len(addr)-1] == '/' {
		addr = addr[:len(addr)-1]
	}

	for !cloud.Paused {
		log.Printf("[UP   ] Dialing Upstream at %q...\n", addr)
		auth := &mqtt.ConnectAuth{
			Username: cloud.Credentials.Username,
			Password: cloud.Credentials.Token,
		}
		client, err := mqtt.Dial(addr, GetLocalID(), false, auth, nil)
		cloud.Client = client
		if counter != cloud.counter {
			client.Disconnect()
			cloud.Client = nil
			return
		}
		if err != nil {
			log.Printf("[UP   ] Error: %v\n", err)
			duration := retries[nretry]
			log.Printf("[UP   ] Waiting %ds before retry.\n", duration/time.Second)
			time.Sleep(duration)

			if counter != cloud.counter {
				cloud.Client = nil
				return
			}

			nretry++
			if nretry == len(retries) {
				nretry = len(retries) - 1
			}
			continue
		}

		log.Printf("[UP   ] Connected.\n")
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
			if counter != cloud.counter {
				client.Disconnect()
				cloud.Client = nil
				return
			}

			log.Printf("[UP   ] Recieved \"%s\" QoS:%d len:%d\n", msg.Topic, msg.QoS, len(msg.Data))

			if Downstream != nil {
				Downstream.Publish(client, msg)
			}
		}

		log.Printf("[UP   ] Disconnected: %v\n", client.Error)
	}

	cloud.Client = nil
}
