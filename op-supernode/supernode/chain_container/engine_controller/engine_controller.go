package engine_controller

import (
	"context"
	"errors"
	"fmt"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// EngineController abstracts access to the L2 execution layer
type EngineController interface {
	// SafeBlockAtTimestamp returns the L2 block ref for the block at or before the given timestamp,
	// clamped to the current SAFE head.
	// Must return ethereum.NotFound if there is no safe block at the specified timestamp.
	SafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error)
	// OutputV0AtBlockNumber returns the output preimage for the given L2 block number.
	OutputV0AtBlockNumber(ctx context.Context, num uint64) (*eth.OutputV0, error)
	// RewindToTimestamp rewinds the L2 execution layer to block at or before the given timestamp.
	RewindToTimestamp(ctx context.Context, timestamp uint64) error
	// Close releases any underlying RPC resources.
	Close() error
}

// l2Provider captures the subset of the engine client we rely on.
type l2Provider interface {
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	OutputV0AtBlockNumber(ctx context.Context, blockNum uint64) (*eth.OutputV0, error)
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error)
	Close()
}

type simpleEngineController struct {
	l2     l2Provider
	rollup *rollup.Config
	log    gethlog.Logger
}

// NewEngineControllerWithL2 wraps an existing L2 provider.
func NewEngineControllerWithL2(l2 l2Provider) EngineController {
	return &simpleEngineController{l2: l2, log: gethlog.New()}
}

func NewEngineControllerWithL2AndRollup(l2 l2Provider, rollup *rollup.Config) EngineController {
	return &simpleEngineController{l2: l2, rollup: rollup, log: gethlog.New()}
}

// NewEngineControllerFromConfig builds an engine client from the op-node L2 endpoint config.
// This creates a separate connection (not passed as an override to op-node).
func NewEngineControllerFromConfig(ctx context.Context, log gethlog.Logger, vncfg *opnodecfg.Config) (EngineController, error) {
	rpc, engCfg, err := vncfg.L2.Setup(ctx, log, &vncfg.Rollup, &opmetrics.NoopRPCMetrics{})
	if err != nil {
		return nil, err
	}
	eng, err := sources.NewEngineClient(rpc, log, nil, engCfg)
	if err != nil {
		return nil, err
	}
	return &simpleEngineController{l2: eng, rollup: &vncfg.Rollup, log: log}, nil
}

var (
	ErrNoEngineClient = errors.New("engine client not initialized")
	ErrNoRollupConfig = errors.New("rollup config not available")

	// Rewind errors
	ErrRewindTargetBlockNotFound      = errors.New("failed to get target block at timestamp")
	ErrRewindComputeTargetsFailed     = errors.New("failed to compute rewind targets")
	ErrRewindInsertSyntheticFailed    = errors.New("failed to insert synthetic payload")
	ErrRewindSyntheticPayloadRejected = errors.New("synthetic payload rejected by engine")
	ErrRewindFCUSyntheticFailed       = errors.New("failed to FCU to synthetic block")
	ErrRewindFCUTargetFailed          = errors.New("failed to FCU to target block")
	ErrRewindVerificationFailed       = errors.New("rewind state verification failed")
	ErrRewindFCURejected              = errors.New("forkchoice update rejected by engine")
)

func (e *simpleEngineController) blockNumberAtTimestamp(ts uint64) (uint64, error) {
	if e.rollup == nil {
		return 0, ErrNoRollupConfig
	}
	// Compute the target block directly from rollup config
	return e.rollup.TargetBlockNumber(ts)
}

func (e *simpleEngineController) blockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	num, err := e.blockNumberAtTimestamp(ts)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	return e.l2.L2BlockRefByNumber(ctx, num)
}

// SafeBlockAtTimestamp returns the L2 block ref for the block at or before the given timestamp,
// clamped to the current SAFE head. Must return ethereum.NotFound if no safe block is available at the timestamp.
func (e *simpleEngineController) SafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	if e.l2 == nil {
		return eth.L2BlockRef{}, ErrNoEngineClient
	}
	num, err := e.blockNumberAtTimestamp(ts)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	safeHead, err := e.l2.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if num > safeHead.Number {
		e.log.Warn("engine_controller: target block number exceeds safe head", "targetBlockNumber", num, "safeHead", safeHead.Number)
		return eth.L2BlockRef{}, ethereum.NotFound
	}
	e.log.Debug("engine_controller: computed safe block number from timestamp",
		"timestamp", ts, "targetBlockNumber", num, "safeHead", safeHead.Number, "safeHeadErr", err)
	return e.l2.L2BlockRefByNumber(ctx, num)
}

func (e *simpleEngineController) OutputV0AtBlockNumber(ctx context.Context, num uint64) (*eth.OutputV0, error) {
	if e.l2 == nil {
		return nil, ErrNoEngineClient
	}
	// Prefer payload WithdrawalsRoot to avoid eth_getProof requirement on compatible nodes
	env, err := e.l2.PayloadByNumber(ctx, num)
	if e.log != nil {
		if err != nil {
			e.log.Debug("engine_controller: payload fetch failed, will try fallback if needed", "blockNumber", num, "err", err)
		} else if env == nil || env.ExecutionPayload == nil {
			e.log.Debug("engine_controller: payload missing, will try fallback", "blockNumber", num)
		} else if env.ExecutionPayload.WithdrawalsRoot == nil {
			e.log.Debug("engine_controller: payload has no withdrawals root (pre-Isthmus?), will try fallback", "blockNumber", num)
		} else {
			e.log.Debug("engine_controller: payload contains withdrawals root; using payload-based OutputV0", "blockNumber", num)
		}
	}
	if err == nil && env != nil && env.ExecutionPayload != nil && env.ExecutionPayload.WithdrawalsRoot != nil {
		p := env.ExecutionPayload
		out := &eth.OutputV0{
			StateRoot:                p.StateRoot,
			MessagePasserStorageRoot: eth.Bytes32(*p.WithdrawalsRoot),
			BlockHash:                p.BlockHash,
		}
		return out, nil
	}
	// Fallback to proof-based method if payload does not include WithdrawalsRoot
	if e.log != nil {
		e.log.Debug("engine_controller: falling back to proof-based OutputV0", "blockNumber", num)
	}
	return e.l2.OutputV0AtBlockNumber(ctx, num)
}

// RewindToTimestamp rewinds the L2 execution layer to the block at or before the given timestamp.
//
// The rewind is performed in two steps:
//  1. Insert a synthetic block (modified fee recipient) and FCU to it, which triggers a reorg
//     that orphans all blocks after the target.
//  2. FCU back to the original target block, completing the rewind.
//
// TODO: in future, we could push the implementation into the engine itself which would reduce the
// number of RPC calls required and remove the need for the synthetic block to be inserted.
func (e *simpleEngineController) RewindToTimestamp(ctx context.Context, timestamp uint64) error {
	if e.l2 == nil {
		return ErrNoEngineClient
	}

	// Step 0: infer the target block:
	// [n-1,parent] <-- [n,target] <-- [m>n,unsafe]
	targetBlock, err := e.blockAtTimestamp(ctx, timestamp)
	if err != nil {
		return fmt.Errorf("%w %d: %w", ErrRewindTargetBlockNotFound, timestamp, err)
	}

	// Step 1: Insert a synthetic block (modified fee recipient) which
	// is built on the parent of the target block:
	// [n-1,parent] <-- [n,target] <--...<-- [m>n,unsafe]
	//
	//                 [n,synthetic]
	syntheticBlockHash, err := e.insertSyntheticPayload(ctx, targetBlock.Number)
	if err != nil {
		return err
	}

	// Step 2: compute rewind targets for safe and finalized heads, ensuring they do not go forwards:
	targetSafeBlock, targetFinalizedBlock, err := e.computeRewindTargets(ctx, targetBlock)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRewindComputeTargetsFailed, err)
	}

	// Step 3: FCU to the synthetic block to trigger a reorg, removing the target block
	// from the canonical chain.
	// We use the parent hash of the target block as the safe and finalized block
	// in the FCU since these are guaranteed to be in the canonical chain of the synthetic block.
	// [n-1,parent]   [n,target]
	//      |\
	//       \_______ [n,synthetic,unsafe]
	parentHash := targetBlock.ParentHash
	if err := e.forkchoiceUpdate(ctx, syntheticBlockHash, parentHash, parentHash); err != nil {
		return fmt.Errorf("%w: %w", ErrRewindFCUSyntheticFailed, err)
	}
	e.log.Info("executed FCU to synthetic block", "syntheticHead", syntheticBlockHash, "safe", parentHash, "finalized", parentHash)

	// Step 4: FCU to the actual target block
	// [n-1,parent] <-- [n,target, unsafe]
	//
	//                  [n,synthetic]
	if err := e.forkchoiceUpdate(ctx, targetBlock.Hash, targetSafeBlock.Hash, targetFinalizedBlock.Hash); err != nil {
		return fmt.Errorf("%w: %w", ErrRewindFCUTargetFailed, err)
	}
	e.log.Info("executed FCU to target block", "head", targetBlock.Hash, "safe", targetSafeBlock.Hash, "finalized", targetFinalizedBlock.Hash)

	// Step 5: Verify the rewind state
	if err := e.verifyRewindState(ctx, targetBlock, targetSafeBlock, targetFinalizedBlock); err != nil {
		return fmt.Errorf("%w: %w", ErrRewindVerificationFailed, err)
	}
	return nil
}

// computeRewindTargets determines the safe and finalized block targets for the rewind.
// Safe and finalized are clamped to not move forward (only backward or stay the same).
func (e *simpleEngineController) computeRewindTargets(ctx context.Context, targetBlock eth.L2BlockRef) (safe, finalized eth.L2BlockRef, err error) {
	currentSafe, err := e.l2.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("failed to get current safe block: %w", err)
	}

	currentFinalized, err := e.l2.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("failed to get current finalized block: %w", err)
	}

	return earliest(currentSafe, targetBlock), earliest(currentFinalized, targetBlock), nil
}

// insertSyntheticPayload creates and inserts a synthetic block derived from the block at the given number.
// The synthetic block has a modified fee recipient to produce a different block hash.
// Returns the hash of the synthetic block.
func (e *simpleEngineController) insertSyntheticPayload(ctx context.Context, blockNumber uint64) (common.Hash, error) {
	envelope, err := e.l2.PayloadByNumber(ctx, blockNumber)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get payload for block %d: %w", blockNumber, err)
	}

	// Modify the payload to create a synthetic block with a different hash
	envelope.ExecutionPayload.FeeRecipient = common.MaxAddress
	syntheticHash, _ := envelope.CheckBlockHash()
	envelope.ExecutionPayload.BlockHash = syntheticHash

	status, err := e.l2.NewPayload(ctx, envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
	if err != nil {
		return common.Hash{}, fmt.Errorf("%w: %w", ErrRewindInsertSyntheticFailed, err)
	}
	if status.Status != eth.ExecutionValid {
		return common.Hash{}, fmt.Errorf("%w: status=%s", ErrRewindSyntheticPayloadRejected, status.Status)
	}

	return syntheticHash, nil
}

// forkchoiceUpdate sends a forkchoice update to the engine and validates the response.
func (e *simpleEngineController) forkchoiceUpdate(ctx context.Context, head, safe, finalized common.Hash) error {
	fcs := eth.ForkchoiceState{
		HeadBlockHash:      head,
		SafeBlockHash:      safe,
		FinalizedBlockHash: finalized,
	}
	res, err := e.l2.ForkchoiceUpdate(ctx, &fcs, nil)
	if err != nil {
		return err
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		return fmt.Errorf("%w: status=%s", ErrRewindFCURejected, res.PayloadStatus.Status)
	}
	return nil
}

// verifyRewindState checks that the engine state matches the expected targets after a rewind.
func (e *simpleEngineController) verifyRewindState(ctx context.Context, targetUnsafe, targetSafe, targetFinalized eth.L2BlockRef) error {
	unsafe, err := e.l2.L2BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to verify unsafe block: %w", err)
	}
	if unsafe.Number != targetUnsafe.Number {
		return fmt.Errorf("unexpected unsafe block number: got %d, want %d", unsafe.Number, targetUnsafe.Number)
	}

	safe, err := e.l2.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return fmt.Errorf("failed to verify safe block: %w", err)
	}
	if safe.Number != targetSafe.Number {
		return fmt.Errorf("unexpected safe block number: got %d, want %d", safe.Number, targetSafe.Number)
	}

	finalized, err := e.l2.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return fmt.Errorf("failed to verify finalized block: %w", err)
	}
	if finalized.Number != targetFinalized.Number {
		return fmt.Errorf("unexpected finalized block number: got %d, want %d", finalized.Number, targetFinalized.Number)
	}

	return nil
}

func (e *simpleEngineController) Close() error {
	if e.l2 != nil {
		e.l2.Close()
	}
	return nil
}

// Interface conformance assertion
var _ EngineController = (*simpleEngineController)(nil)

func earliest(a, b eth.L2BlockRef) eth.L2BlockRef {
	if a.Number < b.Number {
		return a
	}
	return b
}
