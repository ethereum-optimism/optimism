package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l2"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"

	"github.com/ethereum/go-ethereum/common"
)

type DriverAPI interface {
	l2.EngineAPI
	l2.EthBackend
}

// step takes an L1 block id and then creates and executes the L2 block. The forkchoice is updated as well.
func (d *Driver) step(ctx context.Context, l1Input eth.BlockID, l2Parent eth.BlockID, l2Finalized common.Hash) (eth.BlockID, error) {
	logger := d.log.New("input_l1", l1Input, "input_l2_parent", l2Parent, "finalized_l2", l2Finalized)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	bl, receipts, err := d.dl.Fetch(fetchCtx, l1Input)
	if err != nil {
		return eth.BlockID{}, fmt.Errorf("failed to fetch block with receipts: %v", err)
	}
	logger.Debug("fetched L1 data for driver")

	attrs, err := derive.PayloadAttributes(bl, receipts)
	if err != nil {
		return eth.BlockID{}, fmt.Errorf("failed to derive execution payload inputs: %v", err)
	}
	logger.Debug("derived L2 block inputs")

	payload, err := derive.ExecutionPayload(ctx, d.rpc, l2Parent.Hash, l2Finalized, attrs)
	if err != nil {
		return eth.BlockID{}, fmt.Errorf("failed to derive execution payload: %v", err)
	}

	logger = logger.New("derived_l2", payload.ID())
	logger.Info("derived full block")

	err = l2.ExecutePayload(ctx, d.rpc, payload)
	if err != nil {
		return eth.BlockID{}, fmt.Errorf("failed to apply execution payload: %v", err)
	}
	logger.Info("executed block")

	err = l2.ForkchoiceUpdate(ctx, d.rpc, payload.BlockHash, l2Finalized)
	if err != nil {
		return eth.BlockID{}, fmt.Errorf("failed to persist execution payload: %v", err)
	}
	logger.Info("updated fork-choice with block")

	return payload.ID(), nil
}
