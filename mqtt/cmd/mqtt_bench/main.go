package main

import (
	"log"
	"math/rand"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/j-forster/mqtt"
)

var topics []string

var numPublished uint64
var numRecieved uint64

func main() {

	numProducer := 10
	numConsumer := 10
	numTopics := 5 * numConsumer

	topics = make([]string, numTopics)
	for i := 0; i < numTopics; i++ {
		topics[i] = rndTopic()
	}

	runtime.GC()

	var id int32
	for n := 0; n < numConsumer; n++ {
		id = id*1664525 + 1013904223
		go Consumer(id)
	}

	id = 0
	for n := 0; n < numProducer; n++ {
		id = id*1664525 + 1013904223
		go Producer(id)
	}

	// start := time.Now()

	for true {
		time.Sleep(1 * time.Second)
		published := atomic.LoadUint64(&numPublished)
		recieved := atomic.LoadUint64(&numRecieved)
		log.Println("published", published, ", recieved", recieved)
	}
}

func Consumer(n int32) {
	id := "c" + strconv.FormatInt(int64(n), 10)

	client, err := mqtt.Dial("127.0.0.1:1883", id, true, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	numSubscriptions := 15
	for i := 0; i < numSubscriptions; i++ {
		j := (int(n)%len(topics) + len(topics)) % len(topics)
		client.Subscribe(topics[j], byte((n%3+3)%3))
		n = n*1664525 + 1013904223
	}

	for true {
		message, err := client.Message()
		if err != nil {
			log.Fatal(err)
		}
		if message == nil {
			break
		}

		atomic.AddUint64(&numRecieved, 1)
	}

	log.Fatal("disconnected")
}

var shortData = rndBuffer(30)
var mediumData = rndBuffer(300)
var longData = rndBuffer(30000)

func Producer(n int32) {
	id := "p" + strconv.FormatInt(int64(n), 10)

	client, err := mqtt.Dial("127.0.0.1:1883", id, true, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	sendMessage := func() {
		var message mqtt.Message

		for message.QoS == 0 {

			n = n*1664525 + 1013904223
			switch ((n % 3) + 3) % 3 {
			case 0:
				message.Data = shortData
				message.QoS = 0
			case 1:
				message.Data = mediumData
				message.QoS = 1
			case 2:
				message.Data = longData
				message.QoS = 2
			}
			j := (int(n)%len(topics) + len(topics)) % len(topics)
			message.Topic = topics[j]

			client.Publish(&message)

			atomic.AddUint64(&numPublished, 1)
		}
	}

	for i := 0; i < 8; i++ {
		sendMessage()
	}

	for true {
		packet, _, err := client.Packet()
		if err != nil {
			log.Fatal(err)
		}
		if packet == nil {
			log.Fatal("disconnected")
		}
		_, ok := packet.(*mqtt.PubAckPacket)
		if !ok {
			_, ok = packet.(*mqtt.PubCompPacket)
		}
		if ok {
			sendMessage()
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

var r uint32

var chars = []byte("abcdefghijklmnopqrstuvwxyz0123456789")

func rndTopic() string {
	length := nextRnd() % 6
	topic := rndName()
	for i := 0; i < length; i++ {
		topic += "/" + rndName()
	}
	return topic
}

func rndName() string {
	length := 4 + nextRnd()%20
	name := make([]byte, length)
	for i := range name {
		name[i] = rndChar()
	}
	return string(name)
}

func rndChar() byte {
	return chars[nextRnd()%len(chars)]
}

func nextRnd() int {
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	return int(r)
}

func rndBuffer(size int) []byte {
	buf := make([]byte, size)
	rand.Read(buf)
	return buf
}
