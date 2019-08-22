package mqtt

import (
	"io"
	"log"
	"os"
	"testing"
)

func TestRetain(t *testing.T) {

	logger := log.New(os.Stdout, "", 0)
	server := NewServer(authenticator, logger, LogLevelDebug)

	var message *Message

	recv1 := RecieverFunc(func(msg *Message) error {
		message = msg
		return nil
	})
	recv2 := RecieverFunc(func(msg *Message) error {
		message = msg
		return nil
	})
	recv3 := RecieverFunc(func(msg *Message) error {
		message = msg
		return nil
	})

	server.Publish(simpleMessage)
	topic := retainMessage.Topic
	server.Subscribe(recv1, topic, 0x00)

	if message != nil {
		t.Fatalf("recieved unexpected message: %v", message)
	}

	server.Publish(retainMessage)
	server.Subscribe(recv2, topic, 0x00)

	if message == nil {
		t.Fatalf("expected retain message")
	}

	server.Publish(clearRetainMessage)
	message = nil
	server.Subscribe(recv3, topic, 0x00)

	if message != nil {
		t.Fatalf("recieved unexpected message: %v", message)
	}
}

func TestRetainWildcards(t *testing.T) {

	logger := log.New(os.Stdout, "", 0)
	server := NewServer(authenticator, logger, LogLevelDebug)

	topics := []string{
		"a/b",     // 1
		"a/b/h",   // 2
		"b/d/h",   // 3
		"a/y",     // 4
		"a/f/h/g", // 5

		"b/d/h", // again
	}

	for _, topic := range topics {
		server.Publish(&Message{
			Topic:  topic,
			Retain: true,
			Data:   []byte("987xüz"), // must have data
			// no data == clear retain
		})
	}

	subscriptions := []struct {
		topic string
		hit   int
	}{
		{"a/b", 1},   // 1
		{"a/+", 2},   // 1, 4
		{"a/#", 4},   // 1, 2, 4, 5
		{"a/+/h", 1}, // 2
		{"+/+/+", 2}, // 2, 3
		{"#", 5},     // all
	}

	var hit int

	recv := RecieverFunc(func(msg *Message) error {
		hit++
		return nil
	})

	for _, subs := range subscriptions {
		hit = 0
		server.Subscribe(recv, subs.topic, 0x00)
		if hit != subs.hit {
			t.Fatalf("at %q expected %d hits, got %d", subs.topic, subs.hit, hit)
		}
	}

}

func TestServerClose(t *testing.T) {

	logger := log.New(os.Stdout, "", 0)
	server := NewServer(authenticator, logger, LogLevelDebug)

	clientStream, serverStream := NewMemStream()
	go server.Serve(serverStream)

	id := rndID()

	connAck := handshake(t, false, id, 0, simpleAuth, clientStream)
	if connAck.Code != CodeAccepted {
		t.Fatalf("simple client not accepted: code %v", connAck.Code)
	}
	if connAck.SessionPresent {
		t.Fatalf("unexpected connack session present")
	}

	// Subscribe to topic
	clientStream.WritePacket(Subscribe(1, simpleSubscription))
	_ = readPacket(t, clientStream).(*SubAckPacket) // must be SUBACK

	// closing the server must disconnect the client
	server.Close()

	if pkt, err := clientStream.ReadPacket(); err != io.EOF {
		_ = pkt.(*DisconnectPacket) // must be DISCONNECT
		if pkt, err := clientStream.ReadPacket(); err != io.EOF {
			t.Fatalf("server did non close the connection: %v", pkt)
		}
	}
}
