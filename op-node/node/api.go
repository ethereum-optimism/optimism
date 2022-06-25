package node

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/version"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type l2EthClient interface {
	GetBlockHeader(ctx context.Context, blockTag string) (*types.Header, error)
	// GetProof returns a proof of the account, it may return a nil result without error if the address was not found.
	GetProof(ctx context.Context, address common.Address, blockTag string) (*l2.AccountResult, error)

	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	L2BlockRefByNumber(ctx context.Context, l2Num *big.Int) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
}

type ChannelEmitter interface {
	Output(ctx context.Context, history map[derive.ChannelID]uint64, minSize uint64, maxSize uint64, maxBlocksPerChannel uint64) (*derive.BatcherChannelData, error)
}

type nodeAPI struct {
	config  *rollup.Config
	client  l2EthClient
	emitter ChannelEmitter
	log     log.Logger
}

func newNodeAPI(config *rollup.Config, l2Client l2EthClient, log log.Logger) *nodeAPI {
	return &nodeAPI{
		config: config,
		client: l2Client,
		log:    log,
	}
}

func (n *nodeAPI) OutputAtBlock(ctx context.Context, number rpc.BlockNumber) ([]eth.Bytes32, error) {
	// TODO: rpc.BlockNumber doesn't support the "safe" tag. Need a new type

	head, err := n.client.GetBlockHeader(ctx, toBlockNumArg(number))
	if err != nil {
		n.log.Error("failed to get block", "err", err)
		return nil, err
	}
	if head == nil {
		return nil, ethereum.NotFound
	}

	proof, err := n.client.GetProof(ctx, predeploys.L2ToL1MessagePasserAddr, toBlockNumArg(number))
	if err != nil {
		n.log.Error("failed to get contract proof", "err", err)
		return nil, err
	}
	if proof == nil {
		return nil, ethereum.NotFound
	}
	// make sure that the proof (including storage hash) that we retrieved is correct by verifying it against the state-root
	if err := proof.Verify(head.Root); err != nil {
		n.log.Error("invalid withdrawal root detected in block", "stateRoot", head.Root, "blocknum", number, "msg", err)
		return nil, fmt.Errorf("invalid withdrawal root hash")
	}

	var l2OutputRootVersion eth.Bytes32 // it's zero for now
	l2OutputRoot := l2.ComputeL2OutputRoot(l2OutputRootVersion, head.Hash(), head.Root, proof.StorageHash)

	return []eth.Bytes32{l2OutputRootVersion, l2OutputRoot}, nil
}

func (n *nodeAPI) Version(ctx context.Context) (string, error) {
	return version.Version + "-" + version.Meta, nil
}

func toBlockNumArg(number rpc.BlockNumber) string {
	if number == rpc.LatestBlockNumber {
		return "latest"
	}
	if number == rpc.PendingBlockNumber {
		return "pending"
	}
	return hexutil.EncodeUint64(uint64(number.Int64()))
}

type BatchBundleRequest struct {
	// History is a dictionary of channels that were previously used, with the last frame number per channel.
	// The rollup-node then finds which channels are useful to continue,
	// and adds frame data to the output to complete the channel.
	// Remaining space is used for remaining blocks which were not previously encoded in a channel.
	// After a channel times out (w.r.t. the current L1 head the rollup-node sees) blocks that are still
	// considered to be unsafe (i.e. never confirmed on L1) may again be encoded in new channels.
	History map[derive.ChannelID]uint64

	// Minimum size of the data to return
	MinSize hexutil.Uint64
	// Maximum size of the data to return
	MaxSize hexutil.Uint64

	// MaxBlocksPerChannel is the maximum number of L2 blocks that may be compressed together in a channel.
	// The output may still have multiple channels and thus more blocks.
	//
	// If the batch-submitter has trouble to submit blocks across multiple txs (e.g. many txs drop)
	// then this can be reduced, which should reduce the effect of incomplete channels.
	//
	// If this is set very large, then a many L2 blocks can be compressed together, but the L1 tx inclusion is more important
	MaxBlocksPerChannel hexutil.Uint64
}

func (n *nodeAPI) GetBatchBundle(ctx context.Context, req *BatchBundleRequest) (*derive.BatcherChannelData, error) {
	return n.emitter.Output(ctx, req.History, uint64(req.MinSize), uint64(req.MaxSize), uint64(req.MaxBlocksPerChannel))
}
