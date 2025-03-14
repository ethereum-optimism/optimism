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

	options eip1559FeeEstimatorOptions
}

type eip1559FeeEstimatorOptions struct {
	// The base multiplier is used to increase the maxFeePerGas (GasFeeCap) by a factor
	baseMultiplier float64

	// The tip multiplier is used to increase the maxPriorityFeePerGas (GasTipCap) by a factor
	tipMultiplier float64
}

type EIP1559FeeEstimatorOption interface {
	apply(*eip1559FeeEstimatorOptions)
}

type eip1559FeeEstimatorOptionBaseMultiplier float64

func (o eip1559FeeEstimatorOptionBaseMultiplier) apply(opts *eip1559FeeEstimatorOptions) {
	opts.baseMultiplier = float64(o)
}

func WithEIP1559BaseMultiplier(multiplier float64) EIP1559FeeEstimatorOption {
	return eip1559FeeEstimatorOptionBaseMultiplier(multiplier)
}

type eip1559FeeEstimatorOptionTipMultiplier float64

func (o eip1559FeeEstimatorOptionTipMultiplier) apply(opts *eip1559FeeEstimatorOptions) {
	opts.tipMultiplier = float64(o)
}

func WithEIP1559TipMultiplier(multiplier float64) EIP1559FeeEstimatorOption {
	return eip1559FeeEstimatorOptionTipMultiplier(multiplier)
}

func NewEIP1559FeeEstimator(client EIP1159FeeEthClient, opts ...EIP1559FeeEstimatorOption) *EIP1559FeeEstimator {
	options := eip1559FeeEstimatorOptions{
		baseMultiplier: 1.0,
		tipMultiplier:  1.0,
	}

	for _, o := range opts {
		o.apply(&options)
	}

	return &EIP1559FeeEstimator{
		client:  client,
		options: options,
	}
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
		newOpts.GasTipCap = multiplyBigInt(tipCap, f.options.tipMultiplier)
	}

	// Add a gas fee cap if needed
	if newOpts.GasFeeCap == nil {
		block, err := f.client.BlockByNumber(ctx, nil)
		if err != nil {
			return nil, err
		}

		baseFee := block.BaseFee()
		if baseFee != nil {
			// The adjusted base fee takes the multiplier into account
			adjustedBaseFee := multiplyBigInt(baseFee, f.options.baseMultiplier)

			// The total fee (maxFeePerGas) is the sum of the base fee and the tip
			newOpts.GasFeeCap = big.NewInt(0).Add(adjustedBaseFee, newOpts.GasTipCap)
		}
	}

	return &newOpts, nil
}

// EIP1159FeeEthClient is a subset of the ethclient.Client interface required for fee estimation
type EIP1159FeeEthClient interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
}

// multiplyBigInt is a little helper for a messy big.Int x float64 multiplication
//
// It rounds the result away from zero since that's the lower risk behavior for fee estimation
func multiplyBigInt(b *big.Int, m float64) *big.Int {
	bFloat := big.NewFloat(0).SetInt(b)
	mFloat := big.NewFloat(m)
	ceiled, accuracy := big.NewFloat(0).Mul(bFloat, mFloat).Int(nil)
	if accuracy == big.Below {
		ceiled = ceiled.Add(ceiled, big.NewInt(1))
	}

	return ceiled
}
