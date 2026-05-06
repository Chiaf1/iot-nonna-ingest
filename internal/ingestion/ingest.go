package ingestion

import (
	"context"

	"github.com/chiaf1/iot-nonna-ingest/internal/topic"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Ingest struct {
	DbPool   *pgxpool.Pool
	TopicMap *topic.TopicMap
	ctx      *context.Context
}

func NewIngest(dbPool *pgxpool.Pool, tm *topic.TopicMap, c *context.Context) *Ingest {
	return &Ingest{
		DbPool:   dbPool,
		TopicMap: tm,
		ctx:      c,
	}
}
