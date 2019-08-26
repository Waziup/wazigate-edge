package mqtt

import (
	"net"
	"time"
)

// Stream is a Read/Write/Close interface for MQTT Packet streams.
type Stream interface {
	ReadPacket() (Packet, error)
	WritePacket(Packet) error
	Close() error
}

type stream struct {
	conn    net.Conn
	timeout time.Duration
}

// NewStream creates a Stream that can read and write MQTT Packets
// from and to the io.ReadWriteCloser.
func NewStream(conn net.Conn, tout time.Duration) Stream {
	return &stream{conn, tout}
}

func (s stream) ReadPacket() (pkt Packet, err error) {
	s.conn.SetReadDeadline(time.Now().Add(s.timeout))
	pkt, _, err = Read(s.conn)
	return
}

func (s stream) WritePacket(pkt Packet) (err error) {
	_, err = pkt.WriteTo(s.conn)
	return err
}

func (s stream) Close() error {
	return s.conn.Close()
}
