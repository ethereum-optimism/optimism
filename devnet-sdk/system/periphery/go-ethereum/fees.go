package goethereum

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// Ensure that the feeEstimator implements the FeeEstimator interface
	_ FeeEstimator = (*EIP1559FeeEstimator)(nil)

	// Ensure that the EIP1159FeeEthClient implements the EIP1159FeeEthClient interface
	_ EIP1159FeeEthClient = (*ethclient.Client)(nil)
)

// FeeEstimator is a generic fee estimation interface (not specific to EIP-1559)
type FeeEstimator interface {
	EstimateFees(ctx context.Context, opts *bind.TransactOpts) (*bind.TransactOpts, error)
}

// EIP1559FeeEstimator is a fee estimator that uses EIP-1559 fee estimation
type EIP1559FeeEstimator struct {
	// Access to the Ethereum client is needed to get the fee information from the chain
	client EIP1159FeeEthClient

	// The tip multiplier is used to increase the maxPriorityFeePerGas (GasTipCap) by a factor
	tipMultiplier *big.Int
}

func NewEIP1559FeeEstimator(client EIP1159FeeEthClient) *EIP1559FeeEstimator {
	return &EIP1559FeeEstimator{
		client:        client,
		tipMultiplier: big.NewInt(1),
	}
}

func (f *EIP1559FeeEstimator) WithTipMultiplier(multiplier *big.Int) *EIP1559FeeEstimator {
	newF := *f
	newF.tipMultiplier = multiplier

	return &newF
}

func (f *EIP1559FeeEstimator) EstimateFees(ctx context.Context, opts *bind.TransactOpts) (*bind.TransactOpts, error) {
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

// EIP1159FeeEthClient is a subset of the ethclient.Client interface required for fee estimation
type EIP1159FeeEthClient interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
}
