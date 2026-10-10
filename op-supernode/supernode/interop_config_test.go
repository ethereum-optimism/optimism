package supernode

import (
	"testing"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/stretchr/testify/require"
)

func TestResolveInteropActivationTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		vnCfgs  map[eth.ChainID]*opnodecfg.Config
		want    *uint64
		wantErr string
	}{
		{
			name: "derive from consistent rollup configs",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollup.Config{LagoonTime: ptr.New(uint64(1234))}},
				eth.ChainIDFromUInt64(8453): {Rollup: rollup.Config{LagoonTime: ptr.New(uint64(1234))}},
			},
			want: ptr.New(uint64(1234)),
		},
		{
			// op-deployer writes Lagoon-at-genesis as a zero fork time.
			name: "Lagoon at genesis resolves to the genesis timestamp",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollupAt(1000, ptr.New(uint64(0)))},
				eth.ChainIDFromUInt64(8453): {Rollup: rollupAt(1000, ptr.New(uint64(0)))},
			},
			want: ptr.New(uint64(1000)),
		},
		{
			name: "Lagoon before genesis resolves to the earliest genesis timestamp",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollupAt(1000, ptr.New(uint64(500)))},
				eth.ChainIDFromUInt64(8453): {Rollup: rollupAt(1200, ptr.New(uint64(500)))},
			},
			want: ptr.New(uint64(1000)),
		},
		{
			name: "Lagoon after genesis is kept",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollupAt(1000, ptr.New(uint64(1006)))},
				eth.ChainIDFromUInt64(8453): {Rollup: rollupAt(1000, ptr.New(uint64(1006)))},
			},
			want: ptr.New(uint64(1006)),
		},
		{
			// A chain that launches after Lagoon still shares the superchain-wide activation time.
			name: "Lagoon between genesis timestamps is kept",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollupAt(1000, ptr.New(uint64(1100)))},
				eth.ChainIDFromUInt64(8453): {Rollup: rollupAt(1200, ptr.New(uint64(1100)))},
			},
			want: ptr.New(uint64(1100)),
		},
		{
			name: "Lagoon exactly at genesis is kept",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollupAt(1000, ptr.New(uint64(1000)))},
				eth.ChainIDFromUInt64(8453): {Rollup: rollupAt(1000, ptr.New(uint64(1000)))},
			},
			want: ptr.New(uint64(1000)),
		},
		{
			// Both configs activate Lagoon at the shared genesis, encoded two ways.
			name: "zero and absolute Lagoon-at-genesis encodings agree",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollupAt(1000, ptr.New(uint64(0)))},
				eth.ChainIDFromUInt64(8453): {Rollup: rollupAt(1000, ptr.New(uint64(1000)))},
			},
			want: ptr.New(uint64(1000)),
		},
		{
			name: "missing virtual node configs are skipped",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   nil,
				eth.ChainIDFromUInt64(8453): {Rollup: rollupAt(1000, ptr.New(uint64(0)))},
			},
			want: ptr.New(uint64(1000)),
		},
		{
			name: "leave interop disabled when no rollup config enables it",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollup.Config{}},
				eth.ChainIDFromUInt64(8453): {Rollup: rollup.Config{}},
			},
		},
		{
			name: "error on mixed nil and configured rollup timestamps",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollup.Config{}},
				eth.ChainIDFromUInt64(8453): {Rollup: rollup.Config{LagoonTime: ptr.New(uint64(1234))}},
			},
			wantErr: "has no Lagoon activation timestamp",
		},
		{
			name: "error on mismatched rollup timestamps",
			vnCfgs: map[eth.ChainID]*opnodecfg.Config{
				eth.ChainIDFromUInt64(10):   {Rollup: rollup.Config{LagoonTime: ptr.New(uint64(100))}},
				eth.ChainIDFromUInt64(8453): {Rollup: rollup.Config{LagoonTime: ptr.New(uint64(200))}},
			},
			wantErr: "mismatched Lagoon activation timestamps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveInteropActivationTimestamp(tt.vnCfgs)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func rollupAt(genesisTime uint64, lagoonTime *uint64) rollup.Config {
	return rollup.Config{Genesis: rollup.Genesis{L2Time: genesisTime}, LagoonTime: lagoonTime}
}
