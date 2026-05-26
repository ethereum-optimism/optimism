package filter

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// remoteExecMsg constructs a seedLog declaring an executing message that
// targets the given coordinates on a remote chain.
func remoteExecMsg(remote eth.ChainID, remoteBlock, remoteTs uint64, remoteLogIdx uint32) seedLog {
	return seedLog{
		ExecMsg: &seedExecMsg{
			TargetChainID: remote,
			TargetBlock:   remoteBlock,
			TargetTs:      remoteTs,
			TargetLogIdx:  remoteLogIdx,
			Origin:        common.Address{0xaa, 0xbb},
			PayloadHash:   common.Hash{0xee, byte(remoteBlock), byte(remoteLogIdx)},
		},
	}
}

func TestIntegration_GetExecMsgsAtTimestamp_HappyPath(t *testing.T) {
	t.Parallel()

	remote := eth.ChainIDFromUInt64(999)
	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{
				remoteExecMsg(remote, 500, 1198, 0),
				remoteExecMsg(remote, 500, 1198, 1),
			}},
		},
	})

	msgs, err := si.GetExecMsgsAtTimestamp(1200)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	for _, m := range msgs {
		require.Equal(t, remote, m.ExecutingMessage.ChainID)
		require.Equal(t, uint64(500), m.ExecutingMessage.BlockNum)
		require.Equal(t, uint64(1198), m.ExecutingMessage.Timestamp)
		require.Equal(t, uint64(100), m.InclusionBlockNum)
		require.Equal(t, uint64(1200), m.InclusionTimestamp)
	}
}

func TestIntegration_GetExecMsgsAtTimestamp_NoMessagesAtTimestamp(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	msgs, err := si.GetExecMsgsAtTimestamp(1200)
	require.NoError(t, err)
	require.Empty(t, msgs)
}

func TestIntegration_GetExecMsgsAtTimestamp_BeforeInit_Uninitialized(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		NoSealAnchor: true,
		NoIngest:     true,
	})
	require.NoError(t, si.logsDB.Close())
	si.logsDB = nil

	_, err := si.GetExecMsgsAtTimestamp(1200)
	require.ErrorIs(t, err, interop.ErrUninitialized)
}

func TestIntegration_GetExecMsgsAtTimestamp_AnchorOnly_Uninitialized(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		NoIngest:     true,
	})

	_, err := si.GetExecMsgsAtTimestamp(1198)
	require.ErrorIs(t, err, interop.ErrUninitialized)
}

func TestIntegration_GetExecMsgsAtTimestamp_BelowEarliest_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	// Force earliestIngestedBlock past block 50 — already true (it's 100) — so
	// a query for ts=1100 (-> block 50) short-circuits to (nil, nil) before
	// hitting the DB.
	msgs, err := si.GetExecMsgsAtTimestamp(1100)
	require.NoError(t, err)
	require.Empty(t, msgs)
}

func TestIntegration_GetExecMsgsAtTimestamp_FirstSealedBlock_ReturnsExecMsgs(t *testing.T) {
	t.Parallel()

	remote := eth.ChainIDFromUInt64(999)
	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{
				remoteExecMsg(remote, 500, 1198, 0),
			}},
			{Num: 101, Ts: 1202},
		},
	})

	si2 := reopenSeededIngester(t, si)

	msgs, err := si2.GetExecMsgsAtTimestamp(1200)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, remote, msgs[0].ExecutingMessage.ChainID)
}

func TestIntegration_GetExecMsgsAtTimestamp_NonexistentTimestamp_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
			{Num: 101, Ts: 1202, Logs: []seedLog{{}}},
		},
	})

	// 1201 has no block — TargetBlockNumber should resolve to a block whose
	// stored ts is 1200, so the ts mismatch short-circuits to empty.
	msgs, err := si.GetExecMsgsAtTimestamp(1201)
	require.NoError(t, err)
	require.Empty(t, msgs)

	// Future timestamp beyond latest also returns empty (no error).
	msgs, err = si.GetExecMsgsAtTimestamp(9999)
	require.False(t, errors.Is(err, interop.ErrUninitialized))
	require.NoError(t, err)
	require.Empty(t, msgs)
}
