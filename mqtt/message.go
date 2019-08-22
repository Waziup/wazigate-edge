package mqtt

import "bytes"

// Message is a published message for a topic a the given QoS.
type Message struct {
	// Topic of this message.
	Topic string
	// Data
	Data []byte
	// QoS = Quality of Service
	QoS byte
	// Retain is true if the message is a Retain Message.
	Retain bool
}

// Equals checks if both message are the same.
func (msg *Message) Equals(other *Message) bool {
	return msg.Topic == other.Topic &&
		msg.QoS == other.QoS &&
		msg.Retain == other.Retain &&
		bytes.Compare(msg.Data, other.Data) == 0
}
