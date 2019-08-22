package mqtt

import (
	"crypto/rand"
	"io"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"
)

////////////////////

var simpleAuth = &ConnectAuth{
	Username: "myUsername",
	Password: "myPassword",
}

var emptyAuth = &ConnectAuth{
	Username: "",
	Password: "",
}

var noneAuh *ConnectAuth

var forbiddenAuth = &ConnectAuth{
	Username: "forbidden",
	Password: "forbidden",
}

////////////////////

var simpleSubscription = []TopicSubscription{
	{
		Name: "a/b",
		QoS:  0,
	},
}

////////////////////

var simpleMessage = &Message{
	Topic: "a/b",
	QoS:   0,
	Data:  []byte("asd456"),
}

var emptyMessage = &Message{
	Topic: "a/b",
	QoS:   1,
	Data:  []byte(""),
}

var retainMessage = &Message{
	Topic:  "c/de",
	QoS:    1,
	Retain: true,
	Data:   []byte("987vmölp"),
}

var clearRetainMessage = &Message{
	Topic:  "c/de",
	QoS:    1,
	Retain: true,
	Data:   []byte(""), // empty body removes existing retain msg
}

var mediumMessage = &Message{
	Topic: "a/b",
	QoS:   1, // must be QoS 1 for some tests
	Data:  rndBuffer(300),
}

var longMessage = &Message{
	Topic: "a/b",
	QoS:   2,
	Data:  rndBuffer(30000),
}

func rndBuffer(size int) []byte {
	buf := make([]byte, size)
	rand.Read(buf)
	return buf
}

////////////////////

var lastAuth *ConnectAuth

func authenticator(client *Client, auth *ConnectAuth) ConnectCode {
	lastAuth = auth
	if auth != nil {
		if auth.Username == "forbidden" {
			return CodeBatUserOrPassword
		}
	}
	return CodeAccepted
}

////////////////////

// handshake = MQTT Connect + ConAck
func handshake(t *testing.T, cleanSess bool, id string, keepAlive int, auth *ConnectAuth, stream Stream) *ConnAckPacket {

	stream.WritePacket(Connect("MQTT", 0x04, cleanSess, keepAlive, id, nil, auth))

	packet, err := stream.ReadPacket()
	if err != nil {
		t.Fatalf("initial read failed: %v", err)
	}
	if !reflect.DeepEqual(lastAuth, simpleAuth) {
		t.Fatalf("auth was not passed to server: %v", lastAuth)
	}
	return packet.(*ConnAckPacket)
}

func readPacket(t *testing.T, stream Stream) Packet {
	packet, err := stream.ReadPacket()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	return packet
}

func rndID() string {
	seed := uint32(time.Now().UnixNano() + int64(os.Getpid()))
	return strconv.Itoa(int(1e9 + seed%1e9))
}

////////////////////////////////////////////////////////////////////////////////

type memStream struct {
	write    chan Packet
	sigClose chan struct{}
	sibling  *memStream
	closed   bool
}

func (stream *memStream) ReadPacket() (pkt Packet, err error) {
	if stream.closed {
		return nil, io.EOF
	}
	select {
	case packet := <-stream.write:
		return packet, nil

	case <-stream.sigClose:
		close(stream.sigClose)
		close(stream.write)
		stream.closed = true
		return nil, io.EOF
	}
}

func (stream *memStream) WaitClose() {
	<-stream.sigClose
	close(stream.sigClose)
	close(stream.write)
	stream.closed = true
}

func (stream *memStream) WritePacket(packet Packet) error {
	stream.sibling.write <- packet
	return nil
}

func (stream *memStream) Close() error {
	stream.sibling.sigClose <- struct{}{}
	return nil
}

// NewMemStream creates a upstream + downstream pair.
// Write to one stream and read that on the other.
// Close a stream to make the others reading fail (half side closed).
// We use MemStream to avoid the TCP layer for testing - now the tests
// can be done in memory, thats pretty cool and very fast.
func NewMemStream() (Stream, Stream) {
	var a, b memStream
	a.write = make(chan Packet, 8)
	a.sigClose = make(chan struct{}, 1)
	b.write = make(chan Packet, 8)
	b.sigClose = make(chan struct{}, 1)
	a.sibling = &b
	b.sibling = &a
	return &a, &b
}
