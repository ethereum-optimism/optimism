package interop

// This file collects Interop methods that are intended for integration tests
// and debugging tooling only. They expose or mutate otherwise-internal state
// and must not be called by production code paths. Keep all such accessors
// in this file so the boundary is easy to audit.

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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

// BackfillAttempts returns the number of times runLogBackfill has been
// invoked since the most recent Start. Integration tests use it to confirm
// the retry loop has engaged.
func (i *Interop) BackfillAttempts() int32 {
	return i.backfillAttempts.Load()
}

// BackfillCompleted reports whether the log backfill phase has finished
// (either ran and returned nil, or was skipped because logBackfillDepth
// was 0). Integration tests use it to gate assertions on downstream state
// until backfill is done.
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

// BackfillEndTimestamp returns the inclusive last timestamp whose logs were
// sealed by runLogBackfill, or 0 if backfill has not run. The main loop
// starts verification at BackfillEndTimestamp()+1 (or ActivationTimestamp()
// when backfill was skipped).
func (i *Interop) BackfillEndTimestamp() uint64 {
	return i.backfillEndTimestamp
}

// FirstVerifiableTimestamp returns the timestamp at which the main loop begins
// verification. It is intended for tests after startup has completed.
func (i *Interop) FirstVerifiableTimestamp() uint64 {
	ts, err := i.firstVerifiableTimestamp(context.Background())
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
func (i *Interop) FirstSealedBlock(chainID eth.ChainID) (suptypes.BlockSeal, error) {
	db, ok := i.logsDBs[chainID]
	if !ok {
		return suptypes.BlockSeal{}, fmt.Errorf("interop: no logs DB for chain %s", chainID)
	}
	seal, err := db.FirstSealedBlock()
	if err != nil {
		return suptypes.BlockSeal{}, fmt.Errorf("interop: first sealed block for chain %s: %w", chainID, err)
	}
	return seal, nil
}

// LatestSealedBlock returns the most recent block sealed in the logs DB for
// the given chain along with its timestamp. Returns an error if the chain is
// unknown and (zero, false) if the DB is empty.
func (i *Interop) LatestSealedBlock(chainID eth.ChainID) (suptypes.BlockSeal, bool, error) {
	db, ok := i.logsDBs[chainID]
	if !ok {
		return suptypes.BlockSeal{}, false, fmt.Errorf("interop: no logs DB for chain %s", chainID)
	}
	id, has := db.LatestSealedBlock()
	if !has {
		return suptypes.BlockSeal{}, false, nil
	}
	seal, err := db.FindSealedBlock(id.Number)
	if err != nil {
		return suptypes.BlockSeal{}, false, fmt.Errorf("interop: latest sealed block for chain %s: find %d: %w", chainID, id.Number, err)
	}
	return seal, true, nil
}
