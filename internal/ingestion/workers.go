package ingestion

import (
	"log"
	"time"
)

type Message struct {
	Topic   string
	Payload []byte
	Time    time.Time
}

// Message buffer to input
var WorkerInput = make(chan Message, 1000)

// Starts n workers to handle the MQTT messages
func StartWorkers(n int, handler func(Message)) {
	for i := 0; i < n; i++ {
		go func(id int) {
			for msg := range WorkerInput {
				handler(msg)
			}
		}(i)
	}
}

// Process messages over the correct ingestion function
func ProcessMessage(msg Message) {
	switch msg.Topic {
	case "topic/test":
		processTest(msg)
	default:
		log.Printf("[Worker] Unknown topic [%s], payload: %s", msg.Topic, string(msg.Payload))
	}
}

// Process test topic
func processTest(msg Message) {
	log.Printf("[Ingestion] Message received [%s]: %s", msg.Topic, string(msg.Payload))
}
