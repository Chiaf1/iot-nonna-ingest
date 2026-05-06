package postgres

import (
	"context"
	"encoding/json"

	"github.com/chiaf1/iot-nonna-ingest/internal/topic"
	"github.com/jackc/pgx/v5/pgxpool"
)

func LoadTopicsFromDB(ctx context.Context, pool *pgxpool.Pool) (map[string]topic.TopicConfig, error) {
	// Lounch query
	rows, err := pool.Query(ctx, `
		SELECT
			topics,
			readings_table_name,
			device_id,
			sensor_id
			column_schema,
			value_mapping,
			payload_format
		FROM mqtt_topic_list_metadata
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Create a new map with the topic list
	result := make(map[string]topic.TopicConfig)

	// Scroll all rows got back from the query
	for rows.Next() {
		// Create the variables used to read from the rows result, the jsonCols is a []byte to get the raw json
		var (
			topicName     string
			tableName     string
			deviceID      string
			sensorID      string
			columnSchema  []byte
			valueMapping  []byte
			payloadFormat string
		)

		// Scan the row to retrive the data
		if err := rows.Scan(&topicName, &tableName, &deviceID, &sensorID, &columnSchema, &valueMapping, &payloadFormat); err != nil {
			return nil, err
		}

		// parse the columnSchema raw json data into a map[string]topic.TopicColumnDef, if the struct of the json changes this
		// row needs to adapt
		cols := make(map[string]topic.TopicColumnDef)
		if err := json.Unmarshal(columnSchema, &cols); err != nil {
			return nil, err
		}

		// Parse value mappings to map[string]any struct
		var mapping map[string]interface{}
		if len(valueMapping) > 0 {
			if err := json.Unmarshal(valueMapping, &mapping); err != nil {
				return nil, err
			}
		}

		// Populate the new topic map with the new data
		result[topicName] = topic.TopicConfig{
			DeviceID:      deviceID,
			SensorID:      sensorID,
			TableName:     tableName,
			ColumnSchema:  cols,
			ValueMapping:  mapping,
			PayloadFormat: payloadFormat,
		}
	}

	return result, rows.Err()
}
