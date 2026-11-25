package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegration_Durability_RestartAfterClean_ResumesFromLatest(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
			{Num: 101, Ts: 1202, Logs: []seedLog{{}}},
			{Num: 102, Ts: 1204, Logs: []seedLog{{}}},
		},
	})

	si2 := reopenSeededIngester(t, si)
	latest, ok := si2.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(102), latest.Number)

	si2.addBlock(103, 1206, si.blockInfo[102].hash, []seedLog{{}})
	require.NoError(t, si2.ingestBlock(103))

	final, ok := si2.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(103), final.Number)
}

func TestIntegration_Durability_AddLogWithoutSealAcrossRestart_DropsPending(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	latest, ok := si.LatestBlock()
	require.True(t, ok)
	require.NoError(t, si.logsDB.AddLog(
		[32]byte{0xab, 0xcd},
		latest,
		0,
		nil,
	))

	si2 := reopenSeededIngester(t, si)

	postRestart, ok := si2.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(100), postRestart.Number,
		"pending logs that were never sealed must not surface after restart")
}

func TestIntegration_Durability_ManyRestarts_LatestAlwaysMatchesIngested(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	for i := uint64(101); i <= 110; i++ {
		ts := 1200 + (i-100)*2
		parent := si.blockInfo[i-1].hash
		si.addBlock(i, ts, parent, []seedLog{{}})
		require.NoError(t, si.ingestBlock(i))

		si = reopenSeededIngester(t, si)
		latest, ok := si.LatestBlock()
		require.True(t, ok)
		require.Equal(t, i, latest.Number, "iteration %d", i)
		require.Equal(t, si.blockInfo[i].hash, latest.Hash, "iteration %d", i)
	}
}
