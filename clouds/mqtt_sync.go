package clouds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
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
		cloud.client.Subscribe("devices/"+deviceID+"/actuators/+/values", 1)
	}
	cloud.mqttMutex.Unlock()
}

func (cloud *Cloud) mqttSync() {

	cloud.mqttPersistentSync() // blocking
	cloud.PausingMQTT = false
	log.Println("[UP   ] MQTT sync is now paused.")
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
			log.Printf("[UP   ] Connecting to MQTT as %q ...", cloud.Credentials.Username)
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

		tunnelDownTopic := "devices/" + edge.LocalID() + "/tunnel-down/"
		tunnelUpTopic := "devices/" + edge.LocalID() + "/tunnel-up/"

		cloud.mqttMutex.Lock()
		subs := make([]mqtt.TopicSubscription, len(cloud.devices)+1)
		i := 0
		for deviceID := range cloud.devices {
			subs[i] = mqtt.TopicSubscription{
				Name: "devices/" + deviceID + "/actuators/+/values",
				QoS:  1,
			}
			i++
		}
		subs[i] = mqtt.TopicSubscription{
			Name: tunnelDownTopic + "+",
			QoS:  0,
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

			if strings.HasPrefix(msg.Topic, tunnelDownTopic) {
				ref := msg.Topic[len(tunnelDownTopic):]
				resp := tunnel(msg.Data)
				if resp != nil {
					cloud.client.Publish(&mqtt.Message{
						Topic: tunnelUpTopic + ref,
						Data:  resp,
					})
				}
				continue
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

func tunnel(data []byte) []byte {

	// method string
	// uri string
	// header json bytes
	// body bytes
	l, method := readString(data)
	if l == 0 {
		log.Printf("[TUNL ] Error Invalid data.")
		return nil
	}
	data = data[l:]

	l, uri := readString(data)
	if l == 0 {
		log.Printf("[TUNL ] Error Invalid data.")
		return nil
	}
	data = data[l:]

	l, j := readBytes(data)
	if l == 0 {
		log.Printf("[TUNL ] Error Invalid data.")
		return nil
	}
	header := make(http.Header)
	json.Unmarshal(j, &header)
	data = data[l:]

	l, body := readBytes(data)
	if l != len(data) {
		log.Printf("[TUNL ] Error Invalid data.")
		return nil
	}

	req, err := http.NewRequest(method, "http://127.0.0.1"+uri, bytes.NewReader(body))
	if err != nil {
		log.Printf("[TUNL ] Error %v", err)
		return nil
	}
	req.Header = header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[TUNL ] Error %v", err)
		return nil
	}

	var buf bytes.Buffer
	writeInt(&buf, resp.StatusCode)
	j, _ = json.Marshal(resp.Header)
	writeBytes(&buf, j)
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[TUNL ] Error %v", err)
		return nil
	}
	writeBytes(&buf, body)
	log.Printf("[TUNL ] %d %s s:%d", resp.StatusCode, uri, len(body))
	return buf.Bytes()
}

////////////////////////////////////////////////////////////////////////////////

func readString(buf []byte) (int, string) {
	length, b := readBytes(buf)
	return length, string(b)
}

func readBytes(buf []byte) (int, []byte) {

	if len(buf) < 2 {
		return 0, nil
	}
	length := (int(buf[0])<<16 + int(buf[1])<<8 + int(buf[2])) + 3
	if len(buf) < length {
		return 0, nil
	}
	return length, buf[3:length]
}

func writeString(w io.Writer, str string) (int, error) {
	return writeBytes(w, []byte(str))
}

func writeBytes(w io.Writer, b []byte) (int, error) {
	m, err := w.Write([]byte{byte(len(b) >> 16), byte(len(b) >> 8), byte(len(b) & 0xff)})
	if err != nil {
		return m, err
	}
	n, err := w.Write(b)
	return m + n, err
}

func writeInt(w io.Writer, i int) (int, error) {
	return w.Write([]byte{byte(i >> 8), byte(i & 0xff)})
}
