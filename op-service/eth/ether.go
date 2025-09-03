package eth

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/params"
)

func GweiToWei(gwei float64) (*big.Int, error) {
	if math.IsNaN(gwei) || math.IsInf(gwei, 0) {
		return nil, fmt.Errorf("invalid gwei value: %v", gwei)
	}

	// convert float GWei value into integer Wei value
	wei, _ := new(big.Float).Mul(
		big.NewFloat(gwei),
		big.NewFloat(params.GWei)).
		Int(nil)

	if wei.Cmp(abi.MaxUint256) == 1 {
		return nil, errors.New("gwei value larger than max uint256")
	}

	return wei, nil
}

// RoundWeiToGwei rounds a wei value up to the nearest gwei
func RoundWeiToGwei(wei *big.Int) *big.Int {
	if wei == nil {
		return big.NewInt(0)
	}

	// Get the remainder when dividing by gwei
	remainder := new(big.Int).Mod(wei, big.NewInt(params.GWei))

	// If there's no remainder, return the original value
	if remainder.Cmp(big.NewInt(0)) == 0 {
		return new(big.Int).Set(wei)
	}

	// Otherwise, round up to the next gwei
	return new(big.Int).Add(wei, new(big.Int).Sub(big.NewInt(params.GWei), remainder))
}
