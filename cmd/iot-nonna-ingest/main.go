package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chiaf1/iot-nonna-ingest/internal/config"
	"github.com/chiaf1/iot-nonna-ingest/internal/ingestion"
	"github.com/chiaf1/iot-nonna-ingest/internal/mqtt"
	"github.com/chiaf1/iot-nonna-ingest/internal/postgres"
	"github.com/chiaf1/iot-nonna-ingest/internal/topic"
	"github.com/chiaf1/iot-nonna-ingest/internal/workers"
	"github.com/jackc/pgx/v5/pgxpool"
)

const CONFIG_PATH = "./config.yaml"

func main() {
	confPath := os.Getenv("CONFIG_PATH")
	if confPath == "" {
		confPath = CONFIG_PATH
	}
	// 1. Load configs from file
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

	// 2. Connet to db and load topic map
	// DB connection pool creation - Need to be added jitter and max retrys
	var dbPool *pgxpool.Pool
	for {
		dbPool, err = postgres.OpenPool(conf.DB.DbURL)
		if err == nil {
			break
		}
		log.Printf("[DB] Error while opening connection: %v \n", err)
		time.Sleep(5 * time.Second)
	}
	// TopicMap creation
	topicMap := topic.NewTopicMap()
	// Ingest creation
	ingest := ingestion.NewIngest(dbPool, topicMap)
	// Topic first load
	err = ingest.RefreshTopics()
	if err != nil {
		log.Fatalf("[ingest] Error refreshing topics for the first time: %v", err)
	}

	// 3. MQTT connection
	// Client creation
	client := mqtt.NewClient(conf.MQTT.Broker, conf.MQTT.ClientID, conf.MQTT.QoS, conf.MQTT.ConnectionInterval, ingest.TopicMap.GetTopicList())
	// First connection attempt
	err = mqtt.FirstConnect(client, conf.MQTT.MaxRetry, conf.MQTT.ConnectionInterval, conf.MQTT.MaxDelay)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Client connected")

	// 4. Start ingestion workers
	workers.StartWorkers(5, workers.ProcessMessage)

	// 5. Grace full shut down
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	log.Println("Service shutdown started...")

	// Closing the channel, will stop all workers
	close(workers.WorkerInput)

	// Closing MQTT connection
	client.Unsubscribe(ingest.TopicMap.GetTopicList()...)
	client.Disconnect(250)
	time.Sleep(500 * time.Millisecond)

	log.Println("Program ended")
}
