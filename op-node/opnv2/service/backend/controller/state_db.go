package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type ChainDBState struct {
	state State

	rootCtx context.Context

	localUnsafe types.BlockSeal
	crossUnsafe types.BlockSeal
	localSafe   types.DerivedBlockSealPair
	crossSafe   types.DerivedBlockSealPair
	finalized   types.DerivedBlockSealPair

	// non-zero whenever we have an invalidated local-safe block to replace
	invalidated types.DerivedBlockRefPair
	// non-nil whenever we have invalidated something,
	// and are ready to build the replacement block.
	replacementAttributes *derive.AttributesWithParent
	// non-nil once done attributes are converted
	replacementComplete *types.BlockReplacement

	Replacement TaskStateV2

	FinalizeWork TaskStateV2

	crossUnsafeWork struct {
		TaskStateV2
		backoffState
	}
	crossSafeWork struct {
		TaskStateV2
		backoffState
	}

	chainIndexingWork struct {
		TaskStateV2
		backoffState
		// old chain indexing code does not bounce back events with same context, so we can't model tasks
	}

	chainIDState
}

func NewChainDBState(rootCtx context.Context, emitter event.Emitter, chainID eth.ChainID) *ChainDBState {
	out := new(ChainDBState)
	// TODO
	return out
}
