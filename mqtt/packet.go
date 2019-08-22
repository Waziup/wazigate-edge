package mqtt

import (
	"errors"
	"fmt"
	"io"
)

var (
	errIncompleteMessage = errors.New("incomplete message")
	errIncompleteHeader  = errors.New("incomplete header")
)

type Packet interface {
	WriteTo(w io.Writer) (len int, err error)
	Header() *FixedHeader
	String() string
}

// Read reads a packet from an io.Reader.
func Read(reader io.Reader) (Packet, int, error) {

	var fh FixedHeader
	var n int
	var err error
	if n, err = fh.Read(reader); err != nil {
		return nil, n, err
	}

	buf := make([]byte, fh.Length)

	d, err := io.ReadFull(reader, buf)
	n += d
	if err != nil {
		return nil, n, err
	}

	pkt, err := readPacket(&fh, buf)
	return pkt, n, err
}

var errInvalidBuffer = errors.New("invalid buffer length")

// ReadBuffer reads a packet from the buffer.
func ReadBuffer(buf []byte) (Packet, error) {

	var fh FixedHeader
	n, err := fh.ReadBuffer(buf)
	if err != nil {
		return nil, err
	}
	buf = buf[n:]
	if len(buf) != fh.Length {
		return nil, fmt.Errorf("invalid buffer length: header(%d) != buffer(%d)", fh.Length, len(buf))
	}
	return readPacket(&fh, buf)
}
func readPacket(fh *FixedHeader, buf []byte) (Packet, error) {
	switch fh.PacketType {
	case CONNECT:
		return readConnect(fh, buf)
	case CONNACK:
		return readConnAck(fh, buf)
	case SUBSCRIBE:
		return readSubscribe(fh, buf)
	case SUBACK:
		return readSubAck(fh, buf)
	case UNSUBSCRIBE:
		return readUnsubscribe(fh, buf)
	case UNSUBACK:
		return readUnsubAck(fh, buf)
	case PUBLISH:
		return readPublish(fh, buf)
	case PUBACK:
		return readPubAck(fh, buf)
	case PUBREL:
		return readPubRel(fh, buf)
	case PUBREC:
		return readPubRec(fh, buf)
	case PUBCOMP:
		return readPubComp(fh, buf)
	case PINGREQ:
		return readPingReq(fh, buf)
	case PINGRESP:
		return readPingResp(fh, buf)
	case DISCONNECT:
		return readDisconnect(fh, buf)
	default:
		return nil,  fmt.Errorf("unknown MQTT message type: %d", fh.PacketType)
	}
}

////////////////////////////////////////////////////////////////////////////////

// ConnectPacket is a MQTT CONNECT control packet.
type ConnectPacket struct {
	// Header
	header *FixedHeader
	// Variable Header
	Protocol       string
	Version        byte
	CleanSession   bool
	KeepAliveTimer int
	// Payload
	ClientID string
	Will     *Message
	Auth     *ConnectAuth
}

// String stringifies the packet in a human readable format.
func (pkt *ConnectPacket) String() string {
	str := fmt.Sprintf("CONNECT (%s %v) %.24q cs:%v kat:%d", pkt.Protocol, pkt.Version, pkt.ClientID, pkt.CleanSession, pkt.KeepAliveTimer)
	if pkt.Auth != nil {
		str += fmt.Sprintf("\n  auth: %s@%s", pkt.Auth.Username, pkt.Auth.Password)
	}
	if pkt.Will != nil {
		str += fmt.Sprintf("\n  will: %q [%d] qos:%v r:%v", pkt.Will.Topic, len(pkt.Will.Data), pkt.Will.QoS, pkt.Will.Retain)
	}
	return str
}

// WriteTo writes the packet to the io.Writer.
func (pkt *ConnectPacket) WriteTo(w io.Writer) (len int, err error) {

	var d int
	len, err = pkt.Header().WriteTo(w)
	if err != nil {
		return
	}

	d, err = writeString(w, pkt.Protocol)
	len += d
	if err != nil {
		return
	}

	d, err = w.Write([]byte{pkt.Version})
	len += d
	if err != nil {
		return
	}

	var flag byte
	if pkt.CleanSession {
		flag |= 0x02
	}
	if pkt.Will != nil {
		flag |= 0x04
		flag |= pkt.Will.QoS << 3
		if pkt.Will.Retain {
			flag |= 0x20
		}
	}
	if pkt.Auth != nil {
		flag |= 0x40 // Username
		flag |= 0x80 // PAssword
	}

	d, err = w.Write([]byte{flag})
	len += d
	if err != nil {
		return
	}

	d, err = writeInt(w, pkt.KeepAliveTimer)
	len += d
	if err != nil {
		return
	}

	d, err = writeString(w, pkt.ClientID)
	len += d
	if err != nil {
		return
	}

	if pkt.Will != nil {
		d, err = writeString(w, pkt.Will.Topic)
		len += d
		if err != nil {
			return
		}
		d, err = writeBytes(w, pkt.Will.Data)
		len += d
		if err != nil {
			return
		}
	}

	if pkt.Auth != nil {
		d, err = writeString(w, pkt.Auth.Username)
		len += d
		if err != nil {
			return
		}
		d, err = writeString(w, pkt.Auth.Password)
		len += d
		if err != nil {
			return
		}
	}

	return
}

// Header returns the FixedHeader of this packet.
func (pkt *ConnectPacket) Header() *FixedHeader {
	// Protocol + Version + Flags + KeepAlive + ClientID
	length := 2 + len(pkt.Protocol) + 1 + 1 + 2 + 2 + len(pkt.ClientID)
	if pkt.Will != nil {
		length += 2 + len(pkt.Will.Topic) + 2 + len(pkt.Will.Data)
	}
	if pkt.Auth != nil {
		length += 2 + len(pkt.Auth.Username) + 2 + len(pkt.Auth.Password)
	}
	pkt.header.Length = length
	return pkt.header
}

// Connect creates a new MQTT CONNECT control packet.
func Connect(protocol string, version byte, cleanSession bool, keepAliveTimer int, clientID string, will *Message, auth *ConnectAuth) *ConnectPacket {
	return &ConnectPacket{
		header: &FixedHeader{
			PacketType: CONNECT,
		},
		Protocol:       protocol,
		Version:        version,
		CleanSession:   cleanSession,
		KeepAliveTimer: keepAliveTimer,
		ClientID:       clientID,
		Will:           will,
		Auth:           auth,
	}
}

func readConnect(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &ConnectPacket{header: fh}

	l, protocol := readString(buf)
	if l == 0 {
		return pkt, errIncompleteHeader
	}
	pkt.Protocol = protocol
	buf = buf[l:]

	if len(buf) < 1 {
		return pkt, errIncompleteMessage
	}
	pkt.Version = buf[0]
	buf = buf[1:]

	if len(buf) < 1 {
		return pkt, errIncompleteMessage
	}
	connFlags := buf[0]

	pkt.CleanSession = connFlags&0x02 != 0
	willFlag := connFlags&0x04 != 0
	willQoS := connFlags & 0x18 >> 3
	willRetain := connFlags&0x20 != 0
	passwordFlag := connFlags&0x40 != 0
	usernameFlag := connFlags&0x80 != 0

	buf = buf[1:]

	//

	if len(buf) < 2 {
		return pkt, errIncompleteMessage
	}
	pkt.KeepAliveTimer = int(buf[0])<<8 + int(buf[1])
	// TODO set SetDeadline() to conn

	buf = buf[2:]

	//

	l, pkt.ClientID = readString(buf)
	if l == 0 {
		return pkt, errIncompleteMessage
	}
	buf = buf[l:]

	//

	if willFlag {

		pkt.Will = &Message{}

		pkt.Will.Retain = willRetain
		pkt.Will.QoS = willQoS

		l, pkt.Will.Topic = readString(buf)
		if l == 0 {
			return pkt, errIncompleteMessage
		}
		buf = buf[l:]

		l, pkt.Will.Data = readBytes(buf)
		if l == 0 {
			return pkt, errIncompleteMessage
		}

		buf = buf[l:]
	}

	//

	if usernameFlag {

		pkt.Auth = &ConnectAuth{}

		l, pkt.Auth.Username = readString(buf)
		if l == 0 {
			return pkt, errIncompleteMessage
		}
		buf = buf[l:]

		if passwordFlag {

			l, pkt.Auth.Password = readString(buf)
			if l != 0 {
				buf = buf[l:]
			}
		}
	}

	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

type ConnectCode byte

const (
	CodeAccepted ConnectCode = iota
	CodeUnacceptableProtoV
	CodeIDentifierRejected
	CodeServerUnavaliable
	CodeBatUserOrPassword
	CodeNotAuthorized
)

var codeNames = [...]string{
	"Connection accepted.",
	"The Server does not support the level of the MQTT protocol requested by the Client.",
	"The Client identifier is correct UTF-8 but not allowed by the Server.",
	"The Network Connection has been made but the MQTT service is unavailable.",
	"The data in the user name or password is malformed.",
	"The Client is not authorized to connect.",
}

func (c ConnectCode) String() string {
	if c >= 0 && int(c) < len(codeNames) {
		return codeNames[c]
	}
	return fmt.Sprintf("<code %d>", c)
}

// ConnAckPacket is a MQTT CONNACK control packet.
type ConnAckPacket struct {
	header         *FixedHeader
	SessionPresent bool
	Code           ConnectCode
}

// String stringifies the packet in a human readable format.
func (pkt *ConnAckPacket) String() string {
	return fmt.Sprintf("CONNACK code:%d sess:%v", pkt.Code, pkt.SessionPresent)
}

// Header returns the FixedHeader of this packet.
func (pkt *ConnAckPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *ConnAckPacket) WriteTo(w io.Writer) (n int, err error) {
	var d int
	n, err = pkt.header.WriteTo(w)
	if err != nil {
		return
	}
	var sp byte = 0x00
	if pkt.SessionPresent {
		sp = 0x01
	}
	d, err = w.Write([]byte{sp, byte(pkt.Code)})
	n += d
	return
}

// ConnAck creates a new MQTT CONNACK control packet.
func ConnAck(code ConnectCode, sessPresent bool) *ConnAckPacket {
	return &ConnAckPacket{
		header: &FixedHeader{
			PacketType: CONNACK,
			Length:     2,
		},
		Code:           code,
		SessionPresent: sessPresent,
	}
}

func readConnAck(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &ConnAckPacket{header: fh}
	if len(buf) < 2 {
		return pkt, errIncompleteMessage
	}
	pkt.Code = ConnectCode(buf[1])
	if buf[0] == 0x01 {
		pkt.SessionPresent = true
	}
	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// QoS is the "Quality of Service"
type QoS byte

const (
	// AtMostOnce is QoS level 0
	AtMostOnce QoS = 0
	// AtLeasOnce is QoS level 1
	AtLeasOnce QoS = 1
	// ExactlyOnce is QoS level 2
	ExactlyOnce QoS = 2
	// Failure indicates a QoS failure.
	Failure QoS = 128
)

// A TopicSubscription is used in Subscribe packets.
type TopicSubscription struct {
	Name string // Topic Name
	QoS  byte   // Subscriptions QoS
}

// SubscribePacket is a MQTT SUBSCRIBE control packet.
type SubscribePacket struct {
	// Header
	header *FixedHeader
	// Variable Header
	ID int // Message ID
	// Payload
	Topics []TopicSubscription // List of Subscriptions
}

// String stringifies the packet in a human readable format.
func (pkt *SubscribePacket) String() string {
	if len(pkt.Topics) == 0 {
		return fmt.Sprintf("SUBSCRIBE mid:%d <no topics>", pkt.ID)
	}
	str := fmt.Sprintf("SUBSCRIBE mid:%d", pkt.ID)
	for _, topic := range pkt.Topics {
		str += fmt.Sprintf("\n  %q qos:%d", topic.Name, topic.QoS)
	}
	return str
}

// Subscribe creates a new MQTT SUBSCRIBE control packet.
func Subscribe(id int, topics []TopicSubscription) *SubscribePacket {
	return &SubscribePacket{
		header: &FixedHeader{
			PacketType: SUBSCRIBE,
			QoS:        0x01,
		},
		ID:     id,
		Topics: topics,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *SubscribePacket) Header() *FixedHeader {
	// MessageID + Topics*(Topic Length + QoS)
	length := 2 + len(pkt.Topics)*(2+1)
	for _, topic := range pkt.Topics {
		length += len(topic.Name)
	}
	pkt.header.Length = length
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *SubscribePacket) WriteTo(w io.Writer) (len int, err error) {
	var d int
	len, err = pkt.Header().WriteTo(w)
	if err != nil {
		return
	}
	d, err = writeInt(w, pkt.ID)
	len += d
	if err != nil {
		return
	}
	for _, topic := range pkt.Topics {
		d, err = writeString(w, topic.Name)
		len += d
		if err != nil {
			return
		}
		d, err = w.Write([]byte{topic.QoS})
		len += d
		if err != nil {
			return
		}
	}
	return
}

func readSubscribe(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &SubscribePacket{header: fh}

	if len(buf) < 2 {
		return pkt, errIncompleteMessage
	}

	pkt.ID = int(buf[0])<<8 + int(buf[1])
	buf = buf[2:]

	var n int
	// count how many topics
	for i, l := 0, len(buf); i != l; n++ {
		// Lenght MSB + Lenght LSB + 2 byte Length + 1 byte QoS
		i += (int(buf[i]) << 8) + int(buf[i+1]) + 2 + 1
		if i > l {
			return pkt, errIncompleteMessage
		}
	}

	pkt.Topics = make([]TopicSubscription, n)
	n = 0
	for len(buf) != 0 {
		l, topic := readString(buf)
		qos := buf[l] & 0x03
		pkt.Topics[n] = TopicSubscription{topic, qos}
		buf = buf[l+1:]

		// grantedQos
		//body[s] = conn.Subscribe(topic, qos)
		n++
	}

	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// UnsubscribePacket is a MQTT UNSUBSCRIBE control packet.
type UnsubscribePacket struct {
	// Header
	header *FixedHeader
	// Variable Header
	ID int // Message ID
	// Payload
	Topics []string // List of Topics to unsubscribe
}

// String stringifies the packet in a human readable format.
func (pkt *UnsubscribePacket) String() string {
	if len(pkt.Topics) == 0 {
		return fmt.Sprintf("UNSUBSCRIBE mid:%d <no topics>", pkt.ID)
	}
	str := fmt.Sprintf("UNSUBSCRIBE mid:%d", pkt.ID)
	for _, topic := range pkt.Topics {
		str += fmt.Sprintf("\n  %q", topic)
	}
	return str
}

// Unsubscribe creates a new MQTT UNSUBSCRIBE control packet.
func Unsubscribe(id int, topics []string) *UnsubscribePacket {
	return &UnsubscribePacket{
		header: &FixedHeader{
			PacketType: UNSUBSCRIBE,
			QoS:        0x01,
		},
		ID:     id,
		Topics: topics,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *UnsubscribePacket) Header() *FixedHeader {
	// MessageID + Topics*(Topic Length)
	length := 2 + len(pkt.Topics)*(2)
	for _, topic := range pkt.Topics {
		length += len(topic)
	}
	pkt.header.Length = length
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *UnsubscribePacket) WriteTo(w io.Writer) (n int, err error) {
	var d int
	n, err = pkt.Header().WriteTo(w)
	if err != nil {
		return
	}
	d, err = writeInt(w, pkt.ID)
	n += d
	if err != nil {
		return
	}
	for _, topic := range pkt.Topics {
		d, err = writeString(w, topic)
		n += d
		if err != nil {
			return
		}
	}
	return
}

func readUnsubscribe(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &UnsubscribePacket{header: fh}

	if len(buf) < 2 {
		return pkt, errIncompleteMessage
	}

	pkt.ID = int(buf[0])<<8 + int(buf[1])
	buf = buf[2:]

	var n int
	// count how many topics
	for i, l := 0, len(buf); i != l; n++ {
		// Lenght MSB + Lenght LSB + 2 byte Length
		i += (int(buf[i]) << 8) + int(buf[i+1]) + 2
		if i > l {
			return pkt, errIncompleteMessage
		}
	}

	pkt.Topics = make([]string, n)
	n = 0
	for len(buf) != 0 {
		l, topic := readString(buf)
		pkt.Topics[n] = topic
		buf = buf[l:]
		n++
	}

	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// SubAckPacket is a MQTT SUBACK control packet.
type SubAckPacket struct {
	// Header
	header *FixedHeader
	// Variable Header
	ID int // Message ID
	// Payload
	Topics  []TopicSubscription // List of Subscriptions
	Failure byte                // Failure indication
}

// String stringifies the packet in a human readable format.
func (pkt *SubAckPacket) String() string {
	if len(pkt.Topics) == 0 {
		return fmt.Sprintf("SUBACK mid:%d <no topics>", pkt.ID)
	}
	str := fmt.Sprintf("SUBACK mid:%d", pkt.ID)
	for _, topic := range pkt.Topics {
		str += fmt.Sprintf("\n  %q qos:%d", topic.Name, topic.QoS)
	}
	return str
}

func SubAck(id int, topics []TopicSubscription, failure byte) *SubAckPacket {
	return &SubAckPacket{
		header: &FixedHeader{
			PacketType: SUBACK,
		},
		ID:      id,
		Topics:  topics,
		Failure: failure,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *SubAckPacket) Header() *FixedHeader {
	// MessageID + Topics*(granted QoS) + Failure
	length := 2 + len(pkt.Topics) + 1
	pkt.header.Length = length
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *SubAckPacket) WriteTo(w io.Writer) (n int, err error) {
	var d int
	n, err = pkt.Header().WriteTo(w)
	if err != nil {
		return
	}
	d, err = writeInt(w, pkt.ID)
	n += d
	if err != nil {
		return
	}
	for _, topic := range pkt.Topics {
		d, err = w.Write([]byte{topic.QoS})
		n += d
		if err != nil {
			return
		}
	}
	w.Write([]byte{pkt.Failure})
	return
}

var InvalidQoS = errors.New("Invalid QoS.")

func readSubAck(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &SubAckPacket{header: fh}

	if len(buf) < 2 {
		return pkt, errIncompleteMessage
	}

	pkt.ID = int(buf[0])<<8 + int(buf[1])
	buf = buf[2:]

	n := len(buf) - 1
	pkt.Topics = make([]TopicSubscription, n)
	for i := 0; i < n; i++ {
		pkt.Topics[i].QoS = buf[i]
		if (buf[i] != 0x00) && (buf[i] != 0x01) && (buf[i] != 0x02) && (buf[i] != 0x80) {
			return pkt, InvalidQoS
		}
	}
	pkt.Failure = buf[len(buf)-1]

	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// UnsubAckPacket is a MQTT UNSUBACK control packet.
type UnsubAckPacket struct {
	// Header
	header *FixedHeader
	// Variable Header
	ID int // Message ID
}

// String stringifies the packet in a human readable format.
func (pkt *UnsubAckPacket) String() string {
	return fmt.Sprintf("UNSUBACK mid:%d", pkt.ID)
}

// UnsubAck creates a new MQTT UNSUCKACK control packet
func UnsubAck(id int) *UnsubAckPacket {
	return &UnsubAckPacket{
		header: &FixedHeader{
			PacketType: UNSUBACK,
			Length:     2,
		},
		ID: id,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *UnsubAckPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *UnsubAckPacket) WriteTo(w io.Writer) (n int, err error) {
	var d int
	n, err = pkt.Header().WriteTo(w)
	if err != nil {
		return
	}
	d, err = writeInt(w, pkt.ID)
	n += d
	return
}

func readUnsubAck(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &UnsubAckPacket{header: fh}

	if len(buf) < 2 {
		return pkt, errIncompleteMessage
	}

	pkt.ID = int(buf[0])<<8 + int(buf[1])
	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// PublishPacket is a MQTT PUBLISH packet.
type PublishPacket struct {
	// Header
	header *FixedHeader
	// Variable Header
	Topic string // Publish Topic
	ID    int    // Message ID
	// Payload
	Data []byte
}

// String stringifies the packet in a human readable format.
func (pkt *PublishPacket) String() string {
	return fmt.Sprintf("PUBLISH mid:%d t:%q l:%d qos:%d", pkt.ID, pkt.Topic, len(pkt.Data), pkt.header.QoS)
}

// Publish creates a new MQTT PUBLISH control packet.
func Publish(id int, msg *Message) *PublishPacket {
	return &PublishPacket{
		header: &FixedHeader{
			PacketType: PUBLISH,
			QoS:        msg.QoS,
			Retain:     msg.Retain,
		},
		ID:    id,
		Topic: msg.Topic,
		Data:  msg.Data,
	}
}

// Message returns the Message that this Publish packets transports.
func (pkt *PublishPacket) Message() *Message {
	return &Message{
		Topic:  pkt.Topic,
		QoS:    pkt.header.QoS,
		Data:   pkt.Data,
		Retain: pkt.header.Retain,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *PublishPacket) Header() *FixedHeader {
	//       Topic                    + Data
	length := 2 + len(pkt.Topic) + len(pkt.Data)
	if pkt.header.QoS > 0 {
		length += 2 // Message ID field
	}
	pkt.header.Length = length
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *PublishPacket) WriteTo(w io.Writer) (n int, err error) {

	var d int
	n, err = pkt.Header().WriteTo(w)

	d, err = writeString(w, pkt.Topic)
	if err != nil {
		return
	}
	n += d
	if pkt.header.QoS > 0 {
		d, err = writeInt(w, pkt.ID)
		if err != nil {
			return
		}
		n += d
	}
	d, err = w.Write(pkt.Data)
	n += d
	return
}

func readPublish(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &PublishPacket{header: fh}

	if len(buf) < 2 {
		return pkt, errIncompleteMessage
	}
	var l int
	l, pkt.Topic = readString(buf)
	if l == 0 {
		return pkt, errIncompleteMessage
	}
	buf = buf[l:]

	if fh.QoS == 0 { // QoS 0

		pkt.Data = buf

	} else { // QoS 1 or 2

		if len(buf) < 2 {
			// missinge message id
			return pkt, errIncompleteMessage
		}
		pkt.ID = int(buf[0])<<8 + int(buf[1])
		buf = buf[2:]
		pkt.Data = buf
	}
	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// PubAckPacket is a MQTT PUBACK control packet.
type PubAckPacket struct {
	header *FixedHeader
	ID     int
}

// String stringifies the packet in a human readable format.
func (pkt *PubAckPacket) String() string {
	return fmt.Sprintf("PUBACK mid:%d", pkt.ID)
}

// PubAck creates a new MQTT PUBACK packet.
func PubAck(id int) *PubAckPacket {
	return &PubAckPacket{
		header: &FixedHeader{
			PacketType: PUBACK,
			Length:     2, // for Message ID field
		},
		ID: id,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *PubAckPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *PubAckPacket) WriteTo(w io.Writer) (n int, err error) {
	var d int
	n, err = pkt.header.WriteTo(w)
	if err != nil {
		return
	}
	d, err = writeInt(w, pkt.ID)
	n += d
	return
}

func readPubAck(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &PubAckPacket{header: fh}

	if len(buf) < 2 {
		// Missing message id field
		return pkt, errIncompleteMessage
	}
	pkt.ID = int(buf[0])<<8 + int(buf[1])
	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// PubRelPacket is a MQTT PUBREL control packet.
type PubRelPacket struct {
	header *FixedHeader
	ID     int
}

// String stringifies the packet in a human readable format.
func (pkt *PubRelPacket) String() string {
	return fmt.Sprintf("PUBREL mid:%d", pkt.ID)
}

// PubRel creates a new MQTT PUBREL packet.
func PubRel(id int) *PubRelPacket {
	return &PubRelPacket{
		header: &FixedHeader{
			PacketType: PUBREL,
			QoS:        0x01,
			Length:     2, // for Message ID field
		},
		ID: id,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *PubRelPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *PubRelPacket) WriteTo(w io.Writer) (n int, err error) {
	var d int
	n, err = pkt.header.WriteTo(w)
	if err != nil {
		return
	}
	d, err = writeInt(w, pkt.ID)
	n += d
	return
}

func readPubRel(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &PubRelPacket{header: fh}

	if len(buf) < 2 {
		// Missing message id field
		return pkt, errIncompleteMessage
	}
	pkt.ID = int(buf[0])<<8 + int(buf[1])
	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// PubRecPacket is a MQTT PUBREC control packet.
type PubRecPacket struct {
	header *FixedHeader
	ID     int
}

// String stringifies the packet in a human readable format.
func (pkt *PubRecPacket) String() string {
	return fmt.Sprintf("PUBREC mid:%d", pkt.ID)
}

// PubRec creates a new MQTT PUBREC packet.
func PubRec(id int) *PubRecPacket {
	return &PubRecPacket{
		header: &FixedHeader{
			PacketType: PUBREC,
			Length:     2, // for Message ID field
		},
		ID: id,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *PubRecPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *PubRecPacket) WriteTo(w io.Writer) (n int, err error) {
	var d int
	n, err = pkt.header.WriteTo(w)
	if err != nil {
		return
	}
	d, err = writeInt(w, pkt.ID)
	n += d
	return
}

func readPubRec(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &PubRecPacket{header: fh}

	if len(buf) < 2 {
		// Missing message id field
		return pkt, errIncompleteMessage
	}
	pkt.ID = int(buf[0])<<8 + int(buf[1])
	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// PubCompPacket is a MQTT PUBCOMP control packet.
type PubCompPacket struct {
	header *FixedHeader
	ID     int
}

// String stringifies the packet in a human readable format.
func (pkt *PubCompPacket) String() string {
	return fmt.Sprintf("PUBCOMP mid:%d", pkt.ID)
}

// PubComp creates a new MQTT PUBCOMP packet.
func PubComp(id int) *PubCompPacket {
	return &PubCompPacket{
		header: &FixedHeader{
			PacketType: PUBCOMP,
			Length:     2, // for Message ID field
		},
		ID: id,
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *PubCompPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *PubCompPacket) WriteTo(w io.Writer) (n int, err error) {
	var d int
	n, err = pkt.header.WriteTo(w)
	if err != nil {
		return
	}
	d, err = writeInt(w, pkt.ID)
	n += d
	return
}

func readPubComp(fh *FixedHeader, buf []byte) (Packet, error) {

	pkt := &PubCompPacket{header: fh}

	if len(buf) < 2 {
		// Missing message id field
		return pkt, errIncompleteMessage
	}
	pkt.ID = int(buf[0])<<8 + int(buf[1])

	return pkt, nil
}

////////////////////////////////////////////////////////////////////////////////

// PingReqPacket is a MQTT PINGREQ control packet.
type PingReqPacket struct {
	header *FixedHeader
}

// String stringifies the packet in a human readable format.
func (pkt *PingReqPacket) String() string {
	return "PINGREQ"
}

// PingReqPacket creates a new MQTT PINGREQ packet.
func PingReq() *PingReqPacket {
	return &PingReqPacket{
		header: &FixedHeader{
			PacketType: PINGREQ,
			Length:     0,
		},
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *PingReqPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *PingReqPacket) WriteTo(w io.Writer) (int, error) {
	return pkt.header.WriteTo(w)
}

func readPingReq(fh *FixedHeader, buf []byte) (Packet, error) {
	return &PingReqPacket{header: fh}, nil
}

////////////////////////////////////////////////////////////////////////////////

// PingRespPacket is a MQTT PINGRESP control packet.
type PingRespPacket struct {
	header *FixedHeader
}

// String stringifies the packet in a human readable format.
func (pkt *PingRespPacket) String() string {
	return "PINGRESP"
}

// PingResp creates a new MQTT PINGRESP packet.
func PingResp() *PingRespPacket {
	return &PingRespPacket{
		header: &FixedHeader{
			PacketType: PINGRESP,
			Length:     0,
		},
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *PingRespPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *PingRespPacket) WriteTo(w io.Writer) (int, error) {
	return pkt.header.WriteTo(w)
}

func readPingResp(fh *FixedHeader, buf []byte) (Packet, error) {
	return &PingRespPacket{header: fh}, nil
}

////////////////////////////////////////////////////////////////////////////////

// DisconnectPacket is a MQTT DISCONNECT control packet.
type DisconnectPacket struct {
	header *FixedHeader
}

// String stringifies the packet in a human readable format.
func (pkt *DisconnectPacket) String() string {
	return "DISCONNECT"
}

// Disconnect creates a new MQTT DISCONNECT packet.
func Disconnect() *DisconnectPacket {
	return &DisconnectPacket{
		header: &FixedHeader{
			PacketType: DISCONNECT,
			Length:     0,
		},
	}
}

// Header returns the FixedHeader of this packet.
func (pkt *DisconnectPacket) Header() *FixedHeader {
	return pkt.header
}

// WriteTo writes the packet to the io.Writer.
func (pkt *DisconnectPacket) WriteTo(w io.Writer) (int, error) {
	return pkt.header.WriteTo(w)
}

func readDisconnect(fh *FixedHeader, buf []byte) (Packet, error) {
	return &DisconnectPacket{header: fh}, nil
}

////////////////////////////////////////////////////////////////////////////////

func readString(buf []byte) (int, string) {
	length, b := readBytes(buf)
	return length, string(b)
}

func readBytes(buf []byte) (int, []byte) {

	if len(buf) < 2 {
		return 0, nil
	}
	length := (int(buf[0])<<8 + int(buf[1])) + 2
	if len(buf) < length {
		return 0, nil
	}
	return length, buf[2:length]
}

func writeString(w io.Writer, str string) (int, error) {
	return writeBytes(w, []byte(str))
}

func writeBytes(w io.Writer, b []byte) (int, error) {
	m, err := w.Write([]byte{byte(len(b) >> 8), byte(len(b) & 0xff)})
	if err != nil {
		return m, err
	}
	n, err := w.Write(b)
	return m + n, err
}

func writeInt(w io.Writer, i int) (int, error) {
	return w.Write([]byte{byte(i >> 8), byte(i & 0xff)})
}
