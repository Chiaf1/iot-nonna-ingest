package ingestion

import (
	"context"
	"fmt"

	internalMqtt "github.com/chiaf1/iot-nonna-ingest/internal/mqtt"
	"github.com/chiaf1/iot-nonna-ingest/internal/postgres"
	"github.com/chiaf1/iot-nonna-ingest/internal/topic"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Load topics from db and updates the TopicMap Struct
func (i *Ingest) RefreshTopics() error {
	ctx, cancel := context.WithTimeout(context.Background(), i.QueryTimeout_read)
	defer cancel()

	// Load topic map from db
	newMap, err := postgres.LoadTopicsFromDB(ctx, i.DbPool)
	if err != nil {
		return err
	}

	// Update the map
	i.TopicMap.ReplaceMap(newMap)
	return nil
}

// Load new topics from db then subscribe to new ones and unsub from unused ones then replace the topic map with the new one
func (i *Ingest) RefreshTopicsAndSubscriptions(mqttClient mqtt.Client) error {
	// 1. Create a context with timeout for the query
	ctx, cancel := context.WithTimeout(context.Background(), i.QueryTimeout_read)
	defer cancel()

	// 2. Load topic map from db
	newMap, err := postgres.LoadTopicsFromDB(ctx, i.DbPool)
	if err != nil {
		return err
	}

	// 3. Snapshot of oldTopics and new ones
	oldTopics := sliceToSet(i.TopicMap.GetTopicList())
	newTopics := sliceToSet(getKeysFromMap(newMap))

	// 4. Create slice of topic to sub and unsub
	toSub := setDiff(newTopics, oldTopics)
	toUnsub := setDiff(oldTopics, newTopics)

	// 5. Sub to new topics and unsub unused topics
	err = subscribeAllWithQos(mqttClient, toSub, newMap, i.QoS_mqtt)
	if err != nil {
		return err
	}
	if len(toUnsub) > 0 {
		mqttClient.Unsubscribe(toUnsub...)
	}

	// 6. Replace TopicMap atomically
	i.TopicMap.ReplaceMap(newMap)
	return nil
}

// Convertes a slice into a map of struct, basically an empty map with only keys
func sliceToSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, v := range items {
		set[v] = struct{}{}
	}
	return set
}

// Compares two map[string]struct{} and returns the keys that exist in map a but not in map b
func setDiff(a, b map[string]struct{}) []string {
	var out []string
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

// Returns a slice of strings with all the keys of a map[string]any
func getKeysFromMap[V any](m map[string]V) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Resubscribe to all topics
func (i *Ingest) ResubscribeAll(c mqtt.Client) error {
	topics := i.TopicMap.GetTopicList()
	if len(topics) == 0 {
		return nil
	}
	return subscribeAllWithQos(c, topics, i.TopicMap.GetMap(), i.QoS_mqtt)
}

// Subscribe to all topics in the list using the qos in the metadata if exist or the one in the config if not
func subscribeAllWithQos(c mqtt.Client, topics []string, src map[string]topic.TopicConfig, qosFallback uint8) error {
	if len(topics) == 0 {
		return nil
	}
	var errs []error
	for _, t := range topics {
		tConf, ok := src[t]
		var QoS uint8
		if !ok || tConf.Qos_mqtt == nil {
			QoS = qosFallback
		} else {
			QoS = *tConf.Qos_mqtt
		}
		if QoS > 2 {
			QoS = qosFallback
		}
		err := internalMqtt.Subscribe(c, QoS, t)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("Subscribe finished with %d errors: %v", len(errs), errs)
	}

	return nil
}
