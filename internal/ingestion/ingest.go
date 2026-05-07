package ingestion

import (
	"time"

	"github.com/chiaf1/iot-nonna-ingest/internal/topic"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Ingest struct {
	DbPool             *pgxpool.Pool
	QueryTimeout_read  time.Duration
	QueryTimeout_write time.Duration
	TopicMap           *topic.TopicMap
	QoS_mqtt           uint8
}

func NewIngest(dbPool *pgxpool.Pool, tm *topic.TopicMap, queryTimeoutRead, queryTimeoutWrite time.Duration, qos_mqtt uint8) *Ingest {
	return &Ingest{
		DbPool:             dbPool,
		TopicMap:           tm,
		QueryTimeout_read:  queryTimeoutRead,
		QueryTimeout_write: queryTimeoutWrite,
		QoS_mqtt:           qos_mqtt,
	}
}
