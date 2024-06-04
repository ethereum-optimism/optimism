package conductor

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SequencerConductor is an interface for the driver to communicate with the sequencer conductor.
// It is used to determine if the current node is the active sequencer, and to commit unsafe payloads to the conductor log.
type SequencerConductor interface {
	Leader(ctx context.Context) (bool, error)
	CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error
	Close()
}

// NoOpConductor is a no-op conductor that assumes this node is the leader sequencer.
type NoOpConductor struct{}

// Leader returns true if this node is the leader sequencer. NoOpConductor always returns true.
func (c *NoOpConductor) Leader(ctx context.Context) (bool, error) {
	return true, nil
}

// CommitUnsafePayload commits an unsafe payload to the conductor log.
func (c *NoOpConductor) CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	return nil
}

// Close closes the conductor client.
func (c *NoOpConductor) Close() {}
