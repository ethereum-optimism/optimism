package eth

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestL1ChainConfigByChainID(t *testing.T) {
	tc := []struct {
		chainID                        uint64
		expectedDepositContractAddress common.Address
		shouldBeNil                    bool
	}{
		{1, common.HexToAddress("0x00000000219ab540356cbb839cbe05303d7705fa"), false},        // Mainnet
		{11155111, common.HexToAddress("0x7f02c3e3c98b133055b8b348b2ac625669ed295d"), false}, // Sepolia
		{17000, common.HexToAddress("0x4242424242424242424242424242424242424242"), false},    // Holesky
		{560048, common.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa"), false},   // Hoodi
		{560049, common.HexToAddress("0xdeadbeef"), true},                                    // Unknown
	}
	for _, tc := range tc {
		config := L1ChainConfigByChainID(ChainIDFromUInt64(tc.chainID))

		if tc.shouldBeNil {
			require.Nil(t, config)
		} else {
			require.Equal(t, tc.expectedDepositContractAddress, config.DepositContractAddress)
		}
	}
}
