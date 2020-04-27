package mqtt

import (
	"io"
	"time"
)

// Conn is a client connection (like TCP or WebSocket).
type Conn interface {
	io.ReadWriteCloser
	SetReadDeadline(t time.Time) error
}

// Stream is a Read/Write/Close interface for MQTT Packet streams.
type Stream interface {
	ReadPacket() (Packet, error)
	WritePacket(Packet) error
	Close() error
}

type stream struct {
	conn    Conn
	timeout time.Duration
	version byte
}

// NewStream creates a Stream that can read and write MQTT Packets
// from and to the io.ReadWriteCloser.
func NewStream(conn Conn, tout time.Duration) Stream {
	return &stream{conn, tout, 0}
}

func (s *stream) ReadPacket() (pkt Packet, err error) {
	if s.timeout != 0 {
		s.conn.SetReadDeadline(time.Now().Add(s.timeout))
	}
	pkt, _, err = Read(s.conn, s.version)
	if pkt != nil {
		if connectPkt, ok := pkt.(*ConnectPacket); ok {
			s.version = connectPkt.Version
		}
	}
	return
}

func (s *stream) WritePacket(pkt Packet) (err error) {
	_, err = pkt.WriteTo(s.conn, s.version)
	return err
}

func (s *stream) Version() byte {
	return s.version
}

func (s *stream) Close() error {
	return s.conn.Close()
}
