package clouds

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/mqtt"
)

type namedSender struct {
	name string
}

func (s namedSender) ID() string {
	return s.name
}

var cloudSender = namedSender{"downstream"}

// IncludeDevice tells the cloud to sync with that device,
// especially to monitor that device at the remote cloud for actuation data.
func (cloud *Cloud) IncludeDevice(deviceID string) {
	cloud.mqttMutex.Lock()
	cloud.devices[deviceID] = struct{}{}
	if cloud.client != nil {
		log.Printf("[UP   ] Waiting for actuation on \"devices/%q/actuators/+/value(s)\".", deviceID)
		cloud.client.Subscribe("devices/"+deviceID+"/actuators/+/values", 0)
		cloud.client.Subscribe("devices/"+deviceID+"/actuators/+/value", 0)
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
		log.Printf("[UP   ] Waiting %ds with MQTT before retry after error.", duration/time.Second)
		// cloud.setStatus(cloud.StatusCode, fmt.Sprintf("Waiting %ds before retry after error.\n%s", duration/time.Second, cloud.StatusText))
		time.Sleep(duration)

		nretry++
		if nretry == len(retries) {
			nretry = len(retries) - 1
		}
	}

	for !cloud.PausingMQTT {

		for !cloud.PausingMQTT {
			log.Printf("[UP   ] Connecting to MQTT as %q ...", cloud.Username)
			client, err := mqtt.Dial(cloud.getMQTTAddr(), edge.LocalID(), true, &mqtt.ConnectAuth{
				Username: cloud.Username,
				Password: cloud.Token,
			}, nil)
			if err != nil {
				cloud.Printf("Communication Error\nMQTT communication error:\n%s", 500, err.Error())
				retry()
				continue
			}
			cloud.mqttMutex.Lock()
			cloud.client = client
			cloud.mqttMutex.Unlock()
			break
		}

		if cloud.PausingMQTT {
			return
		}

		log.Printf("[UP   ] MQTT is now connected.")

		// cloud.Printf("MQTT sucessfully connected.", 200)

		tunnelDownTopic := "devices/" + edge.LocalID() + "/tunnel-down/"
		tunnelUpTopic := "devices/" + edge.LocalID() + "/tunnel-up/"

		cloud.mqttMutex.Lock()
		subs := make([]mqtt.TopicSubscription, len(cloud.devices)*2+1)
		i := 0
		for deviceID := range cloud.devices {
			subs[i] = mqtt.TopicSubscription{Name: "devices/" + deviceID + "/actuators/+/value"}
			i++
			subs[i] = mqtt.TopicSubscription{Name: "devices/" + deviceID + "/actuators/+/values"}
			i++
		}
		subs[i] = mqtt.TopicSubscription{Name: tunnelDownTopic + "+"}
		cloud.client.SubscribeAll(subs)
		cloud.mqttMutex.Unlock()

		for !cloud.PausingMQTT {

			msg, err := cloud.client.Message()
			if err != nil {
				cloud.Printf("MQTT Error\n%s", 400, err.Error())
				retry()
				break
			}
			if msg == nil {
				cloud.Printf("MQTT Error\nUnexpected disconnect.", 400)
				retry()
				break
			}

			if strings.HasPrefix(msg.Topic, tunnelDownTopic) {
				go cloud.thread_tunnel(msg, tunnelDownTopic, tunnelUpTopic)
				continue
			}

			if downstream != nil {
				log.Printf("[UP   ] Received: %s [%d]", msg.Topic, len(msg.Data))
				downstream.Publish(cloudSender, msg)
			}
		}

		cloud.client.Disconnect()
	}

	cloud.mqttMutex.Lock()
	cloud.client = nil
	cloud.mqttMutex.Unlock()
}

func (cloud *Cloud) thread_tunnel(msg *mqtt.Message, tunnelDownTopic string, tunnelUpTopic string) {
	ref := msg.Topic[len(tunnelDownTopic):]
	resp := tunnel(msg.Data)
	if resp != nil {
		cloud.client.Publish(&mqtt.Message{
			Topic: tunnelUpTopic + ref,
			Data:  resp,
		})
	}
}

var tunnelError []byte

func init() {
	var body = []byte(
		"The Gateway produced an internal error while processing the request.\r\n" +
			"See the Gateway logs for more details.\r\n" +
			"This message was produced by the Gateway.\r\n")
	var buf bytes.Buffer
	writeInt(&buf, http.StatusInternalServerError)
	headers, _ := json.Marshal(http.Header{
		"Content-Type":   []string{"text/plain; charset=utf-8"},
		"Content-Length": []string{strconv.Itoa(len(body))},
	})
	writeBytes(&buf, headers)
	writeBytes(&buf, body)
	tunnelError = buf.Bytes()
}

func tunnel(data []byte) []byte {

	// method string
	// uri string
	// header json bytes
	// body bytes
	l, method := readString(data)
	if l == 0 {
		log.Printf("[TUNL ] Error Invalid data.")
		return tunnelError
	}
	data = data[l:]

	l, uri := readString(data)
	if l == 0 {
		log.Printf("[TUNL ] Error Invalid data.")
		return tunnelError
	}
	data = data[l:]

	l, j := readBytes(data)
	if l == 0 {
		log.Printf("[TUNL ] Error Invalid data.")
		return tunnelError
	}
	header := make(http.Header)
	json.Unmarshal(j, &header)
	data = data[l:]

	l, body := readBytes(data)
	if l != len(data) {
		log.Printf("[TUNL ] Error Invalid data.")
		return tunnelError
	}

	req, err := http.NewRequest(method, "http://wazigate"+uri, bytes.NewReader(body))
	if err != nil {
		log.Printf("[TUNL ] Error %v", err)
		return tunnelError
	}
	req.Header = header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[TUNL ] Error %v", err)
		return tunnelError
	}

	var buf bytes.Buffer
	writeInt(&buf, resp.StatusCode)
	j, _ = json.Marshal(resp.Header)
	writeBytes(&buf, j)
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[TUNL ] Error %v", err)
		return tunnelError
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
