package driver

import "github.com/ethereum-optimism/optimism/op-node/rollup/sequencing"

// aliases to not disrupt op-conductor code
var (
	ErrSequencerAlreadyStarted = sequencing.ErrSequencerAlreadyStarted
	ErrSequencerAlreadyStopped = sequencing.ErrSequencerAlreadyStopped
)
