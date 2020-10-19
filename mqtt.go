package main

import (
	"bytes"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/Waziup/wazigate-edge/tools"
)

type MQTTServer struct {
	mqtt.Server
}

var MethodPublish = "PUBLISH"

func mqttAuth(client *mqtt.Client, auth *mqtt.ConnectAuth) mqtt.ConnectCode {
	client.Server = mqttServer

	// if auth == nil {
	// 	return mqtt.CodeNotAuthorized
	// }

	// var err error
	// if auth.Username == "" {
	// 	_, err = api.CheckToken(auth.Password)
	// } else {
	// 	_, err = edge.CheckUserCredentials(auth.Username, auth.Password)
	// }
	// if err != nil {
	// 	log.Printf("[MQTT ] Login failed: %v", err)
	// 	return mqtt.CodeBatUserOrPassword
	// }

	return mqtt.CodeAccepted
}

func isUnsupervised(path string) bool {
	return path != "device" && !strings.HasPrefix(path, "device/") &&
		path != "sensors" && !strings.HasPrefix(path, "sensors/") &&
		path != "actuators" && !strings.HasPrefix(path, "actuators/") &&
		path != "devices" && !strings.HasPrefix(path, "devices/") &&
		path != "clouds" && !strings.HasPrefix(path, "clouds/") &&
		path != "sys" && !strings.HasPrefix(path, "sys/")
}

func (server *MQTTServer) Publish(sender mqtt.Sender, msg *mqtt.Message) int {

	if sender == nil {
		// internal messages (no client as sender)
		return server.Server.Publish(nil, msg)
	}

	if isUnsupervised(msg.Topic) {
		return server.Server.Publish(nil, msg)
	}

	body := tools.ClosingBuffer{bytes.NewBuffer(msg.Data)}
	uri := "/" + msg.Topic
	rurl, _ := url.Parse(uri)
	req := http.Request{
		Method: MethodPublish,
		URL:    rurl,
		Header: http.Header{
			"X-Tag":   []string{"MQTT "},
			"X-Proto": []string{"mqtt"},
		},
		Body:          &body,
		ContentLength: int64(len(msg.Data)),
		RemoteAddr:    sender.ID(),
		RequestURI:    uri,
	}
	resp := MQTTResponse{
		status: 200,
		header: make(http.Header),
	}
	hit := Serve(&resp, &req)
	return hit
}

/*

func (server *MQTTServer) Connect(client *mqtt.Client, auth *mqtt.ConnectAuth) mqtt.ConnectCode {

	return mqtt.CodeAccepted
}

func (server *MQTTServer) Disconnect(client *mqtt.Client, err error) {

	log.Printf("[MQTT ] Disonnect %q %v\n", client.ID(), err)
}

func (server *MQTTServer) Subscribe(recv mqtt.Reciever, topic string, qos byte) *mqtt.Subscription {

	if client, ok := recv.(*mqtt.Client); ok {
		// TODO: check auth for SUBSCRIBE
		log.Printf("[MQTT ] Subscribe %q %q QoS:%d\n", client.ID(), topic, qos)
	}
	return server.Server.Subscribe(recv, topic, qos)
}

func (server *MQTTServer) Unsubscribe(subs *mqtt.Subscription) {

	if client, ok := subs.Recv.(*mqtt.Client); ok {
		log.Printf("[MQTT ] Unsubscribe %q %q QoS:%d\n", client.ID(), subs.Topic.FullName(), subs.QoS)
	}
	server.Server.Unsubscribe(subs)
}
*/

////////////////////////////////////////////////////////////////////////////////

var mqttServer *MQTTServer

func ListenAndServerMQTT() {

	addr := os.Getenv("WAZIUP_MQTT_ADDR")
	if addr == "" {
		addr = ":1883"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("[MQTT ] Error:\n", err)
	}

	log.Printf("[MQTT ] MQTT Server at %q.", addr)
	go func() {
		log.Fatal(mqtt.Serve(listener, mqttServer))
	}()
}

func ListenAndServeMQTTTLS(config *tls.Config) {

	addr := os.Getenv("WAZIUP_MQTTS_ADDR")
	if addr == "" {
		addr = ":8883"
	}

	listener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		log.Fatalln("[MQTTS] Error:\n", err)
	}

	log.Printf("[MQTTS] MQTT (with TLS) Server at %q.", addr)
	go func() {
		log.Fatal(mqtt.Serve(listener, mqttServer))
	}()
}

////////////////////////////////////////////////////////////////////////////////

type MQTTResponse struct {
	status int
	header http.Header
}

func (resp *MQTTResponse) Header() http.Header {
	return resp.header
}

func (resp *MQTTResponse) Write(data []byte) (int, error) {
	return len(data), nil
}

func (resp *MQTTResponse) WriteHeader(statusCode int) {
	resp.status = statusCode
}
