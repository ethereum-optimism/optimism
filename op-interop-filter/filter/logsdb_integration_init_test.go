package filter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegration_Init_FreshStart_SealsAnchor(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber:   99,
		AnchorTime:     1198,
		StartTimestamp: 1200,
		NoSealAnchor:   true,
		NoIngest:       true,
		Blocks:         []seedBlock{{Num: 100, Ts: 1200}}, // head, not ingested
	})

	nextBlock, err := si.initIngestion()
	require.NoError(t, err)
	require.Equal(t, uint64(100), nextBlock)

	latest, ok := si.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(99), latest.Number)
}

func TestIntegration_Init_AnchorOnly_NotInitialized(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		NoIngest:     true,
	})

	require.False(t, si.earliestIngestedBlockSet.Load(),
		"earliestIngestedBlockSet must stay false when only the anchor is sealed")
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

func TestIntegration_Init_Resume_AnchorOnly_DoesNotMarkInitialized(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		NoIngest:     true,
	})

	si2 := reopenSeededIngester(t, si)
	require.False(t, si2.earliestIngestedBlockSet.Load(),
		"resuming with only the anchor must not mark the ingester initialized")
}

func TestIntegration_Init_FirstSealedBlock_OnEmptyDB_NeverReached(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber:   99,
		AnchorTime:     1198,
		StartTimestamp: 1200,
		NoSealAnchor:   true,
		NoIngest:       true,
		Blocks:         []seedBlock{{Num: 100, Ts: 1200}},
	})

	_, hasSealed := si.logsDB.LatestSealedBlock()
	require.False(t, hasSealed)

	// initIngestion takes the fresh-start branch, which seals the parent without
	// consulting FirstSealedBlock.
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

func TestIntegration_Init_SealParentBlockFails_IngesterNotReady(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber:   99,
		AnchorTime:     1198,
		StartTimestamp: 1200,
		NoSealAnchor:   true,
		NoIngest:       true,
		Blocks:         []seedBlock{{Num: 100, Ts: 1200}},
	})

	si.eth.SetInfoByNumberErr(errors.New("rpc not ready"))

	_, err := si.initIngestion()
	require.Error(t, err)
	require.False(t, si.Ready())
	require.Nil(t, si.Error(), "startup failure must not set an IngesterError (silent-startup-failure contract)")
}
