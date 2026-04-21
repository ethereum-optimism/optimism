package interopgen

import (
	"testing"

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

// devFeatureBitmapForL2Genesis sets the OptimismPortalInteropFlag when interop is enabled.
// Verify the bitmap differs between enabled and disabled states.
func TestDevFeatureBitmapForL2Genesis(t *testing.T) {
	enabled := devFeatureBitmapForL2Genesis(true)
	disabled := devFeatureBitmapForL2Genesis(false)
	require.NotEqual(t, enabled, disabled, "bitmap should differ when interop is enabled vs disabled")
	require.True(t, disabled == (common.Hash{}), "disabled bitmap should be zero")
}
