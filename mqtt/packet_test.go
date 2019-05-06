package mqtt

import (
	"bytes"
	"reflect"
	"testing"
)

func testWriteRead(t *testing.T, pktSend Packet) {

	var buf bytes.Buffer
	n, err := pktSend.WriteTo(&buf)
	if err != nil {
		t.Fatal("WriteTo Error:", err)
	}
	pktRecieved, m, err := Read(&buf)
	if err != nil {
		t.Fatal("Read Error:", err)
	}
	if n != m {
		t.Fatalf("Length Error: Expected %d, Got %d", n, m)
	}
	if !reflect.DeepEqual(pktSend, pktRecieved) {
		t.Fatalf("Unequal: Expected %#v, Got %#v", pktSend, pktRecieved)
	}
}

func TestConnectPacket(t *testing.T) {

	will := &Message{
		Data:  []byte("Data"),
		Topic: "a/b",
		QoS:   0,
	}
	auth := &ConnectAuth{
		Username: "username",
		Password: "password",
	}
	testWriteRead(t, Connect("protocol", 0x12, true, 20, "clientId", will, auth))

	testWriteRead(t, Connect("", 0, false, 0, "", nil, nil))
}

func TestConnAckPacket(t *testing.T) {

	testWriteRead(t, ConnAck(0x01))
	testWriteRead(t, ConnAck(0x45))
	testWriteRead(t, ConnAck(0xff))
}

func TestSubscribePacket(t *testing.T) {

	topics := []TopicSubscription{
		TopicSubscription{
			Name: "a/b",
			QoS:  0x00,
		},
		TopicSubscription{
			Name: "ccccc",
			QoS:  0x01,
		},
		TopicSubscription{
			Name: "d $%&ä^^",
			QoS:  0x02,
		},
	}
	testWriteRead(t, Subscribe(123, topics))
	testWriteRead(t, Subscribe(123, []TopicSubscription{}))
}
