package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l2"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Downloader interface {
	// FetchL1Info fetches the L1 header information corresponding to a L1 block ID
	FetchL1Info(ctx context.Context, id eth.BlockID) (derive.L1Info, error)
	// FetchReceipts of a L1 block
	FetchReceipts(ctx context.Context, id eth.BlockID) ([]*types.Receipt, error)
	// FetchBatches from the given window of L1 blocks
	FetchBatches(ctx context.Context, window []eth.BlockID) ([]derive.BatchData, error)
	// FetchL2Info fetches the L2 header information corresponding to a L2 block ID
	FetchL2Info(ctx context.Context, id eth.BlockID) (derive.L2Info, error)
}

type DriverAPI interface {
	l2.EngineAPI
	l2.EthBackend
}

type outputImpl struct {
	dl     Downloader
	rpc    DriverAPI
	log    log.Logger
	Config rollup.Config
}

// DriverStep derives and processes one or more L2 blocks from the given sequencing window of L1 blocks.
// An incomplete sequencing window will result in an incomplete L2 chain if so.
//
// After the step completes it returns the block ID of the last processed L2 block, even if an error occurs.
func (d *outputImpl) step(ctx context.Context, l2Head eth.BlockID, l2Finalized eth.BlockID, l1Input []eth.BlockID) (out eth.BlockID, err error) {
	if len(l1Input) == 0 {
		return l2Head, fmt.Errorf("empty L1 sequencing window on L2 %s", l2Head)
	}

	logger := d.log.New("input_l1_first", l1Input[0], "input_l1_last", l1Input[len(l1Input)-1],
		"input_l2_parent", l2Head, "finalized_l2", l2Finalized)
	logger.Trace("Running update step on the L2 node")

	epoch := rollup.Epoch(l1Input[0].Number)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	l2Info, err := d.dl.FetchL2Info(fetchCtx, l2Head)
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch L2 block info of %s: %v", l2Head, err)
	}
	logger.Trace("Got l2 info")
	l1Info, err := d.dl.FetchL1Info(fetchCtx, l1Input[0])
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch L1 block info of %s: %v", l1Input[0], err)
	}
	logger.Trace("Got l1 info")
	receipts, err := d.dl.FetchReceipts(fetchCtx, l1Input[0])
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch receipts of %s: %v", l1Input[0], err)
	}
	logger.Trace("Got receipts")
	// TODO: with sharding the blobs may be identified in more detail than L1 block hashes
	batches, err := d.dl.FetchBatches(fetchCtx, l1Input)
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch batches from %s: %v", l1Input, err)
	}
	logger.Trace("Got batches")

	attrsList, err := derive.PayloadAttributes(&d.Config, l1Info, receipts, batches, l2Info)
	if err != nil {
		return l2Head, fmt.Errorf("failed to derive execution payload inputs: %v", err)
	}
	logger.Debug("derived L2 block inputs")

	last := l2Head
	for i, attrs := range attrsList {
		last, err = AddBlock(ctx, logger, d.rpc, last, l2Finalized.Hash, attrs)
		if err != nil {
			return last, fmt.Errorf("failed to extend L2 chain at block %d/%d of epoch %d: %v", i, len(attrsList), epoch, err)
		}
	}

	return last, nil
}

// AddBlock extends the L2 chain by deriving the full execution payload from inputs,
// and then executing and persisting it.
//
// After the step completes it returns the block ID of the last processed L2 block, even if an error occurs.
func AddBlock(ctx context.Context, logger log.Logger, rpc DriverAPI,
	l2Parent eth.BlockID, l2Finalized common.Hash, attrs *l2.PayloadAttributes) (eth.BlockID, error) {

	payload, err := derive.ExecutionPayload(ctx, rpc, l2Parent.Hash, l2Finalized, attrs)
	if err != nil {
		return l2Parent, fmt.Errorf("failed to derive execution payload: %v", err)
	}

	logger = logger.New("derived_l2", payload.ID())
	logger.Info("derived full block", "l2Parent", l2Parent, "attrs", attrs, "payload", payload)

	err = l2.ExecutePayload(ctx, rpc, payload)
	if err != nil {
		return l2Parent, fmt.Errorf("failed to apply execution payload: %v", err)
	}
	logger.Info("executed block")

	err = l2.ForkchoiceUpdate(ctx, rpc, payload.BlockHash, l2Finalized)
	if err != nil {
		return payload.ID(), fmt.Errorf("failed to persist execution payload: %v", err)
	}
	logger.Info("updated fork-choice with block")
	return payload.ID(), nil
}
