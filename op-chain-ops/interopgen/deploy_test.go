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

func TestEncodePermissionedGameArgs(t *testing.T) {
	prestate := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	proposer := common.HexToAddress("0x1111111111111111111111111111111111111111")
	challenger := common.HexToAddress("0x2222222222222222222222222222222222222222")

	args, err := encodePermissionedGameArgs(prestate, proposer, challenger)
	require.NoError(t, err)
	require.Len(t, args, 96)

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	addressType, err := abi.NewType("address", "", nil)
	require.NoError(t, err)

	unpacked, err := abi.Arguments{{Type: bytes32Type}, {Type: addressType}, {Type: addressType}}.Unpack(args)
	require.NoError(t, err)
	require.Equal(t, prestate, common.Hash(unpacked[0].([32]byte)))
	require.Equal(t, proposer, unpacked[1])
	require.Equal(t, challenger, unpacked[2])
}
