package mqtt

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// ConnectAuth is the authentication used with Connect packets.
type ConnectAuth struct {
	Username, Password string
}

var MaxPending = 16

var ErrMaxPending = errors.New("reached max pending")

// Client is a MQTT Client.
// Use `Dial` to create a new client.
type Client struct {

	// ID = Client IDentifier
	id string

	// Pending is a list of packets of QoS > 0 waiting for acknowledgement.
	// Pending map[int]Packet

	pending      map[int]Packet
	pendingMutex sync.Mutex

	MaxPending int

	// timeout time.Duration

	// Acknowledgments: make(chan int, 16),

	subscriptions map[string]*Subscription

	//	CleanSession bool

	stream Stream

	log      *log.Logger
	LogLevel LogLevel

	// subs map[string]*Subscription

	Server Server
	// session *Session

	will *Message

	counter int

	// queue *os.File
}

func (client *Client) ID() string {
	return client.id
}

func (client *Client) Will() *Message {
	return client.will
}

func (client *Client) Disconnect() error {
	err := client.Send(Disconnect())
	client.stream.Close()
	return err
}

// Dial connects to a remote MQTT server.
func Dial(addr string, id string, cleanSession bool, auth *ConnectAuth, will *Message) (*Client, error) {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	stream := NewStream(conn, time.Second*30)

	client := &Client{
		id: id,
		//	CleanSession: cleanSession,
		stream: stream,

		pending:    make(map[int]Packet, 16),
		MaxPending: MaxPending,
	}

	// send MQTT Connect packet
	client.Send(Connect("MQTT", byte(0x04), cleanSession, 5000, id, will, auth))

	pkt, err := stream.ReadPacket()
	if err != nil {
		return client, err
	}

	if connAck, ok := pkt.(*ConnAckPacket); ok {
		switch connAck.Code {
		case CodeAccepted: // Yeah :)
			return client, nil

		default:
			return client, fmt.Errorf("connect error: %s", connAck.Code)
		}
	} else {

		return client, errUnexpectedPacket(pkt.Header().PacketType)
	}
}

func errUnexpectedPacket(p PacketType) error {
	return fmt.Errorf("unexpected packet: %s", p)
}

func (client *Client) Packet() (Packet, *Message, error) {

	server := client.Server

	packet, err := client.stream.ReadPacket()
	if err != nil {

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {

			client.Send(PingReq())
			packet, err = client.stream.ReadPacket()
			if err != nil {
				return nil, nil, err
			}
		} else {

			return nil, nil, err
		}
	}

	if client.log != nil && client.LogLevel >= LogLevelDebug {
		client.log.Printf("%.24q > %s", client.id, packet)
	}

	switch pkt := packet.(type) {

	case *SubscribePacket:

		if server == nil {
			return nil, nil, errUnexpectedPacket(pkt.header.PacketType)
		}

		granted, _ := client.SubscribeAll(pkt.Topics)
		client.Send(SubAck(pkt.ID, granted))

	case *SubAckPacket:

		client.Acknowledge(pkt.ID)

	case *UnsubscribePacket:

		if server == nil {
			return nil, nil, errUnexpectedPacket(pkt.header.PacketType)
		}

		client.Unsubscribe(pkt.Topics...)

		client.Send(UnsubAck(pkt.ID))

	case *UnsubAckPacket:

		client.Acknowledge(pkt.ID)

	case *PublishPacket:

		switch pkt.Header().QoS {
		case 0x00: // At most once

			return pkt, pkt.Message(), nil

		case 0x01: // At least once

			// Acknowledge the Publishing
			client.Send(PubAck(pkt.ID))

			return pkt, pkt.Message(), nil

		case 0x02: // Exactly once

			client.Send(PubRec(pkt.ID))

			client.pendingMutex.Lock()
			_, duplicate := client.pending[pkt.ID]
			client.pendingMutex.Unlock()

			if duplicate {
				return pkt, nil, nil
			}
			return pkt, pkt.Message(), nil
		}

	case *PubAckPacket:

		client.Acknowledge(pkt.ID)

	case *PubRelPacket:

		client.Send(PubComp(pkt.ID))

	case *PubRecPacket:

		client.Acknowledge(pkt.ID)
		client.Send(PubRel(pkt.ID))

	case *PubCompPacket:

		client.Acknowledge(pkt.ID)

	case *PingReqPacket:

		// Ping Request -> Response
		client.Send(PingResp())

	case *PingRespPacket:
		// nothing to do here

	case *DisconnectPacket:

		return nil, nil, nil

	default:
		err := errUnexpectedPacket(pkt.Header().PacketType)
		return pkt, nil, err
	}
	return packet, nil, nil
}

func (client *Client) Acknowledge(mid int) {

	//sess.mutex.Lock()
	client.pendingMutex.Lock()
	delete(client.pending, mid)
	client.pendingMutex.Unlock()
	//sess.mutex.Unlock()
}

func (client *Client) Subscribe(topic string, qos byte) (byte, error) {

	if client.Server == nil {

		client.counter++
		if client.counter == 65000 {
			client.counter = 1
		}
		err := client.Send(Subscribe(client.counter, []TopicSubscription{{topic, qos}}))
		//FIXME: Does not report the actual QoS granted by the server
		return qos, err
	}

	subs := client.subscriptions[topic]
	if subs != nil {
		client.Server.Unsubscribe(subs)
	}
	subs = client.Server.Subscribe(client, topic, qos)
	client.subscriptions[topic] = subs
	return subs.qos, nil
}

func (client *Client) SubscribeAll(topics []TopicSubscription) ([]byte, error) {

	if client.Server == nil {

		client.counter++
		if client.counter == 65000 {
			client.counter = 1
		}
		err := client.Send(Subscribe(client.counter, topics))
		//FIXME: Does not report the actual QoS granted by the server
		return nil, err
	}

	duplicates := make([]*Subscription, 0)

	for _, topic := range topics {
		if subs, ok := client.subscriptions[topic.Name]; ok {
			duplicates = append(duplicates, subs)
		}
	}

	if len(duplicates) != 0 {
		client.Server.Unsubscribe(duplicates...)
	}

	subs := client.Server.SubscribeAll(client, topics)
	granted := make([]byte, len(topics))
	for i, topic := range topics {
		client.subscriptions[topic.Name] = subs[i]
		granted[i] = subs[i].qos
	}
	return granted, nil
}

func (client *Client) Unsubscribe(topics ...string) {

	if client.Server == nil {
		client.counter++
		if client.counter == 65000 {
			client.counter = 1
		}
		client.Send(Unsubscribe(client.counter, topics))
		return
	}

	subs := make([]*Subscription, len(topics))
	for _, topic := range topics {

		if sub, ok := client.subscriptions[topic]; ok {
			subs = append(subs, sub)
			delete(client.subscriptions, topic)
			if client.log != nil && client.LogLevel >= LogLevelNormal {
				client.log.Printf("%.24q Unsubscribed %q", client.id, topic)
			}
		} else {
			if client.log != nil && client.LogLevel >= LogLevelWarnings {
				client.log.Printf("%.24q Unsubscribed unexisting %q", client.id, topic)
			}
		}
	}

	if len(subs) != 0 {
		client.Server.Unsubscribe(subs...)
	}
}

// Message waits for a incomming publish.
func (client *Client) Message() (*Message, error) {

	for true {
		packet, message, err := client.Packet()
		if message != nil || err != nil || packet == nil {
			return message, err
		}
	}
	return nil, nil // unreachable
}

// Send a MQTT controll packet.
func (client *Client) Send(pkt Packet) error {

	if client.log != nil && client.LogLevel >= LogLevelDebug {
		client.log.Printf("%.24q < %s", client.id, pkt)
	}

	header := pkt.Header()
	if header.QoS != 0x00 {
		client.pendingMutex.Lock()
		if len(client.pending) >= client.MaxPending {
			client.pendingMutex.Unlock()
			return ErrMaxPending
		}

		var id int
		switch packet := pkt.(type) {
		case *PublishPacket:
			id = packet.ID
		case *PubRelPacket:
			id = packet.ID
		case *SubscribePacket:
			id = packet.ID
		case *UnsubscribePacket:
			id = packet.ID
		}

		client.pending[id] = pkt
		client.pendingMutex.Unlock()
	}

	return client.stream.WritePacket(pkt)
}

// Ping sends a Ping packet.
func (client *Client) Ping() error {
	return client.Send(PingReq())
}

// NumPending gives the number of outstanding QoS>0 packets that have now been acknowledged yet.
func (client *Client) NumPending() int {
	client.pendingMutex.Lock()
	num := len(client.pending)
	client.pendingMutex.Unlock()
	return num
}

// Publish a new message.
func (client *Client) Publish(msg *Message) error {

	if msg.QoS > 0 {
		client.counter++
		if client.counter == 65000 {
			client.counter = 1
		}
	}

	return client.Send(Publish(client.counter, msg))
}

func (client *Client) SetLogger(logger *log.Logger, level LogLevel) {
	client.LogLevel = level
	client.log = logger
}

/*
// WriteSession writes the Session State of this client to the io.Writer.
// The Session State in the Client consists of:
// - QoS 1 and QoS 2 messages which have been sent to the Server, but have not been completely acknowledged.
// - QoS 2 messages which have been received from the Server, but have not been completely acknowledged.
func (client *Client) WriteSession(out io.Writer) (int, error) {

	var size int
	for _, pkt := range client.Pending {
		s, err := pkt.WriteTo(out)
		size += s
		if err != nil {
			return size, err
		}
	}
	if client.server != nil {
		for _, subs := range client.subs {
			client.server.Unsubscribe(subs)
		}
	}
	return size, nil
}

// ReadSession reads the Session State of this client from the io.Reader.
func (client *Client) ReadSession(in io.Reader) (int, error) {
	var size int
	for true {
		packet, s, err := Read(in)
		size += s

		if err != nil {
			if err == io.EOF {
				return size, nil
			}
			return size, err
		}

		_, err = client.Send(packet)
		if err != nil {
			return size, err
		}
	}
	return size, nil
}
*/
