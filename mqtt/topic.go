package mqtt

import (
	"strconv"
	"strings"
)

type Subscription struct {
	recv Reciever

	topic *topic

	qos byte

	next, prev *Subscription
}

func newSubscription(recv Reciever, qos byte) *Subscription {
	return &Subscription{
		recv: recv,
		qos:  qos,
	}
}

func (s *Subscription) QoS() byte {
	return s.qos
}

func (s *Subscription) Topic() string {
	return s.topic.fullName()
}

func (s *Subscription) publish(msg *Message) int {

	if s == nil {
		return 0
	}

	// ignores errors
	s.recv.Publish(&Message{
		Topic: msg.Topic,
		Data:  msg.Data,
		QoS:   min(msg.QoS, s.qos),
	})

	// walk the chain
	return s.next.publish(msg) + 1
}

func (s *Subscription) chainLength() int {
	if s == nil {
		return 0
	}
	return s.next.chainLength() + 1
}

func (s *Subscription) unsubscribe() {

	t := s.topic

	if t == nil {
		return
	}

	if s.prev == nil {
		if t.subs == s {
			t.subs = s.next
		} else {
			t.mlwcSubs = s.next
		}

		if s.next != nil {
			s.next.prev = nil
		}

		// the topic we unsubscribed can be removed if
		if t.subs == nil && // no subscribers
			t.retainMsg == nil && // no retain message
			t.mlwcSubs == nil && // no /# subscribers
			t.wcTopic == nil && // no /+ topic
			len(t.children) == 0 { // no sub-topics
			t.remove()
		}
	} else {
		s.prev.next = s.next
		if s.next != nil {
			s.next.prev = s.prev
		}
	}

	s.topic = nil
}

///////////////////////////////////////////////////////////////////////////////

type topic struct {
	// topic name like "b" in 'a/b' for b
	name string
	// any sub topic like /b in 'a/b' for a
	children map[string]*topic
	// wildcard (+) topic
	wcTopic *topic
	// parent topic like a in 'a/b' for b
	parent *topic
	// all subscriptions to this topic (double linked list)
	subs *Subscription
	// all subscriptions to /# (multi level wildcard)
	mlwcSubs *Subscription
	// retain message
	retainMsg *Message
}

func newTopic(parent *topic, name string) *topic {
	return &topic{
		children: make(map[string]*topic),
		parent:   parent,
		name:     name,
	}
}

func (t *topic) find(s []string) *Subscription {

	if len(s) == 0 {
		return t.subs
	}
	t, ok := t.children[s[0]]
	if ok {
		return t.find(s[1:])
	}
	return nil
}

func (t *topic) publish(s []string, msg *Message) int {

	var hit int
	retain := msg.Retain

	if len(s) == 0 {
		// len() = 0 means we are at the end of the topics-tree

		// attach retain message to the topic
		if retain {
			if len(msg.Data) != 0 {
				t.retainMsg = msg
			} else {
				t.retainMsg = nil
			}
		}

		// and inform all subscribers here

		msg.Retain = false
		hit = t.subs.publish(msg)
	} else {

		// search for the child note
		c, ok := t.children[s[0]]
		if ok {
			hit = c.publish(s[1:], msg)
		} else {

			if msg.Retain {
				// retain messages are attached to a topic
				// se we need to create the topic as it does not exist
				if len(msg.Data) != 0 {
					t.subscribe(s, nil)
					t.children[s[0]].publish(s[1:], msg)
				}
			}
		}

		// notify all ../+ subscribers
		if t.wcTopic != nil {
			hit += t.wcTopic.publish(s[1:], msg)
		}

		msg.Retain = false
	}

	// the /# subscribers always match

	hit += t.mlwcSubs.publish(msg)

	msg.Retain = retain
	return hit
}

func (t *topic) fullName() string {
	if t.parent != nil {
		return t.parent.fullName() + "/" + t.name
	}
	return t.name

}

// Debugging Helper Function
func (t *topic) String() string {

	var builder strings.Builder
	if n := t.subs.chainLength(); n != 0 {
		builder.WriteString("\n/ (" + strconv.Itoa(n) + " listeners)\n")
	}
	if n := t.mlwcSubs.chainLength(); n != 0 {
		builder.WriteString("\n/# (" + strconv.Itoa(n) + " listeners)\n")
	}

	for sub, t := range t.children {
		builder.WriteString("\n" + sub + " (" + strconv.Itoa(t.subs.chainLength()) + " listeners)")
		t.printIndent(&builder, "  ")
	}
	return builder.String()
}

// Debugging Helper Function
func (t *topic) printIndent(builder *strings.Builder, indent string) {

	if n := t.mlwcSubs.chainLength(); n != 0 {
		builder.WriteString("\n" + indent + "/# (" + strconv.Itoa(n) + " listeners)")
	}

	if t.wcTopic != nil {

		builder.WriteString("\n" + indent + "/+ (" + strconv.Itoa(t.wcTopic.subs.chainLength()) + " listeners)")
		t.wcTopic.printIndent(builder, indent+"  ")
	}

	for sub, t := range t.children {
		builder.WriteString("\n" + indent + "/" + sub + " (" + strconv.Itoa(t.subs.chainLength()) + " listeners)")
		t.printIndent(builder, indent+"  ")
	}
}

func (t *topic) enqueue(queue **Subscription, s *Subscription) {

	s.prev = nil
	s.next = *queue

	if s.next != nil {
		s.next.prev = s
	}
	*queue = s

	s.topic = t
}

func (t *topic) releaseMlRetain(sub *Subscription) {

	if t.retainMsg != nil {
		// ignores errors
		sub.recv.Publish(&Message{
			Topic:  t.retainMsg.Topic,
			Data:   t.retainMsg.Data,
			Retain: true,
			QoS:    min(t.retainMsg.QoS, sub.qos),
		})
	}

	for _, child := range t.children {
		child.releaseMlRetain(sub)
	}
}

func (t *topic) releasRetain(name []string, sub *Subscription) {

	if len(name) == 0 {
		if t.retainMsg != nil {
			// ignores errors
			sub.recv.Publish(&Message{
				Topic:  t.retainMsg.Topic,
				Data:   t.retainMsg.Data,
				Retain: true,
				QoS:    min(t.retainMsg.QoS, sub.qos),
			})
		}
		return
	}
	if name[0] == "+" {
		for _, child := range t.children {
			child.releasRetain(name[1:], sub)
		}
		return
	}
	if name[0] == "#" {
		for _, child := range t.children {
			child.releaseMlRetain(sub)
		}
		return
	}
	if child, ok := t.children[name[0]]; ok {
		child.releasRetain(name[1:], sub)
	}
}

func (t *topic) releaseWcRetain(name []string, sub *Subscription) {

	if len(name) == 0 {
		if t.retainMsg != nil {
			// ignores errors
			sub.recv.Publish(&Message{
				Topic:  t.retainMsg.Topic,
				Data:   t.retainMsg.Data,
				Retain: true,
				QoS:    min(t.retainMsg.QoS, sub.qos),
			})
		}
	} else {
		for _, child := range t.children {
			child.releasRetain(name, sub)
		}
	}
}

func (t *topic) subscribe(name []string, sub *Subscription) {

	if len(name) == 0 {

		if sub != nil {
			t.enqueue(&t.subs, sub)

			if t.retainMsg != nil {
				// ignores errors
				sub.recv.Publish(&Message{
					Topic:  t.retainMsg.Topic,
					Data:   t.retainMsg.Data,
					Retain: true,
					QoS:    min(t.retainMsg.QoS, sub.qos),
				})
			}
		}

	} else {

		if name[0] == "#" { // Multi Level Wildcard
			if sub != nil {
				t.enqueue(&t.mlwcSubs, sub)
			}
			// collect all retain messages
			for _, child := range t.children {
				child.releaseMlRetain(sub)
			}
			return
		}

		if name[0] == "+" { // Single Level Wildcard

			for _, child := range t.children {
				child.releasRetain(name[1:], sub)
			}

			if t.wcTopic == nil {
				t.wcTopic = newTopic(t, "+")
			}
			t.wcTopic.subscribe(name[1:], sub)

		} else {
			child, ok := t.children[name[0]]
			if !ok {
				child = newTopic(t, name[0])
				t.children[name[0]] = child
			}
			child.subscribe(name[1:], sub)
		}
	}
}

func (t *topic) remove() {

	parent := t.parent

	if parent != nil {

		// the wildcard topic is attached different to the parent topic
		if t.name == "+" {

			// also the parent topic if
			if len(parent.children) == 0 && // no sub-topics
				parent.retainMsg == nil && // no retain message
				parent.subs == nil && // no subscriptions
				parent.mlwcSubs == nil && // no /# subscriptions
				parent.parent != nil { // but not the root topic :)

				parent.remove()
				return
			}

			// remove this wildcard topic
			parent.wcTopic = nil
			return
		}

		// also the parent topic if
		if parent.wcTopic == nil && // no /+ subscribers
			parent.retainMsg == nil && // no retain message
			parent.mlwcSubs == nil && // no /# subscribers
			len(parent.children) == 1 && // no sub-topics (just this one)
			parent.subs == nil && // no subscriptions
			parent.parent != nil { // but not the root topic

			parent.remove()
			return
		}

		// remove this topic from the parents sub-topics
		delete(parent.children, t.name)
	}
}

func checkTopic(topic string) (valid bool, wildcards bool) {
	for true {
		i := strings.IndexRune(topic, '/')
		if i == -1 {
			valid = topic != ""
			wildcards = wildcards || topic == "*" || topic == "#"
			return
		}
		sub := topic[:i]
		if sub == "#" || sub == "" {
			valid = false
			return
		}
		wildcards = wildcards || sub == "*"
		topic = topic[i+1:]
	}
	return // unreachable
}

func min(a, b byte) byte {
	if a > b {
		return b
	}
	return a
}
