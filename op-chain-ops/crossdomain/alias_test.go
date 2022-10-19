package crossdomain_test

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
)

func FuzzAliasing(f *testing.F) {
	f.Fuzz(func(t *testing.T, address []byte) {
		addr := common.BytesToAddress(address)
		aliased := crossdomain.ApplyL1ToL2Alias(addr)
		unaliased := crossdomain.UndoL1ToL2Alias(aliased)
		require.Equal(t, addr, unaliased)
	})
}

func TestAliasing(t *testing.T) {
	cases := []struct {
		Input  common.Address
		Output common.Address
	}{
		{
			Input:  common.HexToAddress("0x24eb0f74a434b2f4f07744652630ce90367aab71"),
			Output: common.HexToAddress("0x35fc0f74a434b2f4f07744652630ce90367abc82"),
		},
		{
			Input:  common.HexToAddress("0xd3f11e293c353bd07ce8fd89e911180e12a7eb77"),
			Output: common.HexToAddress("0xe5021e293c353bd07ce8fd89e911180e12a7fc88"),
		},
		{
			Input:  common.HexToAddress("0xa900b52694dfa5de7255e9b0b6161ec3cc522fed"),
			Output: common.HexToAddress("0xba11b52694dfa5de7255e9b0b6161ec3cc5240fe"),
		},
		{
			Input:  common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff"),
			Output: common.HexToAddress("0x1111000000000000000000000000000000001110"),
		},
		{
			Input:  common.HexToAddress("0x0000000000000000000000000000000000000041"),
			Output: common.HexToAddress("0x1111000000000000000000000000000000001152"),
		},
		{
			Input:  common.HexToAddress("0x4c0aa49c57716406043f97c087a72fe96397959b"),
			Output: common.HexToAddress("0x5d1ba49c57716406043f97c087a72fe96397a6ac"),
		},
	}

	for i, test := range cases {
		t.Run(fmt.Sprintf("test%d", i), func(t *testing.T) {
			aliased := crossdomain.ApplyL1ToL2Alias(test.Input)
			require.Equal(t, test.Output, aliased)
			unaliased := crossdomain.UndoL1ToL2Alias(aliased)
			require.Equal(t, test.Input, unaliased)
		})
	}
}
