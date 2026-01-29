package driver

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/finality"
)

type Config struct {
	// VerifierConfDepth is the distance to keep from the L1 head when reading L1 data for L2 derivation.
	VerifierConfDepth uint64 `json:"verifier_conf_depth"`

	// SequencerConfDepth is the distance to keep from the L1 head as origin when sequencing new L2 blocks.
	// If this distance is too large, the sequencer may:
	// - not adopt a L1 origin within the allowed time (rollup.Config.MaxSequencerDrift)
	// - not adopt a L1 origin that can be included on L1 within the allowed range (rollup.Config.SeqWindowSize)
	// and thus fail to produce a block with anything more than deposits.
	SequencerConfDepth uint64 `json:"sequencer_conf_depth"`

	// SequencerEnabled is true when the driver should sequence new blocks.
	SequencerEnabled bool `json:"sequencer_enabled"`

	// SequencerStopped is false when the driver should sequence new blocks.
	SequencerStopped bool `json:"sequencer_stopped"`

	// SequencerMaxSafeLag is the maximum number of L2 blocks for restricting the distance between L2 safe and unsafe.
	// Disabled if 0.
	SequencerMaxSafeLag uint64 `json:"sequencer_max_safe_lag"`

	// RecoverMode forces the sequencer to select the next L1 Origin exactly, and create an empty block,
	// to be compatible with verifiers forcefully generating the same block while catching up the sequencing window timeout.
	RecoverMode bool `json:"recover_mode"`

	// SequencerSealingDuration is the amount of the time the sequencer allocates to sealing the block
	// (i.e. it will fetch the payload from the execution engine this much prior to the block's timestamp).
	// If this is <= 0 it is automatically adjusted to 50ms.
	SequencerSealingDuration time.Duration `json:"sequencer_sealing_duration"`

	// Finalizer contains runtime configuration for finality behavior.
	Finalizer *finality.Config `json:"finalizer,omitempty"`
}
