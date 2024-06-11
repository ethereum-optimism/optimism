package snapshots

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

func TestLoadABIs(t *testing.T) {
	tests := []struct {
		contract string
		method   func() *abi.ABI
	}{
		{"DisputeGameFactory", LoadDisputeGameFactoryABI},
		{"FaultDisputeGame", LoadFaultDisputeGameABI},
		{"PreimageOracle", LoadPreimageOracleABI},
		{"MIPS", LoadMIPSABI},
		{"DelayedWETH", LoadDelayedWETHABI},
	}
	for _, test := range tests {
		test := test
		t.Run(test.contract, func(t *testing.T) {
			actual := test.method()
			require.NotNil(t, actual)
		})
	}
}
