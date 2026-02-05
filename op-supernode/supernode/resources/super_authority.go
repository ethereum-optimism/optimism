package resources

import "github.com/ethereum/go-ethereum/common"

// SuperAuthority is an interface for supernode-level authority operations.
// It is passed to op-node instances during initialization to provide
// supernode-specific functionality and coordination.
type SuperAuthority interface {
	// IsDenied checks if a payload hash is denied at the given block number.
	// Returns true if the payload should not be applied.
	// The error indicates if the check could not be performed.
	IsDenied(blockNumber uint64, payloadHash common.Hash) (bool, error)
}

// NoOpSuperAuthority is a no-op implementation that never denies any payload.
// Used when running op-node outside of a supernode context.
type NoOpSuperAuthority struct{}

func (n *NoOpSuperAuthority) IsDenied(blockNumber uint64, payloadHash common.Hash) (bool, error) {
	return false, nil
}

var _ SuperAuthority = (*NoOpSuperAuthority)(nil)
