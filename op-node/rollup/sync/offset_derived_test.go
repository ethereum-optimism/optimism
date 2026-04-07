// Tests: InteropELSyncBootstrap_Feature.md §2 — offset_derived block-step math and L2AncestorByN.
package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
)

func TestDerivedBlockSteps(t *testing.T) {
	tests := []struct {
		name   string
		offset time.Duration
		bt     uint64
		want   uint64
	}{
		{"zero offset", 0, 2, 0},
		{"negative treated as zero", -time.Hour, 2, 0},
		{"zero block time", time.Hour, 0, 0},
		{"floor BT=2 offset=3s -> 1", 3 * time.Second, 2, 1},
		{"floor BT=2 offset=4s -> 2", 4 * time.Second, 2, 2},
		{"sub-second offset truncates", 500 * time.Millisecond, 1, 0},
		{"12h with 2s blocks", 12 * time.Hour, 2, 21600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DerivedBlockSteps(tt.offset, tt.bt)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestL2AncestorByN_walkAndClamp(t *testing.T) {
	ctx := context.Background()
	genesis := eth.BlockID{Hash: common.Hash{'g'}, Number: 0}
	b0 := eth.L2BlockRef{Hash: genesis.Hash, Number: 0, ParentHash: common.Hash{}}
	b1 := eth.L2BlockRef{Hash: common.Hash{'1'}, Number: 1, ParentHash: b0.Hash}
	b2 := eth.L2BlockRef{Hash: common.Hash{'2'}, Number: 2, ParentHash: b1.Hash}
	b3 := eth.L2BlockRef{Hash: common.Hash{'3'}, Number: 3, ParentHash: b2.Hash}

	m := &testutils.MockL2Client{}
	m.ExpectL2BlockRefByHash(b3.ParentHash, b2, nil)
	m.ExpectL2BlockRefByHash(b2.ParentHash, b1, nil)

	out, err := L2AncestorByN(ctx, m, genesis, b3, 2)
	require.NoError(t, err)
	require.Equal(t, b1, out)

	// n larger than depth — clamp at genesis
	m2 := &testutils.MockL2Client{}
	m2.ExpectL2BlockRefByHash(b1.ParentHash, b0, nil)
	out2, err := L2AncestorByN(ctx, m2, genesis, b1, 100)
	require.NoError(t, err)
	require.Equal(t, b0, out2)
}
