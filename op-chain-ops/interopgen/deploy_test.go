package interopgen

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
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

// devFeatureBitmapForL2Genesis enables the L2CMFlag by default and sets the OptimismPortalInteropFlag when
// interop is enabled. Setting disableL2CM=true clears the L2CM bit.
func TestDevFeatureBitmapForL2Genesis(t *testing.T) {
	interopOnly := devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OptimismPortalInteropFlag)
	l2cmOnly := devfeatures.EnableDevFeature(common.Hash{}, devfeatures.L2CMFlag)
	both := devfeatures.EnableDevFeature(l2cmOnly, devfeatures.OptimismPortalInteropFlag)

	tests := []struct {
		name          string
		enableInterop bool
		disableL2CM   bool
		want          common.Hash
	}{
		{"defaults (no interop, L2CM on)", false, false, l2cmOnly},
		{"interop on, L2CM on", true, false, both},
		{"interop on, L2CM disabled", true, true, interopOnly},
		{"interop off, L2CM disabled", false, true, common.Hash{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, devFeatureBitmapForL2Genesis(tt.enableInterop, tt.disableL2CM))
		})
	}
}
