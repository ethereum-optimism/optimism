package bigs

import (
	"math/big"
)

func Equal(a *big.Int, b *big.Int) bool {
	return a.Cmp(b) == 0
}

func IsZero(val *big.Int) bool {
	return val.Sign() == 0
}

func IsPositive(val *big.Int) bool {
	return val.Sign() > 0
}

func IsNegative(val *big.Int) bool {
	return val.Sign() < 0
}

// Uint64Strict converts a big.Int to a uint64, panicking if the value is not a UInt64.
func Uint64Strict(val *big.Int) uint64 {
	if !val.IsUint64() {
		panic("bigs.Uint64Strict: value does not fit in uint64")
	}
	//nolint:bigint // This is the one place we want to allow it.
	return val.Uint64()
}
