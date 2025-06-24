package syncnode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/interop/indexing"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type RPCSyncNode struct {
	name      string
	cl        client.RPC
	opts      []client.RPCOption
	logger    log.Logger
	dialSetup *RPCDialSetup
}

func NewRPCSyncNode(name string, cl client.RPC, opts []client.RPCOption, logger log.Logger, dialSetup *RPCDialSetup) *RPCSyncNode {
	return &RPCSyncNode{
		name:      name,
		cl:        cl,
		opts:      opts,
		logger:    logger,
		dialSetup: dialSetup,
	}
}

var (
	_ SyncSource  = (*RPCSyncNode)(nil)
	_ SyncControl = (*RPCSyncNode)(nil)
	_ SyncNode    = (*RPCSyncNode)(nil)
)

func (rs *RPCSyncNode) ReconnectRPC(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*60)
	defer cancel()
	cl, err := client.NewRPC(ctx, rs.logger, rs.dialSetup.Endpoint, rs.opts...)
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}
	rs.cl = cl
	return nil
}

func (rs *RPCSyncNode) BlockRefByNumber(ctx context.Context, number uint64) (eth.BlockRef, error) {
	var out *eth.BlockRef
	err := rs.cl.CallContext(ctx, &out, "interop_blockRefByNumber", number)
	if err != nil {
		return eth.BlockRef{}, eth.MaybeAsNotFoundErr(err)
	}
	return *out, nil
}

func (rs *RPCSyncNode) FetchReceipts(ctx context.Context, blockHash common.Hash) (gethtypes.Receipts, error) {
	var out gethtypes.Receipts
	err := rs.cl.CallContext(ctx, &out, "interop_fetchReceipts", blockHash)
	if err != nil {
		return nil, eth.MaybeAsNotFoundErr(err)
	}
	return out, nil
}

func (rs *RPCSyncNode) ChainID(ctx context.Context) (eth.ChainID, error) {
	var chainID eth.ChainID
	err := rs.cl.CallContext(ctx, &chainID, "interop_chainID")
	return chainID, err
}

func (rs *RPCSyncNode) OutputV0AtTimestamp(ctx context.Context, timestamp uint64) (*eth.OutputV0, error) {
	var out *eth.OutputV0
	err := rs.cl.CallContext(ctx, &out, "interop_outputV0AtTimestamp", timestamp)
	return out, err
}

func (rs *RPCSyncNode) PendingOutputV0AtTimestamp(ctx context.Context, timestamp uint64) (*eth.OutputV0, error) {
	var out *eth.OutputV0
	err := rs.cl.CallContext(ctx, &out, "interop_pendingOutputV0AtTimestamp", timestamp)
	return out, err
}

func (rs *RPCSyncNode) L2BlockRefByTimestamp(ctx context.Context, timestamp uint64) (eth.L2BlockRef, error) {
	var out eth.L2BlockRef
	err := rs.cl.CallContext(ctx, &out, "interop_l2BlockRefByTimestamp", timestamp)
	return out, err
}

func (rs *RPCSyncNode) String() string {
	return rs.name
}

func (rs *RPCSyncNode) SubscribeEvents(ctx context.Context, dest chan *types.IndexingEvent) (ethereum.Subscription, error) {
	return rpc.SubscribeStream(ctx, "interop", rs.cl, dest, "events")
}

// PullEvent pulls an event, as alternative to an event-subscription with SubscribeEvents.
// This returns an io.EOF error if no new events are available.
func (rs *RPCSyncNode) PullEvent(ctx context.Context) (*types.IndexingEvent, error) {
	var out *types.IndexingEvent
	err := rs.cl.CallContext(ctx, &out, "interop_pullEvent")
	var x gethrpc.Error
	if err != nil {
		if errors.As(err, &x) && x.ErrorCode() == rpc.OutOfEventsErrCode {
			return nil, io.EOF
		}
		return nil, err
	}
	return out, nil
}

func (rs *RPCSyncNode) UpdateCrossUnsafe(ctx context.Context, id eth.BlockID) error {
	return rs.cl.CallContext(ctx, nil, "interop_updateCrossUnsafe", id)
}

func (rs *RPCSyncNode) UpdateCrossSafe(ctx context.Context, derived eth.BlockID, source eth.BlockID) error {
	return rs.cl.CallContext(ctx, nil, "interop_updateCrossSafe", derived, source)
}

func (rs *RPCSyncNode) UpdateFinalized(ctx context.Context, id eth.BlockID) error {
	return rs.cl.CallContext(ctx, nil, "interop_updateFinalized", id)
}

func (rs *RPCSyncNode) InvalidateBlock(ctx context.Context, seal types.BlockSeal) error {
	return rs.cl.CallContext(ctx, nil, "interop_invalidateBlock", seal)
}

func (rs *RPCSyncNode) Reset(ctx context.Context, lUnsafe, xUnsafe, lSafe, xSafe, finalized eth.BlockID) error {
	return rs.cl.CallContext(ctx, nil, "interop_reset", lUnsafe, xUnsafe, lSafe, xSafe, finalized)
}

func (rs *RPCSyncNode) ResetPreInterop(ctx context.Context) error {
	return rs.cl.CallContext(ctx, nil, "interop_resetPreInterop")
}

func (rs *RPCSyncNode) ProvideL1(ctx context.Context, nextL1 eth.BlockRef) error {
	return rs.cl.CallContext(ctx, nil, "interop_provideL1", nextL1)
}

func (rs *RPCSyncNode) AnchorPoint(ctx context.Context) (types.DerivedBlockRefPair, error) {
	var (
		out     types.DerivedBlockRefPair
		jsonErr gethrpc.Error
	)
	err := rs.cl.CallContext(ctx, &out, "interop_anchorPoint")
	// Translate an interop-inactive error into a ErrFuture.
	if errors.As(err, &jsonErr) && jsonErr.ErrorCode() == indexing.InteropInactiveRPCErrCode {
		return types.DerivedBlockRefPair{}, types.ErrFuture
	}
	return out, err
}

// Contains returns no error iff the specified logHash is recorded in the specified blockNum and logIdx.
// If the log is out of reach and the block is complete, an ErrConflict is returned.
// If the log is out of reach and the block is not complete, an ErrFuture is returned.
// If the log is determined to conflict with the canonical chain, then ErrConflict is returned.
// logIdx is the index of the log in the array of all logs in the block.
// This can be used to check the validity of cross-chain interop events.
// The block-seal of the blockNum block that the log was included in is returned.
func (rs *RPCSyncNode) Contains(ctx context.Context, query types.ContainsQuery) (types.BlockSeal, error) {
	chainID, err := rs.ChainID(ctx)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("failed to get chain ID for verifying access with RPC: %w", err)
	}

	blockRef, err := rs.BlockRefByNumber(ctx, query.BlockNum)
	if err != nil {
		return types.BlockSeal{}, types.ErrFuture
	}

	log, err := rs.getLogAtIndex(ctx, blockRef.Hash, query.LogIdx)
	if err != nil {
		return types.BlockSeal{}, types.ErrConflict
	}

	logHash := processors.LogToLogHash(log)
	entryChecksum := types.ChecksumArgs{
		BlockNumber: query.BlockNum,
		LogIndex:    query.LogIdx,
		Timestamp:   blockRef.Time,
		ChainID:     chainID,
		LogHash:     logHash,
	}.Checksum()
	if entryChecksum != query.Checksum {
		return types.BlockSeal{}, types.ErrConflict
	}

	return types.BlockSeal{
		Hash:      blockRef.Hash,
		Number:    blockRef.Number,
		Timestamp: blockRef.Time,
	}, nil
}

func (rs *RPCSyncNode) getLogAtIndex(ctx context.Context, blockHash common.Hash, logIndex uint32) (*gethtypes.Log, error) {
	receipts, err := rs.FetchReceipts(ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch receipts for verifying access with RPC: %w", err)
	}

	log, err := eth.GetLogAtIndex(receipts, uint(logIndex))
	if err != nil {
		return nil, fmt.Errorf("failed to get log index for verifying access with RPC: %w", err)
	}
	return log, err
}
