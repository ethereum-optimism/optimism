package eth

// Types crossing the supernode's interop test-control surface
// (apis.SupernodeInteropTestAPI). They exist so that surface can be answered by
// a supernode in another process over RPC as readily as by an in-process one:
// every field is plain data, and nothing here is a handle on a running object.

// SupernodeInteropStatus is one snapshot of the test-visible state of a
// supernode's interop verification activity.
//
// It is a single snapshot rather than a method per field because that is how
// tests read it: they poll for "has cold-start finished, and where did
// verification start" as one question, and a round trip per field would let the
// answers disagree with each other.
type SupernodeInteropStatus struct {
	// BackfillAttempts counts cold-start initialization attempts made since the
	// verification activity most recently started. Tests use it to confirm the
	// cold-start retry loop has engaged.
	BackfillAttempts int32 `json:"backfill_attempts"`

	// BackfillCompleted reports whether cold-start initialization has finished,
	// either by running a log backfill or by resuming off existing state and
	// skipping it. The timestamps below are only meaningful once it is true.
	BackfillCompleted bool `json:"backfill_completed"`

	// ActivationTimestamp is the configured interop activation timestamp.
	ActivationTimestamp uint64 `json:"activation_timestamp"`

	// VerificationStartTimestamp is the L2 timestamp the verification loop began
	// at on its most recent start. Zero before initialization completes.
	VerificationStartTimestamp uint64 `json:"verification_start_timestamp"`

	// FirstVerifiableTimestamp is the lowest L2 timestamp the verifier covers.
	// Zero before initialization completes.
	FirstVerifiableTimestamp uint64 `json:"first_verifiable_timestamp"`
}

// SupernodeSealedBlock is one block a supernode has sealed into its interop
// logs DB.
type SupernodeSealedBlock struct {
	ID BlockID `json:"id"`

	// Timestamp is the sealed block's L2 timestamp.
	Timestamp uint64 `json:"timestamp"`
}

// SupernodeSealedBlocks is the extent of one chain's interop logs DB: the
// earliest and most recent blocks the supernode has sealed for it.
//
// Both ends arrive together because the assertion they serve is about the span
// between them — that backfill reached back far enough and handed off far
// enough forward — and reading the ends in separate calls would let the DB move
// underneath the comparison.
type SupernodeSealedBlocks struct {
	// First is the earliest sealed block. Only meaningful when HasBlocks.
	First SupernodeSealedBlock `json:"first"`

	// Latest is the most recent sealed block. Only meaningful when HasBlocks.
	Latest SupernodeSealedBlock `json:"latest"`

	// HasBlocks is false when the chain's logs DB holds no sealed blocks yet.
	// It is not an error: a supernode that has not backfilled anything for a
	// chain answers truthfully rather than failing.
	HasBlocks bool `json:"has_blocks"`
}
