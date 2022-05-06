package drivers

import (
	"errors"
	"math/big"
	"strings"
)

var (
	errMaxPriorityFeePerGasNotFound = errors.New(
		"Method eth_maxPriorityFeePerGas not found",
	)

	// FallbackGasTipCap is the default fallback gasTipCap used when we are
	// unable to query an L1 backend for a suggested gasTipCap.
	FallbackGasTipCap = big.NewInt(1500000000)
)

// IsMaxPriorityFeePerGasNotFoundError returns true if the provided error
// signals that the backend does not support the eth_maxPrirorityFeePerGas
// method. In this case, the caller should fallback to using the constant above.
func IsMaxPriorityFeePerGasNotFoundError(err error) bool {
	return strings.Contains(
		err.Error(), errMaxPriorityFeePerGasNotFound.Error(),
	)
}
