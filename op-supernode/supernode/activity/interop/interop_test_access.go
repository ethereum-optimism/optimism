package interop

// This file collects Interop methods that are intended for integration tests
// and debugging tooling only. They expose or mutate otherwise-internal state
// and must not be called by production code paths. Keep all such accessors
// in this file so the boundary is easy to audit.

import (
	"fmt"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ---------------------------------------------------------------------------
// Pause / Resume
// ---------------------------------------------------------------------------

// PauseAt sets a timestamp at which the interop activity should pause.
// When progressInterop encounters this timestamp or any later timestamp, it
// returns early without processing. Uses >= check so that if the activity is
// already beyond the pause point, it will still stop. Pass 0 to clear the
// pause (equivalent to calling Resume).
func (i *Interop) PauseAt(ts uint64) {
	i.pauseAtTimestamp.Store(ts)
	i.log.Info("interop pause set", "pauseAtTimestamp", ts)
}

// Resume clears any pause timestamp, allowing normal processing to continue.
func (i *Interop) Resume() {
	i.pauseAtTimestamp.Store(0)
	i.log.Info("interop pause cleared")
}

// ---------------------------------------------------------------------------
// Backfill observability
// ---------------------------------------------------------------------------

// BackfillAttempts returns the number of times advanceColdStartInit has been
// invoked since the most recent Start. Integration tests use it to confirm
// the cold-start retry loop has engaged.
func (i *Interop) BackfillAttempts() int32 {
	return i.backfillAttempts.Load()
}

// BackfillCompleted reports whether cold-start init has finished (backfill
// ran, or resume skipped it). Integration tests gate downstream assertions
// on this.
func (i *Interop) BackfillCompleted() bool {
	return i.backfillCompleted.Load()
}

// ---------------------------------------------------------------------------
// Activation-timestamp inspection
// ---------------------------------------------------------------------------

// ActivationTimestamp returns the immutable protocol-defined interop
// activation timestamp. This is the value that fronts protocol-facing RPC
// responses and never advances at runtime.
func (i *Interop) ActivationTimestamp() uint64 {
	return i.activationTimestamp
}

// VerificationStartTimestamp returns the L2 timestamp at which the main loop
// begins verification on the most recent Start. Returns 0 before
// initialization completes.
func (i *Interop) VerificationStartTimestamp() uint64 {
	if !i.initialized.Load() {
		return 0
	}
	return i.verificationStartTimestamp
}

// FirstVerifiableTimestamp returns the lowest timestamp the verifier covers
// (verifiedDB.FirstTimestamp when commits exist, else
// VerificationStartTimestamp). Returns 0 before initialization completes.
func (i *Interop) FirstVerifiableTimestamp() uint64 {
	ts, err := i.firstVerifiableTimestamp()
	if err != nil {
		return 0
	}
	return ts
}

// ---------------------------------------------------------------------------
// LogsDB inspection
// ---------------------------------------------------------------------------

// FirstSealedBlock returns the earliest block sealed in the logs DB for the
// given chain, along with its timestamp. Returns an error if the chain is
// unknown or the logs DB is empty.
func (i *Interop) FirstSealedBlock(chainID eth.ChainID) (messages.BlockSeal, error) {
	db, ok := i.logsDBs[chainID]
	if !ok {
		return messages.BlockSeal{}, fmt.Errorf("interop: no logs DB for chain %s", chainID)
	}
	seal, err := db.FirstSealedBlock()
	if err != nil {
		return messages.BlockSeal{}, fmt.Errorf("interop: first sealed block for chain %s: %w", chainID, err)
	}
	return seal, nil
}

// LatestSealedBlock returns the most recent block sealed in the logs DB for
// the given chain along with its timestamp. Returns an error if the chain is
// unknown and (zero, false) if the DB is empty.
func (i *Interop) LatestSealedBlock(chainID eth.ChainID) (messages.BlockSeal, bool, error) {
	db, ok := i.logsDBs[chainID]
	if !ok {
		return messages.BlockSeal{}, false, fmt.Errorf("interop: no logs DB for chain %s", chainID)
	}
	id, has := db.LatestSealedBlock()
	if !has {
		return messages.BlockSeal{}, false, nil
	}
	seal, err := db.FindSealedBlock(id.Number)
	if err != nil {
		return messages.BlockSeal{}, false, fmt.Errorf("interop: latest sealed block for chain %s: find %d: %w", chainID, id.Number, err)
	}
	return seal, true, nil
}
