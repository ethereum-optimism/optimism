package interopgen

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestInteropAtGenesis(t *testing.T) {
	zero := hexutil.Uint64(0)
	nonzero := hexutil.Uint64(24)

	tests := []struct {
		name   string
		offset *hexutil.Uint64
		want   bool
	}{
		{"nil offset: Interop not scheduled", nil, false},
		{"zero offset: Interop active at genesis", &zero, true},
		{"non-zero offset: Interop delayed activation", &nonzero, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, interopAtGenesis(tt.offset))
		})
	}
}

// devFeatureBitmapForL2Genesis sets the OptimismPortalInteropFlag when interop is enabled and the L2CMFlag when L2CM
// is enabled.
func TestDevFeatureBitmapForL2Genesis(t *testing.T) {
	interopOnly := devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OptimismPortalInteropFlag)
	l2cmOnly := devfeatures.EnableDevFeature(common.Hash{}, devfeatures.L2CMFlag)
	both := devfeatures.EnableDevFeature(interopOnly, devfeatures.L2CMFlag)

	tests := []struct {
		name          string
		enableInterop bool
		useL2CM       bool
		want          common.Hash
	}{
		{"both disabled", false, false, common.Hash{}},
		{"interop only", true, false, interopOnly},
		{"L2CM only", false, true, l2cmOnly},
		{"both enabled", true, true, both},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, devFeatureBitmapForL2Genesis(tt.enableInterop, tt.useL2CM))
		})
	}
}

func TestEncodeSuperPermissionedGameArgs(t *testing.T) {
	proposer := common.HexToAddress("0x1111111111111111111111111111111111111111")

	args, err := encodeSuperPermissionedGameArgs(proposer)
	require.NoError(t, err)
	require.Len(t, args, 32)

	addressType, err := abi.NewType("address", "", nil)
	require.NoError(t, err)

	unpacked, err := abi.Arguments{{Type: addressType}}.Unpack(args)
	require.NoError(t, err)
	require.Equal(t, proposer, unpacked[0])
}
