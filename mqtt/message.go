package mqtt

type Message struct {
	Topic  string
	Data   []byte
	QoS    byte
	Retain bool
}
