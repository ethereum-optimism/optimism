package driver

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

	// SequencerFencingCheckEndpoint is an optional endpoint that the sequencer can check when producing a new block to ensure
	// it has the "right" to produce this block. Useful for scenarios where you are running multiple sequencers with an election
	// protocol and you want to validate that this sequencer is currently the leader.
	// The sequencer will reach out to this endpoint with an HTTP GET request and should receive a 200.
	SequencerFencingCheckEndpoint string `json:"sequencer_fencing_check_endpoint"`
}
