package mqtt

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/chiaf1/iot-nonna-ingest/internal/workers"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// This functoin creates a new MQTT client with keepAlive and autoRecconect options active
func NewClient(broker, clientId string, qOs uint8, retryInterval time.Duration, topics []string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientId)
	opts.SetOrderMatters(false)

	// Keep Alive
	opts.SetKeepAlive(30)
	opts.SetPingTimeout(10)

	// Auto Reconnect
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(retryInterval)

	// Error Logging
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Printf("[MQTT] Connection lost: %v\n", err)
	}

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("[MQTT] Connected to broker MQTT")
		// Subscrive to all topics on connect
		err := SubscribeAll(c, qOs, topics)
		if err != nil {
			log.Printf("[MQTT] Error subscribing to topics: %v", err)
		}
	}

	return mqtt.NewClient(opts)
}

// Subscribes the topic to the MQTT client
func Subscribe(c mqtt.Client, qOs uint8, topic string) error {
	token := c.Subscribe(topic, qOs, onSubscribe)
	token.Wait()

	return token.Error()
}

// Callback function to execute on Subscribe event
// pushes the message to the WorkerInput buffer
func onSubscribe(c mqtt.Client, msg mqtt.Message) {
	workers.WorkerInput <- workers.Message{
		Topic:   msg.Topic(),
		Payload: msg.Payload(),
		Time:    time.Now(),
	}
}

// Subscribes to all topics in the slice topics
func SubscribeAll(c mqtt.Client, qOs uint8, topics []string) error {
	if len(topics) == 0 {
		return fmt.Errorf("No topics to subscribe passed")
	}

	var errs []error
	for _, t := range topics {
		err := Subscribe(c, qOs, t)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("Subscribe finished with %d errors: %v", len(errs), errs)
	}

	return nil
}

// It will start a connection loop waiting for the first connection of the client, at each failed connection
// it will retry after a delay that is increased esponentilly until maxDelay is reached
// if a maxRetryes is set to zero the function will become a forever loop
func FirstConnect(c mqtt.Client, maxRetry int, retryDelay, maxDelay time.Duration) error {
	delay := retryDelay

	for attempt := 1; maxRetry == 0 || attempt <= maxRetry; attempt++ {

		token := c.Connect()
		token.Wait()

		if token.Error() == nil {
			log.Println("[MQTT] Connection established!")
			return nil
		}

		log.Printf("[MQTT] Connect attemp %d failed: %v", attempt, token.Error())

		// Exponential backoff + jitter
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))
		wait := delay + jitter

		if wait > maxDelay {
			wait = maxDelay
		}

		log.Printf("[MQTT] Waiting %v before connection retry...", wait)
		time.Sleep(wait)

		delay *= 2

		if delay > maxDelay {
			delay = maxDelay
		}
	}

	return fmt.Errorf("max connection retry reached (%d)", maxRetry)
}
