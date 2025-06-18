package superevents

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type ChainProcessEvent struct {
	ChainID eth.ChainID
	Target  uint64
	event.Ctx
}

func (ev ChainProcessEvent) String() string {
	return "chain-process"
}

type UpdateCrossUnsafeRequestEvent struct {
	ChainID eth.ChainID
	event.Ctx
}

func (ev UpdateCrossUnsafeRequestEvent) String() string {
	return "update-cross-unsafe-request"
}

type UpdateCrossSafeRequestEvent struct {
	ChainID eth.ChainID
	event.Ctx
}

func (ev UpdateCrossSafeRequestEvent) String() string {
	return "update-cross-safe-request"
}

type LocalUnsafeUpdateEvent struct {
	ChainID        eth.ChainID
	NewLocalUnsafe eth.BlockRef
	event.Ctx
}

func (ev LocalUnsafeUpdateEvent) String() string {
	return "local-unsafe-update"
}

type LocalSafeUpdateEvent struct {
	ChainID      eth.ChainID
	NewLocalSafe types.DerivedBlockSealPair
	event.Ctx
}

func (ev LocalSafeUpdateEvent) String() string {
	return "local-safe-update"
}

type CrossUnsafeUpdateEvent struct {
	ChainID        eth.ChainID
	NewCrossUnsafe types.BlockSeal
	event.Ctx
}

func (ev CrossUnsafeUpdateEvent) String() string {
	return "cross-unsafe-update"
}

type CrossSafeUpdateEvent struct {
	ChainID      eth.ChainID
	NewCrossSafe types.DerivedBlockSealPair
	event.Ctx
}

func (ev CrossSafeUpdateEvent) String() string {
	return "cross-safe-update"
}

type FinalizedL1RequestEvent struct {
	FinalizedL1 eth.BlockRef
	event.Ctx
}

func (ev FinalizedL1RequestEvent) String() string {
	return "finalized-l1-request"
}

type FinalizedL1UpdateEvent struct {
	FinalizedL1 eth.BlockRef
	event.Ctx
}

func (ev FinalizedL1UpdateEvent) String() string {
	return "finalized-l1-update"
}

type FinalizedL2UpdateEvent struct {
	ChainID     eth.ChainID
	FinalizedL2 types.BlockSeal
	event.Ctx
}

func (ev FinalizedL2UpdateEvent) String() string {
	return "finalized-l2-update"
}

type LocalUnsafeReceivedEvent struct {
	ChainID        eth.ChainID
	NewLocalUnsafe eth.BlockRef
	event.Ctx
}

func (ev LocalUnsafeReceivedEvent) String() string {
	return "local-unsafe-received"
}

type LocalDerivedEvent struct {
	ChainID eth.ChainID
	Derived types.DerivedBlockRefPair
	NodeID  string
	event.Ctx
}

func (ev LocalDerivedEvent) String() string {
	return "local-derived"
}

type LocalDerivedOriginUpdateEvent struct {
	ChainID eth.ChainID
	Origin  eth.BlockRef
	event.Ctx
}

func (ev LocalDerivedOriginUpdateEvent) String() string {
	return "local-derived-origin-update"
}

type ResetPreInteropRequestEvent struct {
	ChainID eth.ChainID
	event.Ctx
}

func (ev ResetPreInteropRequestEvent) String() string {
	return "reset-pre-interop-request"
}

type UnsafeActivationBlockEvent struct {
	Unsafe  eth.BlockRef
	ChainID eth.ChainID
	event.Ctx
}

func (ev UnsafeActivationBlockEvent) String() string {
	return "unsafe-activation-block-received"
}

type SafeActivationBlockEvent struct {
	Safe    types.DerivedBlockRefPair
	ChainID eth.ChainID
	event.Ctx
}

func (ev SafeActivationBlockEvent) String() string {
	return "safe-activation-block-received"
}

type InvalidateLocalSafeEvent struct {
	ChainID   eth.ChainID
	Candidate types.DerivedBlockRefPair
	event.Ctx
}

func (ev InvalidateLocalSafeEvent) String() string {
	return "invalidate-local-safe"
}

type RewindL1Event struct {
	IncomingBlock eth.BlockID
	event.Ctx
}

func (ev RewindL1Event) String() string {
	return "rewind-l1"
}

type ReplaceBlockEvent struct {
	ChainID     eth.ChainID
	Replacement types.BlockReplacement
	event.Ctx
}

func (ev ReplaceBlockEvent) String() string {
	return "replace-block-event"
}

type ChainRewoundEvent struct {
	ChainID eth.ChainID
	event.Ctx
}

func (ev ChainRewoundEvent) String() string {
	return "chain-rewound"
}

type UpdateLocalSafeFailedEvent struct {
	ChainID eth.ChainID
	Err     error
	NodeID  string
	event.Ctx
}

func (ev UpdateLocalSafeFailedEvent) String() string {
	return "update-local-safe-failed"
}

type ChainIndexingContinueEvent struct {
	ChainID eth.ChainID
	event.Ctx
}

func (ev ChainIndexingContinueEvent) String() string {
	return "chain-indexing-continue"
}
