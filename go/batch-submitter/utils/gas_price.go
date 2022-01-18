package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// GasPriceFromGwei converts an uint64 gas price in gwei to a big.Int in wei.
func GasPriceFromGwei(gasPriceInGwei uint64) *big.Int {
	return new(big.Int).SetUint64(gasPriceInGwei * params.GWei)
}
