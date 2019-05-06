package mqtt

import (
	"strconv"
	"strings"
)

///////////////////////////////////////////////////////////////////////////////

type Subscription struct {
	Recv Reciever
	// topic string
	Topic *Topic

	QoS byte

	Next, Prev *Subscription
}

func NewSubscription(recv Reciever, qos byte) *Subscription {
	return &Subscription{
		Recv: recv,
		QoS:  qos,
	}
}

func (s *Subscription) Publish(client *Client, msg *Message) {

	if s == nil {
		return
	}

	s.Recv.Publish(client, msg)

	s.Next.Publish(client, msg)
}

func (s *Subscription) ChainLength() int {
	if s == nil {
		return 0
	}
	return s.Next.ChainLength() + 1
}

func (sub *Subscription) Unsubscribe() {

	topic := sub.Topic

	if topic == nil {
		return
	}

	if sub.Prev == nil {
		if topic.Subs == sub {
			topic.Subs = sub.Next
		} else {
			topic.MLWCSubs = sub.Next
		}

		if sub.Next != nil {
			sub.Next.Prev = nil
		}

		// the topic we unsubscribed can be removed if
		if topic.Subs == nil && // no subscribers
			topic.RetainMsg == nil && // no retrain message
			topic.MLWCSubs == nil && // no /# subscribers
			topic.WCTopic == nil && // no /+ topic
			len(topic.Children) == 0 { // no sub-topics
			topic.Remove()
		}
	} else {
		sub.Prev.Next = sub.Next
		if sub.Next != nil {
			sub.Next.Prev = sub.Prev
		}
	}

	sub.Topic = nil
}

///////////////////////////////////////////////////////////////////////////////

type Topic struct {
	// topic name like "b" in 'a/b' for b
	Name string
	// any sub topic like /b in 'a/b' for a
	Children map[string]*Topic
	// wildcard (+) topic
	WCTopic *Topic
	// parent topic like a in 'a/b' for b
	Parent *Topic
	// all subscriptions to this topic (double linked list)
	Subs *Subscription
	// all subscriptions to /# (multi level wildcard)
	MLWCSubs *Subscription
	// retain message
	RetainMsg *Message
}

func NewTopic(parent *Topic, name string) *Topic {
	return &Topic{
		Children: make(map[string]*Topic),
		Parent:   parent,
		Name:     name,
	}
}

func (topic *Topic) Find(s []string) *Subscription {

	if len(s) == 0 {

		return topic.Subs
	} else {

		t, ok := topic.Children[s[0]]
		if ok {
			return t.Find(s[1:])
		}
	}
	return nil
}

func (topic *Topic) Publish(s []string, client *Client, msg *Message) {

	if len(s) == 0 {

		// len() = 0 means we are at the end of the topics-tree
		// and inform all subscribers here
		topic.Subs.Publish(client, msg)

		// attach retain message to the topic
		if msg.Retain {
			topic.RetainMsg = msg
		}
	} else {

		// search for the child note
		t, ok := topic.Children[s[0]]
		if ok {
			t.Publish(s[1:], client, msg)
		} else {

			if msg.Retain {
				// retain messages are attached to a topic
				// se we need to create the topic as it does not exist
				t.Subscribe(s[1:], nil)
			}
		}

		// notify all ../+ subscribers
		if topic.WCTopic != nil {
			topic.WCTopic.Publish(s[1:], client, msg)
		}
	}

	// the /# subscribers always match
	topic.MLWCSubs.Publish(client, msg)
}

func (topic *Topic) FullName() string {
	if topic.Parent != nil {
		return topic.Parent.FullName() + "/" + topic.Name
	} else {
		return topic.Name
	}
}

func (topic *Topic) String() string {

	var builder strings.Builder
	if n := topic.Subs.ChainLength(); n != 0 {
		builder.WriteString("\n/ (" + strconv.Itoa(n) + " listeners)\n")
	}
	if n := topic.MLWCSubs.ChainLength(); n != 0 {
		builder.WriteString("\n/# (" + strconv.Itoa(n) + " listeners)\n")
	}

	for sub, topic := range topic.Children {
		builder.WriteString("\n" + sub + " (" + strconv.Itoa(topic.Subs.ChainLength()) + " listeners)")
		topic.PrintIndent(&builder, "  ")
	}
	return builder.String()
}

func (topic *Topic) PrintIndent(builder *strings.Builder, indent string) {

	if n := topic.MLWCSubs.ChainLength(); n != 0 {
		builder.WriteString("\n" + indent + "/# (" + strconv.Itoa(n) + " listeners)")
	}

	if topic.WCTopic != nil {

		builder.WriteString("\n" + indent + "/+ (" + strconv.Itoa(topic.WCTopic.Subs.ChainLength()) + " listeners)")
		topic.WCTopic.PrintIndent(builder, indent+"  ")
	}

	for sub, t := range topic.Children {
		builder.WriteString("\n" + indent + "/" + sub + " (" + strconv.Itoa(t.Subs.ChainLength()) + " listeners)")
		t.PrintIndent(builder, indent+"  ")
	}
}

func (topic *Topic) Enqueue(queue **Subscription, s *Subscription) {

	s.Prev = nil
	s.Next = *queue

	if s.Next != nil {
		s.Next.Prev = s
	}
	*queue = s

	s.Topic = topic
}

func (topic *Topic) Subscribe(t []string, sub *Subscription) {

	if len(t) == 0 {

		topic.Enqueue(&topic.Subs, sub)
		if topic.RetainMsg != nil {
			sub.Recv.Publish(nil, topic.RetainMsg)
		}

	} else {

		if t[0] == "#" {
			topic.Enqueue(&topic.MLWCSubs, sub)
			return
		}

		var child *Topic
		var ok bool

		if t[0] == "+" {

			if topic.WCTopic == nil {
				topic.WCTopic = NewTopic(topic, "+")
			}
			child = topic.WCTopic
		} else {
			child, ok = topic.Children[t[0]]
			if !ok {
				child = NewTopic(topic, t[0])
				topic.Children[t[0]] = child
			}
		}

		child.Subscribe(t[1:], sub)
	}
}

func (topic *Topic) Remove() {

	parent := topic.Parent

	if parent != nil {

		// the wildcard topic is attached different to the parent topic
		if topic.Name == "+" {

			// also the parent topic if
			if len(parent.Children) == 0 && // no sub-topics
				parent.RetainMsg == nil && // no retain message
				parent.Subs == nil && // no subscriptions
				parent.MLWCSubs == nil && // no /# subscriptions
				parent.Parent != nil { // but not the root topic :)

				parent.Remove()
				return
			}

			// remove this wildcard topic
			parent.WCTopic = nil
			return
		}

		// also the parent topic if
		if parent.WCTopic == nil && // no /+ subscribers
			parent.RetainMsg == nil && // no retain message
			parent.MLWCSubs == nil && // no /# subscribers
			len(parent.Children) == 1 && // no sub-topics (just this one)
			parent.Subs == nil && // no subscriptions
			parent.Parent != nil { // but not the root topic

			parent.Remove()
			return
		}

		// remove this topic from the parents sub-topics
		delete(parent.Children, topic.Name)
	}
}
