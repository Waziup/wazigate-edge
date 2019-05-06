package mqtt

import (
	"bytes"
	"io"
	"log"
	"testing"
	"time"
)

////////////////////////////////////////////////////////////////////////////////

type LoggingServer struct {
	Server Server
}

var connectCounter, disconnectCounter int

func (server *LoggingServer) PreConnect(stream io.ReadWriteCloser, connect *ConnectPacket, s Server) {
	server.Server.PreConnect(stream, connect, s)
}

func (server *LoggingServer) Publish(client *Client, msg *Message) error {

	log.Printf("PUBLISH     %q %q %q\n", client, msg.Topic, msg.Data)
	return server.Server.Publish(client, msg)
}

func (server *LoggingServer) Connect(client *Client, auth *ConnectAuth) byte {
	connectCounter++
	log.Printf("CONNECT     %d %q cs:%v %+v\n", connectCounter, client, client.CleanSession, auth)
	if auth != nil {
		if auth.Username+auth.Password == "not allowed" {
			return CodeBatUserOrPassword
		}
	}
	return CodeAccepted
}

func (server *LoggingServer) Disconnect(client *Client, err error) {
	disconnectCounter++
	log.Printf("DISCONNECT  %d %q %v\n", disconnectCounter, client, err)
}

func (server *LoggingServer) Subscribe(recv Reciever, topic string, qos byte) *Subscription {

	log.Printf("SUBSCRIBE   %q %q QoS:%d\n", recv, topic, qos)
	return server.Server.Subscribe(recv, topic, qos)
}

func (server *LoggingServer) Unsubscribe(subs *Subscription) {

	log.Printf("UNSUBSCRIBE %q %q QoS:%d\n", subs.Recv, subs.Topic.FullName(), subs.QoS)
	server.Server.Unsubscribe(subs)
}

////////////////////////////////////////////////////////////////////////////////

func TestServer(t *testing.T) {

	if !testing.Short() {

		addr := ":1883"
		server := &LoggingServer{NewServer()}

		go ListenAndServe(addr, server)

		//////////

		mario, err := Dial(addr, "It'sMeMario!", true, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		if connectCounter != 1 {
			t.Fatalf("Connected to server that was not this server... ? (%d)", connectCounter)
		}

		mario.Subscribe("a/+", 2)
		mario.Subscribe("a/y/#", 2)
		time.Sleep(time.Second / 2)

		//////////

		luigi, err := Dial(addr, "It'sMeLuigi!", true, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		data := []byte("Hello my brother :)")
		luigi.Publish(nil, &Message{
			Topic: "a/b",
			QoS:   2,
			Data:  data,
		})
		luigi.Disconnect()

		//////////

		msg := <-mario.Message()
		if msg == nil {
			t.Fatalf("Subscription closed prematurely.")
		}

		if msg.Topic != "a/b" {
			t.Fatalf("Recieved wrong topic: %q\n", msg.Topic)
		}
		if bytes.Compare(msg.Data, data) != 0 {
			t.Fatalf("Recieved wrong data: %q\n", msg.Data)
		}
		// log.Printf("Recieved: %q %q QoS:%d\n", msg.Topic, msg.Data, msg.QoS)

		//////////

		mario.Disconnect()

		/*
			msg = <-unused
			if msg != nil {
				log.Fatalln("Got a message for an unused subscription.")
			}
		*/

		//////////

		auth := &ConnectAuth{
			Username: "not ",
			Password: "allowed",
		}
		if _, e := Dial(addr, "NotAllowed", true, auth, nil); e == nil {
			t.Fatal("Client 'NotAllowed' was not rejected.")
		}

		//////////

		sess, err := Dial(addr, "WithSession1", false, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		sess.Subscribe("z", 1)
		time.Sleep(time.Second / 2)
		sess.Closer.Close() // close tcp

		publ, _ := Dial(addr, "Publisher1", true, nil, nil)
		data = []byte("Woohoo :)")
		publ.Publish(nil, &Message{Topic: "z", QoS: 2, Data: data})
		publ.Disconnect()

		sess, err = Dial(addr, "WithSession1", false, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		// subs, _ = sess.Subscribe("z", 0)
		msg = <-sess.Message()
		sess.Disconnect()

		if msg == nil || bytes.Compare(msg.Data, data) != 0 {
			t.Fatalf("Go no or invalid message: %v\n", msg)
		}

		//////////

		time.Sleep(time.Second / 2)

		if connectCounter != 6 {
			// it should be 6 connections in this test
			t.Fatalf("Expected 6 connections, got %d.\n", connectCounter)
		}

		if disconnectCounter != 5 {
			// 5 disconnections (because of the 'not-allowed' test)
			t.Fatalf("Expected 5 disconnections, got %d.\n", disconnectCounter)
		}
	}
}
