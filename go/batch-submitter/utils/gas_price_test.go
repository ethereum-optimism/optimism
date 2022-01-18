package utils_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/go/batch-submitter/utils"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestGasPriceFromGwei asserts that the integer value is scaled properly by
// 10^9.
func TestGasPriceFromGwei(t *testing.T) {
	require.Equal(t, utils.GasPriceFromGwei(0), new(big.Int))
	require.Equal(t, utils.GasPriceFromGwei(1), big.NewInt(params.GWei))
	require.Equal(t, utils.GasPriceFromGwei(100), big.NewInt(100*params.GWei))
}
