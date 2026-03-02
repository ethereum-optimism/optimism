package derive

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestMakeEmptyBatch_SameEpoch(t *testing.T) {
	cfg := testRollupConfig()

	// Cursor at L1 origin 5, next L1 block at num 6 has time 12.
	// Next L2 timestamp = 100 + 2 = 102, which is < 12... wait, our test
	// L1 blocks have Time = num*2, so L1#6 has time 12.
	// With cursor.Timestamp = 100, nextTimestamp = 102 which is >= 12,
	// so it would advance. Let's set up so it stays.
	cursor := l2Cursor{
		Number:    10,
		Timestamp: 4, // nextTimestamp = 6
		L1Origin:  testL1Ref(5).ID(),
	}

	// L1 block 6 has time 12. nextTimestamp = 6 < 12 → stays at same epoch.
	findL1 := func(num uint64) *L1Input {
		if num <= 6 {
			return makeTestL1Input(num)
		}
		return nil
	}

	batch, epochL1, newOrigin := makeEmptyBatch(cursor, findL1, cfg)
	require.NotNil(t, batch)
	require.NotNil(t, epochL1)
	require.Equal(t, uint64(5), newOrigin.Number)
	require.Equal(t, uint64(6), batch.Timestamp)
	require.Equal(t, cursor.L1Origin.Number, uint64(batch.EpochNum))
}

func TestMakeEmptyBatch_AdvancesEpoch(t *testing.T) {
	cfg := testRollupConfig()

	// Cursor at L1 origin 5. L1 block 5 has time 10, L1 block 6 has time 12.
	// If cursor.Timestamp = 100, nextTimestamp = 102 >= 12 → advances epoch.
	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  testL1Ref(5).ID(),
	}

	findL1 := func(num uint64) *L1Input {
		if num <= 6 {
			return makeTestL1Input(num)
		}
		return nil
	}

	batch, epochL1, newOrigin := makeEmptyBatch(cursor, findL1, cfg)
	require.NotNil(t, batch)
	require.NotNil(t, epochL1)
	require.Equal(t, uint64(6), newOrigin.Number)
	require.Equal(t, uint64(102), batch.Timestamp)
	require.Equal(t, uint64(6), uint64(batch.EpochNum))
}

func TestMakeEmptyBatch_MissingL1(t *testing.T) {
	cfg := testRollupConfig()

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  eth.BlockID{Number: 99},
	}

	findL1 := func(num uint64) *L1Input {
		return nil
	}

	batch, epochL1, _ := makeEmptyBatch(cursor, findL1, cfg)
	require.Nil(t, batch)
	require.Nil(t, epochL1)
}
