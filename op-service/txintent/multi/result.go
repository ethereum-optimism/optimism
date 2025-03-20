package txintent

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/core/types"
)

var _ txintent.Result = (*MulticallOutput)(nil)

type MulticallOutput struct {
	receipt    *types.Receipt
	includedIn eth.BlockRef
	chainID    eth.ChainID
}

func (m *MulticallOutput) Init() txintent.Result {
	return &MulticallOutput{}
}

func (m *MulticallOutput) FromReceipt(ctx context.Context, rec *types.Receipt, includedIn eth.BlockRef, chainID eth.ChainID) error {
	m.receipt = rec
	m.includedIn = includedIn
	m.chainID = chainID
	return nil
}
