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
			readings_table_column,
			device_id,
			sensor_id
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
			topicName string
			tableName string
			jsonCols  []byte
			deviceID  string
			sensorID  string
		)

		// Scan the row to retrive the data
		if err := rows.Scan(&topicName, &tableName, &jsonCols, &deviceID, &sensorID); err != nil {
			return nil, err
		}

		// parse the jsonCols raw json data into a map[string]string, if the struct of the json changes this row needs to adapt
		cols := make(map[string]string)
		if err := json.Unmarshal(jsonCols, &cols); err != nil {
			return nil, err
		}

		// Populate the new topic map with the new data
		result[topicName] = topic.TopicConfig{
			DeviceID:     deviceID,
			SensorID:     sensorID,
			TableName:    tableName,
			TableColumns: cols,
		}
	}

	return result, rows.Err()
}
