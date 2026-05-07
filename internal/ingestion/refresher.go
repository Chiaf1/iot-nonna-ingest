package ingestion

import (
	"context"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Start go function for periodic topic refreshing and sub/unsub dei topic
func (i *Ingest) StartTopicRefresher(ctx context.Context, interval time.Duration, client mqtt.Client) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if !client.IsConnected() {
					continue
				}
				if err := i.RefreshTopicsAndSubscriptions(client); err != nil {
					log.Printf("[Topics] Refresh failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
