package opcm

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestL2GenesisRefusesChainIDOver64Bits checks that an L2 chain ID that forge cannot take fails
// with an error before any script runs.
func TestL2GenesisRefusesChainIDOver64Bits(t *testing.T) {
	var input L2GenesisInput
	input.L2ChainID = new(big.Int).Lsh(big.NewInt(1), 64)
	_, err := L2Genesis(context.Background(), nil, common.Address{0x5e}, false, input)
	require.ErrorContains(t, err, "does not fit in 64 bits")
}
