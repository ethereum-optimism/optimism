// Package heartbeat provides a service for sending heartbeats to a server.
package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// SendInterval determines the delay between requests. This must be larger than the MinHeartbeatInterval in the server.
const SendInterval = 10 * time.Minute

type Payload struct {
	Version string `json:"version"`
	Meta    string `json:"meta"`
	Moniker string `json:"moniker"`
	PeerID  string `json:"peerID"`
	ChainID uint64 `json:"chainID"`
}

// Beat sends a heartbeat to the server at the given URL. It will send a heartbeat immediately, and then every SendInterval.
// Beat spawns a goroutine that will send heartbeats until the context is canceled.
func Beat(
	ctx context.Context,
	log log.Logger,
	url string,
	payload *Payload,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telemetry crashed: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	send := func() {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadJSON))
		req.Header.Set("User-Agent", fmt.Sprintf("op-node/%s", payload.Version))
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			log.Error("error creating heartbeat HTTP request", "err", err)
			return
		}
		res, err := client.Do(req)
		if err != nil {
			log.Warn("error sending heartbeat", "err", err)
			return
		}
		res.Body.Close()

		if res.StatusCode < 200 || res.StatusCode > 204 {
			log.Warn("heartbeat server returned non-200 status code", "status", res.StatusCode)
			return
		}

		log.Info("sent heartbeat")
	}

	send()
	tick := time.NewTicker(SendInterval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			send()
		case <-ctx.Done():
			return nil
		}
	}
}
