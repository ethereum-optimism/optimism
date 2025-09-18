package txmgr

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/params"
)

type GasPriceEstimatorFn func(ctx context.Context, backend ETHBackend, chainCfg *params.ChainConfig) (*big.Int, *big.Int, *big.Int, error)

func DefaultGasPriceEstimatorFn(ctx context.Context, backend ETHBackend, chainCfg *params.ChainConfig) (*big.Int, *big.Int, *big.Int, error) {
	tip, err := backend.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	head, err := backend.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	if head.BaseFee == nil {
		return nil, nil, nil, errors.New("txmgr does not support pre-london blocks that do not have a base fee")
	}

	var blobFee *big.Int
	if head.ExcessBlobGas != nil {
		blobFee = eth.CalcBlobFeeDefault(head, chainCfg)
	}

	return tip, head.BaseFee, blobFee, nil
}
