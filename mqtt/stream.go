package mqtt

import "io"

// Stream is a Read/Write/Close interface for MQTT Packet streams.
type Stream interface {
	ReadPacket() (Packet, error)
	WritePacket(Packet) error
	Close() error
}

type stream struct {
	rwc io.ReadWriteCloser
}

// NewStream creates a Stream that can read and write MQTT Packets
// from and to the io.ReadWriteCloser.
func NewStream(rwc io.ReadWriteCloser) Stream {
	return &stream{rwc}
}

func (s stream) ReadPacket() (pkt Packet, err error) {
	pkt, _, err = Read(s.rwc)
	return
}

func (s stream) WritePacket(pkt Packet) (err error) {
	_, err = pkt.WriteTo(s.rwc)
	return err
}

func (s stream) Close() error {
	return s.rwc.Close()
}
