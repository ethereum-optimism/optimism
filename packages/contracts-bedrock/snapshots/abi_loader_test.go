package snapshots

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

func TestLoadABIs(t *testing.T) {
	tests := []struct {
		contract string
		method   func() (*abi.ABI, error)
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
			actual, err := test.method()
			require.NoError(t, err)
			require.NotNil(t, actual)
		})
	}
}
