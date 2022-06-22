package node

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/metrics"

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

// TODO: decide on sanity limit to not keep adding more blocks when the data size is huge.
// I.e. don't batch together the whole L2 chain
const MaxL2BlocksPerBatchResponse = 100

type l2EthClient interface {
	GetBlockHeader(ctx context.Context, blockTag string) (*types.Header, error)
	// GetProof returns a proof of the account, it may return a nil result without error if the address was not found.
	GetProof(ctx context.Context, address common.Address, blockTag string) (*l2.AccountResult, error)

	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	L2BlockRefByNumber(ctx context.Context, l2Num *big.Int) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
}

type nodeAPI struct {
	config *rollup.Config
	client l2EthClient
	log    log.Logger
	m      *metrics.Metrics
}

func newNodeAPI(config *rollup.Config, l2Client l2EthClient, log log.Logger, m *metrics.Metrics) *nodeAPI {
	return &nodeAPI{
		config: config,
		client: l2Client,
		log:    log,
		m:      m,
	}
}

func (n *nodeAPI) OutputAtBlock(ctx context.Context, number rpc.BlockNumber) ([]eth.Bytes32, error) {
	recordDur := n.m.RecordRPCServerRequest("optimism_outputAtBlock")
	defer recordDur()
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
	recordDur := n.m.RecordRPCServerRequest("optimism_version")
	defer recordDur()
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
	// L2History is a list of L2 blocks that are already in-flight or confirmed.
	// The rollup-node then finds the common point, and responds with that point as PrevL2BlockHash and PrevL2BlockNum.
	// The L2 history is read in order of the provided hashes, which may contain arbitrary gaps and skips.
	// The first common hash will be the continuation point.
	// A batch-submitter may search the history using gaps to find a common point even with deep reorgs.
	L2History []common.Hash

	MinSize hexutil.Uint64
	MaxSize hexutil.Uint64
}

type BatchBundleResponse struct {
	PrevL2BlockHash common.Hash
	PrevL2BlockNum  hexutil.Uint64

	// LastL2BlockHash is the L2 block hash of the last block in the bundle.
	// This is the ideal continuation point for the next batch submission.
	// It will equal PrevL2BlockHash if there are no batches to submit.
	LastL2BlockHash common.Hash
	LastL2BlockNum  hexutil.Uint64

	// Bundle represents the encoded bundle of batches.
	// Each batch represents the inputs of a L2 block, i.e. a batch of L2 transactions (excl. deposits and such).
	// The bundle encoding supports versioning and compression.
	// The rollup-node determines the version to use based on configuration.
	// Bundle is empty if there is nothing to submit.
	Bundle hexutil.Bytes
}

func (n *nodeAPI) GetBatchBundle(ctx context.Context, req *BatchBundleRequest) (*BatchBundleResponse, error) {
	var found eth.BlockID
	// First find the common point with L2 history so far
	for i, h := range req.L2History {
		l2Ref, err := n.client.L2BlockRefByHash(ctx, h)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) { // on reorgs and such we expect that blocks may be missing
				continue
			}
			return nil, fmt.Errorf("failed to check L2 history for block hash %d in request %s: %v", i, h, err)
		}
		// found a block that exists! Now make sure it's really a canonical block of L2
		canonBlock, err := n.client.L2BlockRefByNumber(ctx, big.NewInt(int64(l2Ref.Number)))
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				continue
			}
			return nil, fmt.Errorf("failed to check L2 history for block number %d, expecting block %s: %v", l2Ref.Number, h, err)
		}
		if canonBlock.Hash == h {
			// found a common canonical block!
			found = eth.BlockID{Hash: canonBlock.Hash, Number: canonBlock.Number}
			break
		}
	}
	if found == (eth.BlockID{}) { // none of the L2 history could be found.
		return nil, ethereum.NotFound
	}

	var bundleBuilder = NewBundleBuilder(found)
	var totalBatchSizeBytes uint64
	var hasLargeNextBatch bool
	// Now continue fetching the next blocks, and build batches, until we either run out of space, or run out of blocks.
	for i := found.Number + 1; i < found.Number+MaxL2BlocksPerBatchResponse+1; i++ {
		l2Block, err := n.client.BlockByNumber(ctx, big.NewInt(int64(i)))
		if err != nil {
			if errors.Is(err, ethereum.NotFound) { // block number too high
				break
			}
			return nil, fmt.Errorf("failed to retrieve L2 block by number %d: %v", i, err)
		}
		batch, err := l2.BlockToBatch(n.config, l2Block)
		if err != nil {
			return nil, fmt.Errorf("failed to convert L2 block %d (%s) to batch: %v", i, l2Block.Hash(), err)
		}
		if batch == nil { // empty block, nothing to submit as batch
			bundleBuilder.AddCandidate(BundleCandidate{
				ID: eth.BlockID{
					Hash:   l2Block.Hash(),
					Number: l2Block.Number().Uint64(),
				},
				Batch: nil,
			})
			continue
		}

		// Encode the single as a batch to get a size estimate. This should
		// slightly overestimate the size of the final batch, since each
		// serialization will contribute the bundle version byte that is
		// typically amortized over the entire bundle.
		//
		// TODO(conner): use iterative encoder when switching to calldata
		// compression.
		var buf bytes.Buffer
		err = derive.EncodeBatches(n.config, []*derive.BatchData{batch}, &buf)
		if err != nil {
			return nil, fmt.Errorf("failed to encode batch for size estimate: %v", err)
		}

		nextBatchSizeBytes := uint64(len(buf.Bytes()))
		if totalBatchSizeBytes+nextBatchSizeBytes > uint64(req.MaxSize) {
			// Adding this batch causes the bundle to be too large. Record
			// whether the bundle size without the batch fails to meet the
			// minimum size constraint. This is used below to determine whether
			// or not to ignore the minimum size check, since in this scnario it
			// can't be avoided, and the batch submitter must submit the
			// undersized batch to avoid live locking.
			hasLargeNextBatch = totalBatchSizeBytes < uint64(req.MinSize)
			break
		}

		totalBatchSizeBytes += nextBatchSizeBytes
		bundleBuilder.AddCandidate(BundleCandidate{
			ID: eth.BlockID{
				Hash:   l2Block.Hash(),
				Number: l2Block.Number().Uint64(),
			},
			Batch: batch,
		})
	}

	var pruneCount int
	for {
		if !bundleBuilder.HasCandidate() {
			return bundleBuilder.Response(nil), nil
		}

		var buf bytes.Buffer
		err := derive.EncodeBatches(n.config, bundleBuilder.Batches(), &buf)
		if err != nil {
			return nil, fmt.Errorf("failed to encode selected batches as bundle: %v", err)
		}

		bundleSize := uint64(len(buf.Bytes()))

		// Sanity check the bundle size respects the desired maximum. If we have
		// exceeded the max size, prune the last block. This is very unlikely to
		// occur since our initial greedy estimate has a very small, bounded
		// error tolerance, so simply remove the last block and try again.
		if bundleSize > uint64(req.MaxSize) {
			bundleBuilder.PruneLast()
			pruneCount++
			continue
		}

		// There are two specific cases in which we choose to ignore the minimum
		// L1 tx size. These cases are permitted since they arise from
		// situations where the difference between the configured MinTxSize and
		// MaxTxSize is less than the maximum L2 tx size permitted by the
		// mempool.
		//
		// This configuration is useful when trying to ensure the profitability
		// is sufficient, and we permit batches to be submitted with less than
		// our desired configuration only if it is not possible to construct a
		// batch within the given parameters.
		//
		// The two cases are:
		// 1. When the next batch is larger than the difference between the
		//    min and the max, causing the batch to be too small without the
		//    element, and too large with it.
		// 2. When pruning a batch that initially exceeds the max size, and then
		//    becomes too small as a result. This is avoided by only applying
		//    the min size check when the pruneCount is zero.
		ignoreMinSize := pruneCount > 0 || hasLargeNextBatch
		if !ignoreMinSize && bundleSize < uint64(req.MinSize) {
			return nil, nil
		}

		return bundleBuilder.Response(buf.Bytes()), nil
	}
}
