package mqtt

import (
	"testing"
)

func TestTopic(t *testing.T) {

	root := newTopic(nil, "")


	otherMessage := &Message{}

	var message *Message

	recv := RecieverFunc(func(msg *Message) error {
		message = msg
		return nil
	})

	subsAB := newSubscription(recv, 0x00)

	root.subscribe([]string{"a", "b"}, subsAB)
	root.publish([]string{"a", "b"}, simpleMessage)

	root.publish([]string{"a"}, otherMessage)
	root.publish([]string{"a", "b", "c"}, otherMessage)

	if message == nil {
		t.Fatalf("reciever not called")
	}

	if message.Equals(otherMessage) {
		t.Fatalf("recieved wrong message")
	}

	if !message.Equals(simpleMessage) {
		t.Fatalf("recieved unknown message")
	}

	subsAB.unsubscribe()

	if len(root.children) != 0 {
		t.Fatalf("topics tree no properly destructed")
	}

	root.publish([]string{"a", "b"}, otherMessage)
	if !message.Equals(simpleMessage) {
		t.Fatalf("reciever called after unsubscribe")
	}
}

func TestTopicRetain(t *testing.T) {
	root := newTopic(nil, "")

	simpleMessage := &Message{
		Retain: true,
		Data:   []byte("1234"),
	}

	emptyMessage := &Message{
		Retain: true,
	}

	var message *Message

	recv := RecieverFunc(func(msg *Message) error {
		message = msg
		return nil
	})

	subsAB := newSubscription(recv, 0x00)

	root.subscribe([]string{"a", "b"}, subsAB)
	root.publish([]string{"a", "b"}, simpleMessage)

	if message == nil {
		t.Fatalf("recieved no message")
	}

	subsAB.unsubscribe()
	message = nil

	root.subscribe([]string{"a", "b"}, subsAB)

	if !message.Equals(simpleMessage) {
		t.Fatalf("did not recieve retained message")
	}

	message = nil

	subsAB.unsubscribe()
	root.publish([]string{"a", "b"}, emptyMessage)
	root.subscribe([]string{"a", "b"}, subsAB)

	if message != nil {
		t.Fatalf("recieved message after removing retained message")
	}

	subsAB.unsubscribe()

	if len(root.children) != 0 {
		t.Fatalf("topics tree no properly destructed")
	}
}
