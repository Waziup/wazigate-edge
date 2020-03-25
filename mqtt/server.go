package mqtt

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"strings"
	"sync"
)

// Reciever can get messages from subscriptions.
type Reciever interface {
	Publish(msg *Message) error
	ID() string
}

// The RecieverFunc type is an adapter to allow the use of
// ordinary functions as Recievres.
type RecieverFunc func(msg *Message) error

// Publish calls func.
func (f RecieverFunc) Publish(msg *Message) error {
	return f(msg)
}

func (f RecieverFunc) ID() string {
	return "<recieverFunc>"
}

var ErrServerClose = errors.New("server close")
var ErrSessionOvertake = errors.New("session overtaken")

/*
// Subscription
type Subscription interface {
	// Topic is the topic name this subscription belongs to.
	Topic() string
	// QoS is the subspcition QoS.
	QoS() byte
}
*/

type Sender interface {
	ID() string
}

// Server represents an MQTT server.
type Server interface {
	// Connect checks the ConnectAuth of a new client.
	Connect(client *Client, auth *ConnectAuth) ConnectCode
	// Close terminates the server an all client connection.
	Close() error
	// Serve makes the stream part of this server. This call is blocking and will read and write from the stream until the client or the server closes the connection.
	Serve(stream Stream)
	// Publish can be called from clients to emit new messages.
	Publish(sender Sender, msg *Message) int
	// Subscribe adds a new subscription to the topics tree.
	Subscribe(recv Reciever, topic string, qos byte) *Subscription
	// SubscribeAll adds a list of subscriptions to the topics tree.
	SubscribeAll(recv Reciever, topics []TopicSubscription) []*Subscription
	// Unsubscribe releases a subscription.
	Unsubscribe(subs ...*Subscription)
	// Disconnect removes a client that has diconnected.
	Disconnect(client *Client, reason error)
}

type LogLevel int

const (
	LogLevelErrors   LogLevel = -2
	LogLevelWarnings LogLevel = -1
	LogLevelNormal   LogLevel = 0
	LogLevelVerbose  LogLevel = 1
	LogLevelDebug             = 2
)

type server struct {
	topics      *topic
	topicsMutex sync.RWMutex

	auth Authenticate

	MaxPending int

	// sessions      map[string]*Client
	// sessionsMutex sync.Mutex

	log      *log.Logger
	LogLevel LogLevel
}

// Authenticate is a simple function to check a client and its authentication.
type Authenticate func(client *Client, auth *ConnectAuth) ConnectCode

// NewServer creates a new server.
func NewServer(auth Authenticate, log *log.Logger, ll LogLevel) Server {

	server := &server{
		auth:   auth,
		topics: newTopic(nil, ""),
		// sessions:   make(map[string]*Client),
		log:        log,
		LogLevel:   ll,
		MaxPending: MaxPending,
	}

	return server
}

func (server *server) Close() error {
	// server.sessionsMutex.Lock()
	// sessions := server.sessions
	// server.sessions = nil
	// for _, client := range sessions {
	// 	client.Disconnect()
	// }
	// server.sessionsMutex.Unlock()
	// for _, client := range sessions {
	// 	server.Disconnect(client, ErrServerClose)
	// }
	return nil
}

func (server *server) Connect(client *Client, auth *ConnectAuth) ConnectCode {
	if server.auth != nil {
		return server.auth(client, auth)
	}
	return CodeAccepted
}

func (server *server) Serve(stream Stream) {

	pkt, err := stream.ReadPacket()
	if err != nil {
		if server.log != nil && server.LogLevel >= LogLevelWarnings {
			server.log.Printf("Err Pre-connect client %v", err)
		}
		stream.Close()
		return
	}

	connectPkt, ok := pkt.(*ConnectPacket)
	if !ok {
		if server.log != nil && server.LogLevel >= LogLevelWarnings {
			server.log.Printf("Err Handshake")
		}
		stream.Close()
		return
	}

	id := connectPkt.ClientID

	if server.log != nil && server.LogLevel >= LogLevelDebug {
		server.log.Printf("%.24q > %v", id, connectPkt)
	}

	if !(connectPkt.Version == 4 && connectPkt.Protocol == "MQTT") &&
		!(connectPkt.Version == 3 && connectPkt.Protocol == "MQIsdp") {

		if server.log != nil && server.LogLevel >= LogLevelWarnings {
			server.log.Printf("Err Client Protocol 0x%x %q", connectPkt.Version, connectPkt.Protocol)
		}
		stream.WritePacket(ConnAck(CodeUnacceptableProtoV, false))
		return
	}

	client := &Client{
		id:     id,
		stream: stream,
		will:   connectPkt.Will,

		Server: server,

		LogLevel: server.LogLevel,
		log:      server.log,

		subscriptions: make(map[string]*Subscription),

		pending:    make(map[int]Packet),
		MaxPending: server.MaxPending,
	}

	if code := server.Connect(client, connectPkt.Auth); code != CodeAccepted {
		client.Send(ConnAck(code, false))
		if server.log != nil && server.LogLevel >= LogLevelWarnings {
			server.log.Printf("%.24q Rejected %s", id, code)
		}
		return
	}

	s := client.Server

	if server.log != nil && server.LogLevel >= LogLevelNormal {
		server.log.Printf("%.24q Connected Protocol 0x%x", id, connectPkt.Version)
	}

	connectPkt = nil

	// server.sessionsMutex.Lock()
	// oldClient := server.sessions[client.id]
	// server.sessions[client.id] = client
	// server.sessionsMutex.Unlock()

	// if oldClient != nil {
	// 	if server.log != nil && server.LogLevel >= LogLevelVerbose {
	// 		server.log.Printf("%.24q Session overtake", id)
	// 	}
	// 	oldClient.Disconnect()
	// 	s.Disconnect(oldClient, ErrSessionOvertake)
	// }

	client.Send(ConnAck(CodeAccepted, false))

	////////////////////

	var msg *Message

	for true {

		msg, err = client.Message()
		if err != nil {
			if server.log != nil && server.LogLevel >= LogLevelNormal {
				server.log.Printf("%.24q Err %v", id, err)
			}
			break
		}
		if msg == nil {
			break
		}
		if server.log != nil {
			if server.LogLevel >= LogLevelNormal {
				server.log.Printf("%.24q Message %q s:%d r:%v q:%d", id, msg.Topic, len(msg.Data), msg.Retain, msg.QoS)
			}
			if server.LogLevel >= LogLevelDebug {
				server.log.Printf("  Data: %q", msg.Data)
			}
		}
		hit := s.Publish(client, msg)
		if server.log != nil && server.LogLevel >= LogLevelVerbose {
			server.log.Printf("  Hit %d subscribers", hit)
		}
	}

	////////////////////

	if len(client.subscriptions) != 0 {
		if server.log != nil && server.LogLevel >= LogLevelDebug {
			server.log.Printf("%.24q Release subscriptions:", id)
		}
		for _, subs := range client.subscriptions {
			if server.log != nil && server.LogLevel >= LogLevelDebug {
				server.log.Printf("  %q qos:%d", subs.Topic(), subs.QoS())
			}
			s.Unsubscribe(subs)
		}
	}

	if err != nil {

		if will := client.Will(); will != nil {
			if server.log != nil && server.LogLevel >= LogLevelVerbose {
				server.log.Printf("%.24q Will %q s:%d r:%v q:%d", id, will.Topic, len(will.Data), will.Retain, will.QoS)
			}
			hit := s.Publish(client, will)
			if server.log != nil && server.LogLevel >= LogLevelVerbose {
				server.log.Printf("  Hit %d subscribers", hit)
			}
		}
	}

	// server.sessionsMutex.Lock()
	// oldClient = server.sessions[id]
	// if oldClient == client {
	// 	delete(server.sessions, id)
	// }
	// server.sessionsMutex.Unlock()

	// if oldClient == client {
	// 	client.Disconnect()
	// 	s.Disconnect(client, err)
	// }
}

func (server *server) Publish(sender Sender, msg *Message) int {

	valid, hasWildcards := checkTopic(msg.Topic)
	if !valid || hasWildcards {
		if server.log != nil && server.LogLevel >= LogLevelWarnings {
			server.log.Printf("Err Can not publish to %q", msg.Topic)
		}
		return 0
	}
	name := strings.Split(msg.Topic, "/")
	server.topicsMutex.RLock()
	hit := server.topics.publish(name, msg)
	server.topicsMutex.RUnlock()
	return hit
}

func (server *server) Subscribe(recv Reciever, topic string, qos byte) *Subscription {
	name := strings.Split(topic, "/")
	subs := newSubscription(recv, qos)
	server.topicsMutex.Lock()
	server.topics.subscribe(name, subs)
	server.topicsMutex.Unlock()

	if server.log != nil && server.LogLevel >= LogLevelNormal {
		server.log.Printf("%.24q Subscribed %q qos:%d", recv.ID(), topic, qos)
	}
	return subs
}

func (server *server) SubscribeAll(recv Reciever, topics []TopicSubscription) []*Subscription {
	subs := make([]*Subscription, len(topics))
	server.topicsMutex.Lock()

	for i, topic := range topics {
		name := strings.Split(topic.Name, "/")
		s := newSubscription(recv, topic.QoS)
		server.topics.subscribe(name, s)
		subs[i] = s
	}

	server.topicsMutex.Unlock()

	if server.log != nil && server.LogLevel >= LogLevelNormal {
		for i, topic := range topics {
			server.log.Printf("%.24q Subscribed %q qos:%d", recv.ID(), topic.Name, subs[i].QoS())
		}
	}
	return subs
}

func (server *server) Unsubscribe(subs ...*Subscription) {
	server.topicsMutex.Lock()
	for _, sub := range subs {
		sub.unsubscribe()
	}
	server.topicsMutex.Unlock()
}

func (server *server) Disconnect(client *Client, reason error) {
	if server.log != nil {
		if reason == nil {
			server.log.Printf("%.24q Disconnected", client.id)
		} else {
			server.log.Printf("%.24q Err Disconnected: %v", client.id, reason)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

// ListenAndServe listens at the give tcp address.
func ListenAndServe(addr string, server Server) error {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return Serve(listener, server)
}

// ListenAndServeTLS listens at the give tcp address.
func ListenAndServeTLS(addr string, config *tls.Config, server Server) error {

	listener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return err
	}

	return Serve(listener, server)
}

// Serve accepts listeners and hands them over to the server.
func Serve(listener net.Listener, server Server) error {

	if server == nil {
		server = NewServer(nil, nil, 0)
	}

	for {
		conn, err := listener.Accept()
		if err == nil {
			go server.Serve(NewStream(conn, 0))
		} else {
			return err
		}
	}
}
