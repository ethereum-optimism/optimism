package ws

import (
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
)

// HubCallbacks defines optional callback functions for Hub events.
// These can be used for testing, monitoring, and observability in production.
type HubCallbacks struct {
	// OnClientRegistered is called when a new client connects
	OnClientRegistered func(*Client)

	// OnClientUnregistered is called when a client disconnects
	OnClientUnregistered func(*Client)

	// OnMessageBroadcast is called after attempting to broadcast a message
	// Parameters: message, successCount, dropCount
	OnMessageBroadcast func([]byte, int, int)

	// OnShutdown is called when the hub is shutting down
	OnShutdown func()
}

// newHubWithCallbacks creates a new hub with optional callbacks.
// This allows for both testing and production monitoring/observability.
func newHubWithCallbacks(m metrics.Metricer, callbacks HubCallbacks) *Hub {
	hub := newHub(m)
	hub.callbacks = callbacks
	return hub
}

// SetCallbacks allows setting callbacks on an existing hub.
// Useful for adding monitoring to production instances.
func (h *Hub) SetCallbacks(callbacks HubCallbacks) {
	h.callbacks = callbacks
}
