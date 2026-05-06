package ingestion

import (
	"context"

	"github.com/chiaf1/iot-nonna-ingest/internal/postgres"
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
