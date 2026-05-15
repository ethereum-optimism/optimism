package filter

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestIntegration_BlockHashAt_KnownBlock(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		Blocks: []seedBlock{{Num: 100, Ts: 1200}},
	})

	hash, ok := si.BlockHashAt(100)
	require.True(t, ok)
	require.Equal(t, si.blockInfo[100].hash, hash)
}

func TestIntegration_BlockHashAt_FutureBlock_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		Blocks: []seedBlock{{Num: 100, Ts: 1200}},
	})

	_, ok := si.BlockHashAt(200)
	require.False(t, ok)
}

func TestIntegration_BlockHashAt_BelowEarliest_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks:       []seedBlock{{Num: 100, Ts: 1200}},
	})

	_, ok := si.BlockHashAt(50)
	require.False(t, ok)
}

func TestIntegration_BlockHashAt_BeforeInit_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		NoSealAnchor: true,
		NoIngest:     true,
	})
	require.NoError(t, si.logsDB.Close())
	si.logsDB = nil

	_, ok := si.BlockHashAt(100)
	require.False(t, ok)
}

func TestIntegration_LatestBlock_Empty_ReturnsNotOk(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		NoSealAnchor: true,
		NoIngest:     true,
	})

	_, ok := si.LatestBlock()
	require.False(t, ok)
}

func TestIntegration_LatestBlock_AnchorOnly_ReturnsAnchor(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		NoIngest:     true,
	})

	anchor, ok := si.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(99), anchor.Number)
	require.Equal(t, si.blockInfo[99].hash, anchor.Hash)
}

func TestIntegration_LatestBlock_AfterIngest_AdvancesToHead(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200},
			{Num: 101, Ts: 1202},
		},
	})

	latest, ok := si.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(101), latest.Number)
	require.Equal(t, si.blockInfo[101].hash, latest.Hash)
}

func TestIntegration_LatestTimestamp_TracksLatestSealedBlock(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200},
			{Num: 101, Ts: 1202},
			{Num: 102, Ts: 1204},
		},
	})

	ts, ok := si.LatestTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(1204), ts)

	for n, want := range map[uint64]common.Hash{
		100: si.blockInfo[100].hash,
		101: si.blockInfo[101].hash,
		102: si.blockInfo[102].hash,
	} {
		got, ok := si.BlockHashAt(n)
		require.True(t, ok, "block %d", n)
		require.Equal(t, want, got, "block %d", n)
	}
}
