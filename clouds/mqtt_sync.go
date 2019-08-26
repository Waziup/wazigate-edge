package clouds

import (
	"fmt"
	"log"
	"time"

	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/mqtt"
)

type Downstream interface {
	Publish(msg *mqtt.Message) int
}

var downstream Downstream = nil

func SetDownstream(ds Downstream) {
	downstream = ds
}

func (cloud *Cloud) IncludeDevice(deviceID string) {
	cloud.mqttMutex.Lock()
	cloud.devices[deviceID] = struct{}{}
	if cloud.client != nil {
		cloud.client.Subscribe("devices/"+deviceID+"/actuators/*/values", 1)
	}
	cloud.mqttMutex.Unlock()
}

func (cloud *Cloud) mqttSync() {

	cloud.mqttPersistentSync() // blocking
	cloud.PausingMQTT = false
}

func (cloud *Cloud) mqttPersistentSync() {

	nretry := 0

	retry := func() {

		if cloud.PausingMQTT {
			return
		}

		duration := retries[nretry]
		cloud.setStatus(cloud.StatusCode, fmt.Sprintf("Waiting %ds before retry after error.\n%s", duration/time.Second, cloud.StatusText))
		time.Sleep(duration)

		nretry++
		if nretry == len(retries) {
			nretry = len(retries) - 1
		}
	}

	for !cloud.PausingMQTT {

		for !cloud.PausingMQTT {
			client, err := mqtt.Dial(cloud.getMQTTAddr(), edge.LocalID(), true, &mqtt.ConnectAuth{
				Username: cloud.Credentials.Username,
				Password: cloud.Credentials.Token,
			}, nil)
			if err != nil {
				cloud.setStatus(0, "Err MQTT "+err.Error())
				retry()
				continue
			}
			cloud.mqttMutex.Lock()
			cloud.client = client
			cloud.mqttMutex.Unlock()
			break
		}

		cloud.setStatus(200, "MQTT Successfully connected.")

		cloud.mqttMutex.Lock()
		subs := make([]mqtt.TopicSubscription, len(cloud.devices))
		i := 0
		for deviceID := range cloud.devices {
			subs[i] = mqtt.TopicSubscription{
				Name: "devices/" + deviceID + "/actuators/*/values",
				QoS:  1,
			}
			i++
		}
		cloud.client.SubscribeAll(subs)
		cloud.mqttMutex.Unlock()

		for !cloud.PausingMQTT {

			msg, err := cloud.client.Message()
			if err != nil {
				log.Printf("[UP   ] MQTT Err %v", err)
				retry()
				break
			}
			if msg == nil {
				log.Printf("[UP   ] MQTT Err Unexpected disconnect.")
				retry()
				break
			}

			if downstream != nil {
				downstream.Publish(msg)
			}
		}

		cloud.client.Disconnect()
	}

	cloud.mqttMutex.Lock()
	cloud.client = nil
	cloud.mqttMutex.Unlock()
}
