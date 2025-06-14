package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Make the engine sync from L1.
// To sync, it will first need a reference point from the DB, to start syncing from.
// The DB is the only source of truth of which L2 block is derived from which L1 block.
// If the DB does not have this data anymore, then we prefer to fall back to other sync methods.
// If the other sync methods don't work,
// we can attempt to find a sync starting point using legacy methods ("find sync start and channel walk back").
func (c *RWELState) maybeSyncFromL1() {
	db, ok := c.state.ChainDB(c.ChainID())
	if !ok {
		return
	}

	// If we're at or past the DB local-safe point already,
	// then we will sync new blocks naturally as part of the DB local-safe derivation progress.
	if c.localUnsafe.Number >= db.localSafe.Derived.Number {
		return
	}

	// Ask the DB for a sync starting point.
	c.L1Sync.Emit(c.rootCtx, FindDerivationStartEvent{}, c.awaitFoundDerivationStart)

	// Syncing RWEL directly from L1:
	// for any RWEL, if RWEL local-unsafe == RWEL local-safe, and within L1-syncing-distance,
	//   then we can derive attribs for it, to sync from L1.
}

func (c *RWELState) awaitFoundDerivationStart(ctx context.Context, ev event.Event) {
	switch ev.(type) {
	case FoundDerivationStartEvent:
		break
	case Err:
		// abort
		c.L1Sync.Reset()
		return
	}

	// Find a pipeline that is not busy
	p, ok := First[*PipelineState](c.state.IterPipelines(func(state *PipelineState) bool {
		return !state.deriveTask.IsBusy()
	}))
	if !ok {
		// TODO If no pipeline is available for syncing, then backoff.
		c.L1Sync.Reset()
		return
	}

	// TODO reset the pipeline to the engine starting point
	_ = p
}

type FindDerivationStartEvent struct {
	LastL2 eth.BlockID
}

func (ev FindDerivationStartEvent) String() string {
	return "find-derivation-start"
}

type FoundDerivationStartEvent struct {
	Start types.DerivedBlockSealPair
}

func (ev FoundDerivationStartEvent) String() string {
	return "found-derivation-start"
}

// TODO: above events need to be hooked up to the local-safe DB,
// to find a derived-from source of an L2 block.
