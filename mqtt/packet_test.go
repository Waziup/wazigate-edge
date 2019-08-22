package mqtt

import (
	"bytes"
	"testing"
)

// This test has a lot of different packets and
// - writes the packet to a binary buffer A
// - reads the binary buffer to a new packet
// - writes the new packet to a new buffer B
// The old buffer A and the new buffer B must be equal.
func TestPacket(t *testing.T) {

	subscriptions := []TopicSubscription{
		{
			Name: "abc/def/#",
			QoS:  0,
		}, {
			Name: "X",
			QoS:  1,
		}, {
			Name: "a/*/b/abcdefghijklmnopqrstuvwxyz123456789äüöß",
			QoS:  2,
		},
	}

	topics := []string{
		"abc/def/#",
		"X",
		"a/*/b/abcdefghijklmnopqrstuvwxyz123456789äüöß",
	}

	//

	packets := []Packet{
		Connect("MQTT", 0x04, true, 1234, "myClientID", simpleMessage, noneAuh),
		Connect("", 0x00, false, 0, "", emptyMessage, emptyAuth),
		Connect("MQTT", 0x04, true, 1234, "abcdefghijklmnopqrstuvwxyz123456789äüöß-_", mediumMessage, simpleAuth),

		ConnAck(CodeAccepted, true),
		ConnAck(CodeBatUserOrPassword, false),

		Publish(1234, emptyMessage),
		Publish(0, simpleMessage),
		Publish(1234, mediumMessage),
		Publish(2123, longMessage),

		PubAck(0),
		PubAck(1234),

		PubRec(0),
		PubRec(1234),

		PubRel(0),
		PubRel(1234),

		PubComp(0),
		PubComp(1234),

		Subscribe(0, nil),
		Subscribe(1234, subscriptions),

		SubAck(0, nil, 0),
		SubAck(1234, subscriptions, 0x02),

		Unsubscribe(0, nil),
		Unsubscribe(1234, topics),

		UnsubAck(0),
		UnsubAck(1234),

		PingReq(),

		PingResp(),

		Disconnect(),
	}

	//

	buf := &bytes.Buffer{}
	cloneBuf := &bytes.Buffer{}

	for _, pkt := range packets {

		buf.Reset()
		size, err := pkt.WriteTo(buf)
		if err != nil {
			t.Fatalf("can not write packet to buffer: %v\n%v", err, pkt)
		}

		pktBytes := buf.Bytes()

		clone, _, err := Read(buf)
		if err != nil {
			t.Fatalf("can not read packet from buffer: %v\n%v", err, pkt)
		}

		cloneBuf.Reset()
		sizeClone, err := pkt.WriteTo(cloneBuf)
		if err != nil {
			t.Fatalf("can not write packet clone to buffer: %v\n%v", err, clone)
		}

		pktCloneBytes := cloneBuf.Bytes()

		if size != sizeClone {
			t.Fatalf("packet read write unequal: size %d <> %d \n%#v", size, sizeClone, pkt)
		}

		if !bytes.Equal(pktBytes, pktCloneBytes) {
			t.Fatalf("packet read write unequal\n%#v", pkt)
		}
	}
}
