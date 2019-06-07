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

	"github.com/Waziup/wazigate-edge/api"
	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/Waziup/wazigate-edge/tools"
)

type MQTTServer struct {
	mqtt.Server
}

func (server *MQTTServer) Publish(client *mqtt.Client, msg *mqtt.Message) error {

	if client != nil {

		// TODO: check auth for PUBLISH

		log.Printf("[MQTT ] Publish %q %q QoS:%d len:%d\n", client.Id, msg.Topic, msg.QoS, len(msg.Data))

		body := tools.ClosingBuffer{bytes.NewBuffer(msg.Data)}
		rurl, _ := url.Parse("/" + msg.Topic)
		req := http.Request{
			Method: "PUBLISH",
			URL:    rurl,
			Header: http.Header{
				"X-Tag": []string{"MQTT "},
			},
			Body:          &body,
			ContentLength: int64(len(msg.Data)),
			RemoteAddr:    client.Id,
			RequestURI:    msg.Topic,
		}
		resp := MQTTResponse{
			status: 200,
			header: make(http.Header),
		}

		Serve(&resp, &req)
		if resp.status >= 200 && resp.status < 300 {
			server.Server.Publish(client, msg)
		}
	} else {

		server.Server.Publish(client, msg)
	}

	// Forward data to
	if strings.Contains(msg.Topic, "/sensors/") || strings.HasSuffix(msg.Topic, "/sensors") {
		pkt := mqtt.Publish(msg)
		for _, cloud := range api.Clouds {
			cloud.Queue.WritePacket(pkt)
		}
	}

	return nil
}

func (server *MQTTServer) Connect(client *mqtt.Client, auth *mqtt.ConnectAuth) byte {

	// TODO: check auth for CONNECT

	log.Printf("[MQTT ] Connect %q %+v\n", client.Id, auth)
	return mqtt.CodeAccepted
}

func (server *MQTTServer) Disconnect(client *mqtt.Client, err error) {

	log.Printf("[MQTT ] Disonnect %q %v\n", client.Id, err)
}

func (server *MQTTServer) Subscribe(recv mqtt.Reciever, topic string, qos byte) *mqtt.Subscription {

	if client, ok := recv.(*mqtt.Client); ok {
		// TODO: check auth for SUBSCRIBE
		log.Printf("[MQTT ] Subscribe %q %q QoS:%d\n", client.Id, topic, qos)
	}
	return server.Server.Subscribe(recv, topic, qos)
}

func (server *MQTTServer) Unsubscribe(subs *mqtt.Subscription) {

	if client, ok := subs.Recv.(*mqtt.Client); ok {
		log.Printf("[MQTT ] Unsubscribe %q %q QoS:%d\n", client.Id, subs.Topic.FullName(), subs.QoS)
	}
	server.Server.Unsubscribe(subs)
}

////////////////////////////////////////////////////////////////////////////////

var mqttServer = &MQTTServer{mqtt.NewServer()}

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
