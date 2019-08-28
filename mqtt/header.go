package mqtt

import (
	"errors"
	"io"
)

var (
	errReservedPacketType = errors.New("reserved message type")
	// errIncompleteHeader      = errors.New("incomplete header")
	errMessageLengthExceeded = errors.New("message length exceeds server maximum")
	errMessageLengthInvalid  = errors.New("message length exceeds maximum")
	// UnknownPacketType    = errors.New("Unknown mqtt message type.")
)

// FixedHeader is used with every package.
type FixedHeader struct {
	PacketType PacketType
	Dup        bool
	QoS        byte
	Retain     bool
	Length     int
}

// Read reads a FixedHeader from an io.Reader.
func (fh *FixedHeader) Read(reader io.Reader) (int, error) {

	var headBuf [1]byte
	n, err := reader.Read(headBuf[:])
	if n == 0 {
		return 0, err
	}

	fh.PacketType = PacketType(headBuf[0] >> 4)
	fh.Dup = bool(headBuf[0]&0x8 != 0)
	fh.QoS = byte((headBuf[0] & 0x6) >> 1)
	fh.Retain = bool(headBuf[0]&0x1 != 0)

	if fh.PacketType == 0 || fh.PacketType == 15 {
		return 1, errReservedPacketType
	}

	var multiplier = 1
	var length int
	var d int

	for {
		d, err = reader.Read(headBuf[:])
		if d == 0 {
			return n, err
		}
		n++

		length += int(headBuf[0]&127) * multiplier

		// if length > MaxMessageLength {
		// 	return n, errMessageLengthExceeded
		// }

		if headBuf[0]&128 == 0 {
			break
		}

		if multiplier > 0x4000 {
			return n, errMessageLengthInvalid
		}

		multiplier *= 128
	}

	fh.Length = length
	return n, nil
}

// Read reads a FixedHeader from an io.Reader.
func (fh *FixedHeader) ReadBuffer(buf []byte) (int, error) {

	if len(buf) < 2 {
		return 0, errIncompleteHeader
	}

	fh.PacketType = PacketType(buf[0] >> 4)
	fh.Dup = bool(buf[0]&0x8 != 0)
	fh.QoS = byte((buf[0] & 0x6) >> 1)
	fh.Retain = bool(buf[0]&0x1 != 0)

	if fh.PacketType == 0 || fh.PacketType == 15 {
		return 0, errReservedPacketType
	}

	buf = buf[1:]
	n := 1

	var multiplier = 1
	var length int

	if buf[0] == 0 {
		fh.Length = 0
		return 2, nil
	}

	for {
		n++
		length += int(buf[0]&127) * multiplier

		if buf[0]&128 == 0 {
			break
		}

		if multiplier > 0x4000 {
			return n, errMessageLengthInvalid
		}

		multiplier *= 128
	}

	fh.Length = length
	return n, nil
}

var (
	errMessageTooLong = errors.New("message too long")
)

// WriteTo writes the FixedHeader to an io.Writer.
func (fh *FixedHeader) WriteTo(w io.Writer) (int, error) {

	var b byte
	b = byte(fh.PacketType) << 4
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
	return 0, errMessageTooLong
}
