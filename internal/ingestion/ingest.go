package ingestion

import (
	"time"

	"github.com/chiaf1/iot-nonna-ingest/internal/topic"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Ingest struct {
	DbPool       *pgxpool.Pool
	QueryTimeout time.Duration
	TopicMap     *topic.TopicMap
}

func NewIngest(dbPool *pgxpool.Pool, tm *topic.TopicMap, queryTimeout time.Duration) *Ingest {
	return &Ingest{
		DbPool:       dbPool,
		TopicMap:     tm,
		QueryTimeout: queryTimeout,
	}
}
