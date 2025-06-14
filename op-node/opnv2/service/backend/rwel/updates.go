package rwel

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// LocalUnsafeUpdateEvent signals that the given block is now considered local-unsafe.
// This is a subset of the ForkchoiceUpdateEvent event, emitted only when local-unsafe changes.
type LocalUnsafeUpdateEvent struct {
	Ref eth.BlockRef
}

func (ev LocalUnsafeUpdateEvent) String() string {
	return "local-unsafe-update"
}

// CrossSafeUpdateEvent signals that a block has been marked as cross-safe.
// This is a subset of the ForkchoiceUpdateEvent event, emitted only when cross-safety changes.
type CrossSafeUpdateEvent struct {
	CrossSafe eth.BlockRef
}

func (ev CrossSafeUpdateEvent) String() string {
	return "cross-safe-update"
}

// FinalizedUpdateEvent signals that a block has been marked as finalized.
// This is a subset of the ForkchoiceUpdateEvent event, emitted only when finality changes.
type FinalizedUpdateEvent struct {
	Ref eth.BlockRef
}

func (ev FinalizedUpdateEvent) String() string {
	return "finalized-update"
}

type ForkchoiceUpdateEvent struct {
	LocalUnsafe eth.BlockRef
	CrossSafe   eth.BlockRef
	Finalized   eth.BlockRef
}

func (ev ForkchoiceUpdateEvent) String() string {
	return "forkchoice-update"
}

// SyncingUpdateEvent is emitted whenever the engine says it is syncing.
type SyncingUpdateEvent struct {
	SyncTarget eth.BlockID
}

func (ev SyncingUpdateEvent) String() string {
	return "sync-target-update"
}

// NoSyncingEvent is emitted whenever the engine signals success without syncing status.
type NoSyncingEvent struct {
	LocalUnsafe eth.BlockRef
}

func (ev NoSyncingEvent) String() string {
	return "no-syncing"
}
