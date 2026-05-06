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

	// 2. Parse payload
	payload, err := parsePayload(msg.Payload, cfg.PayloadFormat)
	if err != nil {
		log.Printf("[Ingest] Error parsing payload on topic %s: %v", msg.Topic, err)
		return
	}

	// 3. Dynamic columns contrustion
	columns := make([]string, 0, len(cfg.ColumnSchema))
	placeholders := make([]string, 0, len(cfg.ColumnSchema))
	values := make([]interface{}, 0, len(cfg.ColumnSchema))

	idx := 1
	for jsonKey, dbCol := range cfg.ColumnSchema {
		val, exists := payload[jsonKey]
		if !exists {
			// If the key is missing from the json payload it skips it
			continue
		}
		// Normalize value from mapping
		mappedVal, ok := normalizeValue(val, cfg.ValueMapping)
		if !ok {
			// normalization not possible
			log.Printf("[Ingest] Error normalizing value \"%v\" on topic %s", val, msg.Topic)
			continue
		}
		// Cast value to correct data type
		castVal, err := castValue(mappedVal, dbCol.Type)
		if err != nil {
			log.Printf("[Ingest] Error during casting of topic %s: %v", msg.Topic, err)
			continue
		}

		columns = append(columns, dbCol.Column)
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		values = append(values, castVal)
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

	_, err = i.DbPool.Exec(ctx, query, values...)
	if err != nil {
		log.Printf("[Ingest] DB insert failed (topic=%s): %v", msg.Topic, err)
	}
}

// Parse payload based on format, supported formats:
// - "json"
// - "raw"
func parsePayload(msg []byte, format string) (map[string]interface{}, error) {
	switch format {
	case "json":
		var data map[string]interface{}
		err := json.Unmarshal(msg, &data)
		return data, err
	case "raw":
		return map[string]interface{}{
			"$payload": string(msg),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported payload_format: %s", format)
	}
}

// Normalize values based on mappings if any and only on strings, returns the normalized value and a bool to know if it was done or not
func normalizeValue(val interface{}, mapping map[string]interface{}) (interface{}, bool) {
	// no mapping needed
	if mapping == nil {
		return val, true
	}

	// If value not string, assume it's already normalized
	if _, ok := val.(string); !ok {
		return val, true
	}

	// Mapping applies only on strings
	s, ok := val.(string)
	if !ok {
		return nil, false
	}

	mapped, ok := mapping[strings.ToLower(s)]
	return mapped, ok
}

// Cast values to correct type for insertion to database
func castValue(val interface{}, typ string) (interface{}, error) {
	switch typ {
	case "float":
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float, got %T", val)
		}
		return f, nil
	case "int":
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected int, got %T", val)
		}
		return int64(f), nil
	case "bool":
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", val)
		}
		return b, nil
	case "string":
		return fmt.Sprintf("%v", val), nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typ)
	}
}
