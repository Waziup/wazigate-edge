package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
)

type Reciever interface {
	Publish(client *Client, msg *Message) error
}

type Authenticator interface {
	Authenticate(client *Client, auth *ConnectAuth) byte
}

const (
	actionCreate = 1
	actionRemove = 2
)

type subscriptionChange struct {
	action int
	subs   *Subscription
	topic  string
}

type publication struct {
	sender *Client
	msg    *Message
}

type connection struct {
	stream io.ReadWriteCloser
	packet *ConnectPacket
	server Server
}

// The DefaultServer is as MQTT Server implementation with topics, subscriptions and all that.
// It will accept all clients, so use the Auth interface for custom permission checking.
type DefaultServer struct {
	sigclose     chan (struct{})
	subsQueue    chan subscriptionChange
	pubQueue     chan publication
	connectQueue chan connection

	// Use a custom Authenticator to check each client for permissions.
	Auth Authenticator
	// The topics tree, holding all the subscriptions.
	Topics *Topic
}

// An interface for use as a MQTT server, that can recieve Publish-Messages,
// authenticate clients and manage subscriptions and unsubscriptions.
type Server interface {
	Reciever
	Connect(client *Client, auth *ConnectAuth) byte
	PreConnect(stream io.ReadWriteCloser, connect *ConnectPacket, server Server)
	Disconnect(client *Client, err error)
	Subscribe(recv Reciever, topic string, qos byte) *Subscription
	Unsubscribe(subs *Subscription)
}

// Create a new MQTT server.
func NewServer() *DefaultServer {
	server := &DefaultServer{
		sigclose:     make(chan struct{}),
		subsQueue:    make(chan subscriptionChange),
		pubQueue:     make(chan publication),
		connectQueue: make(chan connection),
		Topics:       NewTopic(nil, ""),
	}

	go server.schedule()
	return server
}

func (server *DefaultServer) Publish(client *Client, msg *Message) error {
	server.pubQueue <- publication{client, msg}
	return nil
}

func (server *DefaultServer) Connect(client *Client, auth *ConnectAuth) byte {
	if server.Auth != nil {
		return server.Auth.Authenticate(client, auth)
	}
	return CodeAccepted
}

func (server *DefaultServer) PreConnect(stream io.ReadWriteCloser, connect *ConnectPacket, s Server) {
	server.connectQueue <- connection{stream, connect, s}
}

func (server *DefaultServer) Disconnect(client *Client, err error) {
}

func (server *DefaultServer) Subscribe(recv Reciever, topic string, qos byte) *Subscription {

	subs := NewSubscription(recv, qos)
	server.subsQueue <- subscriptionChange{actionCreate, subs, topic}
	return subs
}

func (server *DefaultServer) Unsubscribe(subs *Subscription) {

	server.subsQueue <- subscriptionChange{actionRemove, subs, ""}
}

func (server *DefaultServer) Close() {

	close(server.sigclose)
}

////////////////////////////////////////////////////////////////////////////////

type closer interface {
	Close(err error)
}

var serverClosed = errors.New("Server closed.")

func (server *DefaultServer) schedule() {

RUN:
	for {
		select {
		case <-server.sigclose:

			close(server.subsQueue)
			close(server.pubQueue)
			close(server.connectQueue)

			subs := server.Topics.Find([]string{"$SYS", "all"})
			for ; subs != nil; subs = subs.Next {
				if c, ok := subs.Recv.(closer); ok {
					c.Close(serverClosed)
				}
			}

			break RUN

		case evt := <-server.subsQueue:

			switch evt.action {
			case actionCreate:
				server.Topics.Subscribe(strings.Split(evt.topic, "/"), evt.subs)

			case actionRemove:
				evt.subs.Unsubscribe()
			}

			// log.Println("[DEBUG] Topics:", server.Topics)

		case pub := <-server.pubQueue:

			if strings.HasPrefix(pub.msg.Topic, "$SYS/") {
				// we don't do that here..
				return
			}

			// n := len(pub.msg.Data)
			// if n > 30 {
			// 	n = 30
			// }

			//if svr.debug {
			//	log.Printf("[DEBUG] Publish: %q: %q", msg.Topic, string(msg.Buf[:n]))
			//}
			topic := strings.Split(pub.msg.Topic, "/")
			server.Topics.Publish(topic, pub.sender, pub.msg)

		case conn := <-server.connectQueue:

			var client *Client
			subs := server.Topics.Find([]string{"$SYS", "all"})
			for ; subs != nil; subs = subs.Next {
				if recv, ok := subs.Recv.(*Client); ok {
					if conn.packet.ClientId == recv.Id {
						if conn.packet.CleanSession {
							recv.cleanup()
						} else {
							client = recv
						}
						break
					}
				}
			}

			if client == nil {
				client = &Client{
					pending: make(map[int]Packet),
					// queuePacket: make(chan Packet),
					// queueWriter: make(chan io.Writer),
					Closer:    conn.stream,
					Server:    conn.server,
					subs:      make(map[string]*Subscription),
					State:     StateConnecting,
					Context:   context.Background(),
					sigServed: make(chan struct{}),
				}

				if client.connect(conn.packet, conn.stream) == nil {

					client.sysall = NewSubscription(client, 0)
					server.Topics.Subscribe([]string{"$SYS", "all"}, client.sysall)

					go client.serveWriter(conn.stream)
					go client.serveReader(conn.stream)
				}

			} else {

				client.sigServed = make(chan struct{})
				client.Closer = conn.stream
				client.State = StateConnecting
				if client.connect(conn.packet, conn.stream) == nil {

					client.serveWriter(conn.stream)
					go client.serveReader(conn.stream)
				}
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func ListenAndServe(addr string, server Server) error {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return Serve(listener, server)
}

func ListenAndServeTLS(addr string, config *tls.Config, server Server) error {

	listener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return err
	}

	return Serve(listener, server)
}

func Serve(listener net.Listener, server Server) error {

	if server == nil {
		server = NewServer()
	}

	for {
		conn, err := listener.Accept()
		if err == nil {
			go ServeConn(conn, server)
		} else {
			return err
		}
	}
}

func ServeConn(stream io.ReadWriteCloser, server Server) {

	if server == nil {
		server = NewServer()
	}

	packet, _, err := Read(stream)
	if err != nil {
		stream.Close()
		return
	}

	if connect, ok := packet.(*ConnectPacket); ok {

		server.PreConnect(stream, connect, server)
	} else {

		stream.Close()
	}
}
