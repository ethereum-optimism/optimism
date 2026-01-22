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
	// FinalizedBlock return the Finalized L2 Block
	FinalizedBlock(ctx context.Context) (eth.L2BlockRef, error)
	// OutputV0AtBlockNumber returns the output preimage for the given L2 block number.
	OutputV0AtBlockNumber(ctx context.Context, num uint64) (*eth.OutputV0, error)
	// ForkchoiceUpdate
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState) (*eth.ForkchoiceUpdatedResult, error)
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
	NewPayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) (*eth.PayloadStatusV1, error)
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
)

// SafeBlockAtTimestamp returns the L2 block ref for the block at or before the given timestamp,
// clamped to the current SAFE head. Must return ethereum.NotFound if no safe block is available at the timestamp.
func (e *simpleEngineController) SafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	if e.l2 == nil {
		return eth.L2BlockRef{}, ErrNoEngineClient
	}
	if e.rollup == nil {
		return eth.L2BlockRef{}, ErrNoRollupConfig
	}
	// Compute the target block directly from rollup config
	num, err := e.rollup.TargetBlockNumber(ts)
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

func (e *simpleEngineController) FinalizedBlock(ctx context.Context) (eth.L2BlockRef, error) {
	if e.l2 == nil {
		return eth.L2BlockRef{}, ErrNoEngineClient
	}
	return e.l2.L2BlockRefByLabel(ctx, eth.Finalized)
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
func (e *simpleEngineController) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState) (*eth.ForkchoiceUpdatedResult, error) {
	if e.l2 == nil {
		return nil, ErrNoEngineClient
	}
	return e.l2.ForkchoiceUpdate(ctx, state, nil) // use nil PayloadAttributes to avoid triggering block building
}

func (e *simpleEngineController) RewindToTimestamp(ctx context.Context, timestamp uint64) error {
	if e.l2 == nil {
		return ErrNoEngineClient
	}

	targetSafeBlock, err := e.SafeBlockAtTimestamp(ctx, timestamp)
	if err != nil {
		return err
	}

	currentFinalizedBlock, err := e.l2.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return err
	}

	// Don't move finalized block forward, only back:
	// TODO convert this to a helper / lambda
	var targetFinalizedBlock eth.L2BlockRef
	if currentFinalizedBlock.Time < targetSafeBlock.Number {
		targetFinalizedBlock = currentFinalizedBlock
	} else {
		targetFinalizedBlock = targetSafeBlock
	}

	syntheticPayload, err := e.l2.PayloadByNumber(ctx, targetSafeBlock.Number)
	if err != nil {
		return err
	}

	// create a synthetic block to FCU to,
	// this will reorg out the unwanted blocks:
	syntheticPayload.ExecutionPayload.FeeRecipient = common.Address{}
	syntheticBlockHash, _ := syntheticPayload.CheckBlockHash()
	syntheticPayload.ExecutionPayload.BlockHash = syntheticBlockHash
	status, err := e.l2.NewPayload(ctx, syntheticPayload)
	if err != nil {
		return err
	}

	if status.Status != eth.ExecutionValid {
		return fmt.Errorf("failed to create synthetic payload: %+v", status)
	}

	fcs := eth.ForkchoiceState{
		HeadBlockHash:      syntheticBlockHash,
		SafeBlockHash:      syntheticBlockHash,
		FinalizedBlockHash: targetFinalizedBlock.Hash,
	}
	res, err := e.l2.ForkchoiceUpdate(ctx, &fcs, nil)
	if err != nil {
		return fmt.Errorf("failed to FCU to synthetic block: %w", err)
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		return fmt.Errorf("failed to FCU to synthetic block: %+v", res.PayloadStatus)
	}

	// Next, FCU to the targetSafeBlock
	fcs = eth.ForkchoiceState{
		HeadBlockHash:      targetSafeBlock.Hash,
		SafeBlockHash:      targetSafeBlock.Hash,
		FinalizedBlockHash: targetFinalizedBlock.Hash,
	}
	res, err = e.l2.ForkchoiceUpdate(ctx, &fcs, nil)
	if err != nil {
		return fmt.Errorf("failed to rewind engine: %w", err)
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		return fmt.Errorf("failed to rewind engine: %+v", res.PayloadStatus)
	}

	resultingUnsafeBlock, err := e.l2.L2BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return err
	}

	if resultingUnsafeBlock.Number != targetSafeBlock.Number {
		return fmt.Errorf("unexpected unsafe block number: %d != %d", resultingUnsafeBlock.Number, targetUnsafeBlock.Number)
	}

	resultingSafeBlock, err := e.l2.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return err
	}

	if resultingSafeBlock.Number != targetSafeBlock.Number {
		return fmt.Errorf("unexpected safe block number: %d != %d", resultingSafeBlock.Number, targetSafeBlock.Number)
	}

	resultingFinalizedBlock, err := e.l2.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return err
	}

	if resultingFinalizedBlock.Number != targetFinalizedBlock.Number {
		return fmt.Errorf("unexpected finalized block number: %d != %d", resultingFinalizedBlock.Number, targetFinalizedBlock.Number)
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
