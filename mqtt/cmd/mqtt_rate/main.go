package main

import (
	"log"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/j-forster/mqtt"
)

var numPublished uint64
var numRecieved uint64

func main() {

	runtime.GC()

	for i := 0; i < 5; i++ {
		go Consumer(i)
		go Producer(i)
	}

	for true {
		time.Sleep(1 * time.Second)
		published := atomic.LoadUint64(&numPublished)
		recieved := atomic.LoadUint64(&numRecieved)
		log.Printf("Published %d, Recieved %d", published, recieved)
	}
}

func Consumer(i int) {
	id := "c" + strconv.Itoa(i)
	topic := "topic" + strconv.Itoa(i)

	client, err := mqtt.Dial("127.0.0.1:1883", id, true, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	_, err = client.Subscribe(topic, 0x01)
	if err != nil {
		log.Fatalf("Consumer %d Err %v", i, err)
	}
	for true {
		message, err := client.Message()
		if err != nil {
			log.Fatalf("Consumer %d Err %v", i, err)
		}
		if message == nil {
			break
		}

		atomic.AddUint64(&numRecieved, 1)
	}

	log.Fatalf("Consumer %d Disconnected", i)
}

var data = []byte("Hello World :) 1234")

func Producer(i int) {
	id := "p" + strconv.Itoa(i)
	topic := "topic" + strconv.Itoa(i)

	client, err := mqtt.Dial("127.0.0.1:1883", id, true, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	sendMessage := func() {

		client.Publish(&mqtt.Message{
			QoS:   0x01,
			Topic: topic,
			Data:  data,
		})

		atomic.AddUint64(&numPublished, 1)
	}

	sendMessage()

	for true {
		packet, _, err := client.Packet()
		if err != nil {
			log.Fatalf("Producer %d Err %v", i, err)
		}
		if packet == nil {
			log.Fatalf("Producer %d Disconnected", i)
		}
		if _, ok := packet.(*mqtt.PubAckPacket); ok {
			sendMessage()
		}
	}
}
