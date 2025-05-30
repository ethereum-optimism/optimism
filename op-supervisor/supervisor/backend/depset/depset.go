package depset

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DependencySetSource interface {
	LoadDependencySet(ctx context.Context) (DependencySet, error)
}

// DependencySet is an initialized dependency set, ready to answer queries
// of what is and what is not part of the dependency set.
type DependencySet interface {
	// Chains returns the list of chains that are part of the dependency set.
	Chains() []eth.ChainID

	// HasChain determines if a chain is being tracked for interop purposes.
	// See CanExecuteAt and CanInitiateAt to check if a chain may message at a given time.
	HasChain(chainID eth.ChainID) bool

	// MessageExpiryWindow returns the message expiry window to use for this dependency set.
	MessageExpiryWindow() uint64
}
