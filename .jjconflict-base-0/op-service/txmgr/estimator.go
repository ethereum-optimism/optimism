package txmgr

import (
	"context"
	"errors"
	"math/big"
)

type GasPriceEstimatorFn func(ctx context.Context, backend ETHBackend) (*big.Int, *big.Int, *big.Int, *big.Int, error)

func DefaultGasPriceEstimatorFn(ctx context.Context, backend ETHBackend) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	tip, err := backend.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	head, err := backend.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if head.BaseFee == nil {
		return nil, nil, nil, nil, errors.New("txmgr does not support pre-london blocks that do not have a base fee")
	}

	blobBaseFee, err := backend.BlobBaseFee(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	blobTipFee := big.NewInt(0) // using zero value for the default gas price estimator (if bgpo is not available)
	return tip, head.BaseFee, blobTipFee, blobBaseFee, nil
}
