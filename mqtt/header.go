package mqtt

import (
	"errors"
	"io"
)

var (
	ReservedMessageType   = errors.New("Reserved message type.")
	IncompleteHeader      = errors.New("Incomplete header.")
	IncompleteMessage     = errors.New("Incomplete message.")
	MessageLengthExceeded = errors.New("Message length exceeds server maximum.")
	MessageLengthInvalid  = errors.New("Message length exceeds maximum.")
	// UnknownMessageType    = errors.New("Unknown mqtt message type.")
)

const (
	CONNECT     = 1
	CONNACK     = 2
	PUBLISH     = 3
	PUBACK      = 4
	PUBREC      = 5
	PUBREL      = 6
	PUBCOMP     = 7
	SUBSCRIBE   = 8
	SUBACK      = 9
	UNSUBSCRIBE = 10
	UNSUBACK    = 11
	PINGREQ     = 12
	PINGRESP    = 13
	DISCONNECT  = 14
)

var MessageTypes = [...]string{
	"",
	"CONNECT",
	"CONNACK",
	"PUBLISH",
	"PUBACK",
	"PUBREC",
	"PUBREL",
	"PUBCOMP",
	"SUBSCRIBE",
	"SUBACK",
	"UNSUBSCRIBE",
	"UNSUBACK",
	"PINGREQ",
	"PINGRESP",
	"DISCONNECT",
}

var MaxMessageLength = 15360

type FixedHeader struct {
	MType  byte
	Dup    bool
	QoS    byte
	Retain bool
	Length int
}

func (fh *FixedHeader) Read(reader io.Reader) (int, error) {

	var headBuf [1]byte
	n, err := reader.Read(headBuf[:])
	if n == 0 {
		return 0, err // connection closed
	}

	fh.MType = byte(headBuf[0] >> 4)
	fh.Dup = bool(headBuf[0]&0x8 != 0)
	fh.QoS = byte((headBuf[0] & 0x6) >> 1)
	fh.Retain = bool(headBuf[0]&0x1 != 0)

	if fh.MType == 0 || fh.MType == 15 {
		return 1, ReservedMessageType // reserved type
	}

	var multiplier int = 1
	var length int
	var d int

	for {
		d, err = reader.Read(headBuf[:])
		if d == 0 {
			return n, err // connection closed in header
		}
		n++

		length += int(headBuf[0]&127) * multiplier

		if length > MaxMessageLength {
			return n, MessageLengthExceeded // server maximum message size exceeded
		}

		if headBuf[0]&128 == 0 {
			break
		}

		if multiplier > 0x4000 {
			return n, MessageLengthInvalid // mqtt maximum message size exceeded
		}

		multiplier *= 128
	}

	fh.Length = length
	return n, nil
}

var (
	MessageTooLong = errors.New("Message too long.")
)

func (fh *FixedHeader) WriteTo(w io.Writer) (int, error) {

	var b byte
	b = fh.MType << 4
	if fh.Dup {
		b |= 0x80
	}
	b |= fh.QoS << 1
	if fh.Retain {
		b |= 0x01
	}

	if fh.Length < 0x80 {
		return w.Write([]byte{b, byte(fh.Length)})
	} else if fh.Length < 0x4000 {
		return w.Write([]byte{b, byte(fh.Length&127) | 0x80, byte(fh.Length >> 7)})
	} else if fh.Length < 0x200000 {
		return w.Write([]byte{b, byte(fh.Length&127) | 0x80, byte((fh.Length>>7)&127) | 0x80, byte(fh.Length >> 14)})
	}
	return 0, MessageTooLong
}
