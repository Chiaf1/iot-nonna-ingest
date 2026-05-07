package postgres

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open pool for database connection
func OpenPool(dsn string, queryTimeout time.Duration) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	// loading configs drom connection url
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// Adjusting some configs
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Connection test
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func NewPgPool(dsn string, queryTimeout time.Duration, maxRetry int, retryDelay, maxDelay time.Duration) (*pgxpool.Pool, error) {
	delay := retryDelay

	for attempt := 1; maxRetry == 0 || attempt <= maxRetry; attempt++ {
		dbPool, err := OpenPool(dsn, queryTimeout)
		if err == nil {
			return dbPool, nil
		}
		log.Printf("[DB] Error while opening connection: %v \n", err)

		// Exponential backoff + jitter
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))
		wait := delay + jitter

		if wait > maxDelay {
			wait = maxDelay
		}

		log.Printf("[DB] Waiting %v before connection retry...", wait)
		time.Sleep(wait)

		delay *= 2

		if delay > maxDelay {
			delay = maxDelay
		}
	}

	return nil, fmt.Errorf("max connection retry reached (%d)", maxRetry)
}
