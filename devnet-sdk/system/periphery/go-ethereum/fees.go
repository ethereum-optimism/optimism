package goethereum

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	// Ensure that the feeEstimator implements the FeeEstimator interface
	_ FeeEstimator = (*eip1559eeEstimator)(nil)
)

// FeeEstimator is a generic fee estimation interface (not specific to EIP-1559)
type FeeEstimator interface {
	EstimateFees(ctx context.Context, opts *bind.TransactOpts) (*bind.TransactOpts, error)
}

// eip1559eeEstimator is a fee estimator that uses EIP-1559 fee estimation
type eip1559eeEstimator struct {
	// Access to the Ethereum client is needed to get the fee information from the chain
	client FeeEthClient

	// The tip multiplier is used to increase the maxPriorityFeePerGas (GasTipCap) by a factor
	tipMultiplier *big.Int
}

func NewEIP1559FeeEstimator(client FeeEthClient) *eip1559eeEstimator {
	return &eip1559eeEstimator{
		client:        client,
		tipMultiplier: big.NewInt(1),
	}
}

func (f *eip1559eeEstimator) WithTipMultiplier(multiplier *big.Int) *eip1559eeEstimator {
	newF := *f
	newF.tipMultiplier = multiplier

	return &newF
}

func (f *eip1559eeEstimator) EstimateFees(ctx context.Context, opts *bind.TransactOpts) (*bind.TransactOpts, error) {
	newOpts := *opts

	// Add a gas tip cap if needed
	if newOpts.GasTipCap == nil {
		tipCap, err := f.client.SuggestGasTipCap(ctx)

		if err != nil {
			return nil, err
		}

		// GasTipCap represents the maxPriorityFeePerGas
		newOpts.GasTipCap = big.NewInt(0).Mul(tipCap, f.tipMultiplier)
	}

	// Add a gas fee cap if needed
	if newOpts.GasFeeCap == nil {
		block, err := f.client.BlockByNumber(ctx, nil)

		if err != nil {
			return nil, err
		}

		baseFee := block.BaseFee()
		if baseFee != nil {
			// The total fee (maxFeePerGas) is the sum of the base fee and the tip
			newOpts.GasFeeCap = big.NewInt(0).Add(block.BaseFee(), newOpts.GasTipCap)
		}
	}

	return &newOpts, nil
}

// FeeEthClient is a subset of the ethclient.Client interface required for fee estimation
type FeeEthClient interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
}
