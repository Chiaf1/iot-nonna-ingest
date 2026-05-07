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
	paho_mqtt "github.com/eclipse/paho.mqtt.golang"
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
		dbPool, err = postgres.OpenPool(conf.DB.DbURL, conf.DB.Query_timeout_read)
		if err == nil {
			break
		}
		log.Printf("[DB] Error while opening connection: %v \n", err)
		time.Sleep(5 * time.Second)
	}
	// TopicMap creation
	topicMap := topic.NewTopicMap()
	// Ingest creation
	ingest := ingestion.NewIngest(dbPool, topicMap, conf.DB.Query_timeout_read, conf.DB.Query_timeout_write, conf.MQTT.QoS)
	// Topic first load
	err = ingest.RefreshTopics()
	if err != nil {
		log.Fatalf("[ingest] Error refreshing topics for the first time: %v", err)
	}

	// 3. Context for life cycle
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. MQTT connection
	// Client creation
	client := mqtt.NewClient(
		conf.MQTT.Broker,
		conf.MQTT.ClientID,
		conf.MQTT.ConnectionInterval,
		func(c paho_mqtt.Client) {
			err := ingest.ResubscribeAll(c)
			if err != nil {
				log.Printf("[MQTT] Resubscribe failed: %v", err)
			}
		},
	)

	// 5. First connection attempt
	err = mqtt.FirstConnect(client, conf.MQTT.MaxRetry, conf.MQTT.ConnectionInterval, conf.MQTT.MaxDelay)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("MQTT connected")

	// 6. Spawn ingestion workers
	workers.StartWorkers(conf.Workers.Number, ingest.HandleMessage)

	// 7. Start topic refresher
	ingest.StartTopicRefresher(ctx, conf.DB.TopicRefreshingInterval, client)

	// 8. Wait for shut down signal
	<-ctx.Done()

	log.Println("Service shutdown started...")

	// Closing the channel, will stop all workers
	close(workers.WorkerInput)

	// Closing MQTT connection
	client.Unsubscribe(ingest.TopicMap.GetTopicList()...)
	client.Disconnect(250)
	time.Sleep(500 * time.Millisecond)

	// Closing db connection
	dbPool.Close()

	log.Println("Program ended")
}
