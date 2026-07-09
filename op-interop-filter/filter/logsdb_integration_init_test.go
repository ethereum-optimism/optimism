package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegration_Init_FreshStart_StartsWithEmptyDB(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber:   99,
		AnchorTime:     1198,
		StartTimestamp: 1200,
		NoIngest:       true,
		Blocks:         []seedBlock{{Num: 100, Ts: 1200}}, // head, not ingested
	})

	nextBlock, err := si.initIngestion()
	require.NoError(t, err)
	require.Equal(t, uint64(100), nextBlock)

	_, ok := si.LatestBlock()
	require.False(t, ok, "fresh start must not seal any block before the first ingestion")
}

func TestIntegration_Init_EmptyDB_NotInitialized(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		NoIngest:     true,
	})

	require.False(t, si.earliestIngestedBlockSet.Load(),
		"earliestIngestedBlockSet must stay false until the first block is ingested")
}

func TestIntegration_Init_Resume_FromExistingDB_FindsEarliest(t *testing.T) {
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
	require.True(t, si2.earliestIngestedBlockSet.Load())
	require.Equal(t, uint64(100), si2.earliestIngestedBlock.Load())

	latest, ok := si2.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(102), latest.Number)
}

func TestIntegration_Init_FirstSealedBlock_OnEmptyDB_NeverReached(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber:   99,
		AnchorTime:     1198,
		StartTimestamp: 1200,
		NoIngest:       true,
		Blocks:         []seedBlock{{Num: 100, Ts: 1200}},
	})

	_, hasSealed := si.logsDB.LatestSealedBlock()
	require.False(t, hasSealed)

	// initIngestion takes the fresh-start branch without consulting FirstSealedBlock.
	_, err := si.initIngestion()
	require.NoError(t, err)
}

func TestIntegration_Init_OpenBlockOnFirstBlock_Succeeds(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}, {}}},
			{Num: 101, Ts: 1202, Logs: []seedLog{{}}},
		},
	})

	si2 := reopenSeededIngester(t, si)
	ref, logCount, _, err := si2.logsDB.OpenBlock(100)
	require.NoError(t, err)
	require.Equal(t, uint64(100), ref.Number)
	require.Equal(t, uint64(1200), ref.Time)
	require.Equal(t, uint32(2), logCount)
}

func TestIntegration_Init_Close_FlushesAndStops(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	require.NoError(t, si.logsDB.Close())
	si.logsDB = nil

	si2 := reopenSeededIngester(t, si)
	latest, ok := si2.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(100), latest.Number)
}
