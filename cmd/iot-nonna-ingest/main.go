package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chiaf1/iot-nonna-ingest/internal/config"
	"github.com/chiaf1/iot-nonna-ingest/internal/mqtt"
	"github.com/chiaf1/iot-nonna-ingest/internal/workers"
)

const CONFIG_PATH = "./config.yaml"

func main() {
	// Load configs from file
	var conf config.Config
	err := conf.Load(CONFIG_PATH)
	if err != nil {
		log.Fatal(err)
	}
	err = conf.Validate()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Config Loaded")

	// MQTT connection
	// Client creation
	client := mqtt.NewClient(conf.Broker, conf.ClientID, conf.QoS, conf.ConnectionInterval, conf.Topics)
	// First connection attempt
	err = mqtt.FirstConnect(client, conf.MaxRetry, conf.ConnectionInterval, conf.MaxDelay)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Client connected")

	// Start ingestion workers
	workers.StartWorkers(5, workers.ProcessMessage)

	// Grace full shut down
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	log.Println("Service shutdown started...")

	// Closing the channel, will stop all workers
	close(workers.WorkerInput)

	// Closing MQTT connection
	client.Unsubscribe(conf.Topics...)
	client.Disconnect(250)
	time.Sleep(500 * time.Millisecond)

	log.Println("Program ended")
}
