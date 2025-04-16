package syncnode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/rpc"

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

var _ SyncSource = (*RPCSyncNode)(nil)
var _ SyncControl = (*RPCSyncNode)(nil)
var _ SyncNode = (*RPCSyncNode)(nil)

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
		var jsonErr gethrpc.Error
		if errors.As(err, &jsonErr) {
			if jsonErr.ErrorCode() == 0 { // TODO
				return eth.BlockRef{}, ethereum.NotFound
			}
		}
		return eth.BlockRef{}, err
	}
	return *out, nil
}

func (rs *RPCSyncNode) FetchReceipts(ctx context.Context, blockHash common.Hash) (gethtypes.Receipts, error) {
	var out gethtypes.Receipts
	err := rs.cl.CallContext(ctx, &out, "interop_fetchReceipts", blockHash)
	if err != nil {
		var jsonErr gethrpc.Error
		if errors.As(err, &jsonErr) {
			if jsonErr.ErrorCode() == 0 { // TODO
				return nil, ethereum.NotFound
			}
		}
		return nil, err
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

func (rs *RPCSyncNode) SubscribeEvents(ctx context.Context, dest chan *types.ManagedEvent) (ethereum.Subscription, error) {
	return rpc.SubscribeStream(ctx, "interop", rs.cl, dest, "events")
}

// PullEvent pulls an event, as alternative to an event-subscription with SubscribeEvents.
// This returns an io.EOF error if no new events are available.
func (rs *RPCSyncNode) PullEvent(ctx context.Context) (*types.ManagedEvent, error) {
	var out *types.ManagedEvent
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

func (rs *RPCSyncNode) ProvideL1(ctx context.Context, nextL1 eth.BlockRef) error {
	return rs.cl.CallContext(ctx, nil, "interop_provideL1", nextL1)
}

func (rs *RPCSyncNode) AnchorPoint(ctx context.Context) (types.DerivedBlockRefPair, error) {
	var out types.DerivedBlockRefPair
	err := rs.cl.CallContext(ctx, &out, "interop_anchorPoint")
	return out, err
}
