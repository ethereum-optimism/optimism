package provider

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
)

// Heartbeat poll for expected transactions
func (p *Provider) Heartbeat(ctx context.Context) {
	log.Debug("heartbeat", "provider", p.name)
}

// Roundtrip send a new transaction to measure round trip latency
func (p *Provider) Roundtrip(ctx context.Context) {
	log.Debug("roundtrip", "provider", p.name)
}
