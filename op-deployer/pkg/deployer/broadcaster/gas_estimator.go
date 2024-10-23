package broadcaster

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	// baseFeePadFactor = 50% as a divisor
	baseFeePadFactor = big.NewInt(2)
	// tipMulFactor = 20 as a multiplier
	tipMulFactor = big.NewInt(20)
	// dummyBlobFee is a dummy value for the blob fee. Since this gas estimator will never
	// post blobs, it's just set to 1.
	dummyBlobFee = big.NewInt(1)
)

// DeployerGasPriceEstimator is a custom gas price estimator for use with op-deployer.
// It pads the base fee by 50% and multiplies the suggested tip by 20.
func DeployerGasPriceEstimator(ctx context.Context, client txmgr.ETHBackend) (*big.Int, *big.Int, *big.Int, error) {
	chainHead, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get block: %w", err)
	}

	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get gas tip cap: %w", err)
	}

	baseFeePad := new(big.Int).Div(chainHead.BaseFee, baseFeePadFactor)
	paddedBaseFee := new(big.Int).Add(chainHead.BaseFee, baseFeePad)
	paddedTip := new(big.Int).Mul(tip, tipMulFactor)
	return paddedTip, paddedBaseFee, dummyBlobFee, nil
}
