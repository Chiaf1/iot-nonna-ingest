package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/chiaf1/iot-nonna-ingest/internal/workers"
)

func (i *Ingest) HandleMessage(msg workers.Message) {
	// 1. Lookup topic config in topic map
	cfg, ok := i.TopicMap.GetTopicConf(msg.Topic)
	if !ok {
		log.Printf("[Ingest] Topic not found in TopicMap: %s", msg.Topic)
		return
	}

	// 2. Decode JSON payload dynamically
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("[Ingest] Invalid JSON payload on topic %s: %v", msg.Topic, err)
		return
	}

	// 3. Dynamic columns contrustion
	columns := make([]string, 0, len(cfg.TableColumns))
	placeholders := make([]string, 0, len(cfg.TableColumns))
	values := make([]interface{}, 0, len(cfg.TableColumns))

	idx := 1
	for jsonKey, dbCol := range cfg.TableColumns {
		val, exists := payload[jsonKey]
		if !exists {
			// If the key is missing from the json payload it skips it
			continue
		}

		columns = append(columns, dbCol)
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		values = append(values, val)
		idx++
	}

	// 4. Append standard columns
	columns = append(columns, "device_id", "sensor_id")
	placeholders = append(placeholders, fmt.Sprintf("$%d", idx), fmt.Sprintf("$%d", idx+1))
	values = append(values, cfg.DeviceID, cfg.SensorID)

	// 5. Query assembly
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		cfg.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// 6. Query execution
	ctx, cancel := context.WithTimeout(context.Background(), i.QueryTimeout_read)
	defer cancel()

	_, err := i.DbPool.Exec(ctx, query, values...)
	if err != nil {
		log.Printf("[Ingest] DB insert failed (topic=%s): %v", msg.Topic, err)
	}
}
