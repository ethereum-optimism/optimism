package db

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestHeadsForChain tests the heads for a chain,
// confirming the Unsafe, Safe and Finalized all return the correct head for the chain.
// and confirming that the chainID matters when finding the value
func TestHeadsForChain(t *testing.T) {
	h := heads.NewHeads()
	chainHeads := heads.ChainHeads{
		Unsafe:         entrydb.EntryIdx(1),
		CrossUnsafe:    entrydb.EntryIdx(2),
		LocalSafe:      entrydb.EntryIdx(3),
		CrossSafe:      entrydb.EntryIdx(4),
		LocalFinalized: entrydb.EntryIdx(5),
		CrossFinalized: entrydb.EntryIdx(6),
	}
	h.Put(types.ChainIDFromUInt64(1), chainHeads)
	chainsDB := NewChainsDB(nil, &stubHeadStorage{h}, testlog.Logger(t, log.LevelDebug))
	tcases := []struct {
		name          string
		chainID       types.ChainID
		checkerType   types.SafetyLevel
		expectedLocal entrydb.EntryIdx
		expectedCross entrydb.EntryIdx
	}{
		{
			"Unsafe Head",
			types.ChainIDFromUInt64(1),
			Unsafe,
			entrydb.EntryIdx(1),
			entrydb.EntryIdx(2),
		},
		{
			"Safe Head",
			types.ChainIDFromUInt64(1),
			Safe,
			entrydb.EntryIdx(3),
			entrydb.EntryIdx(4),
		},
		{
			"Finalized Head",
			types.ChainIDFromUInt64(1),
			Finalized,
			entrydb.EntryIdx(5),
			entrydb.EntryIdx(6),
		},
		{
			"Incorrect Chain",
			types.ChainIDFromUInt64(100),
			Safe,
			entrydb.EntryIdx(0),
			entrydb.EntryIdx(0),
		},
	}

	for _, c := range tcases {
		t.Run(c.name, func(t *testing.T) {
			checker := NewSafetyChecker(c.checkerType, chainsDB)
			localHead := checker.LocalHeadForChain(c.chainID)
			crossHead := checker.CrossHeadForChain(c.chainID)
			require.Equal(t, c.expectedLocal, localHead)
			require.Equal(t, c.expectedCross, crossHead)
		})
	}
}

func TestCheck(t *testing.T) {
	h := heads.NewHeads()
	chainHeads := heads.ChainHeads{
		Unsafe:         entrydb.EntryIdx(6),
		CrossUnsafe:    entrydb.EntryIdx(5),
		LocalSafe:      entrydb.EntryIdx(4),
		CrossSafe:      entrydb.EntryIdx(3),
		LocalFinalized: entrydb.EntryIdx(2),
		CrossFinalized: entrydb.EntryIdx(1),
	}
	h.Put(types.ChainIDFromUInt64(1), chainHeads)

	// the logStore contains just a single stubbed log DB
	logDB := &stubLogDB{}
	logsStore := map[types.ChainID]LogStorage{
		types.ChainIDFromUInt64(1): logDB,
	}

	chainsDB := NewChainsDB(logsStore, &stubHeadStorage{h}, testlog.Logger(t, log.LevelDebug))

	tcases := []struct {
		name             string
		checkerType      types.SafetyLevel
		chainID          types.ChainID
		blockNum         uint64
		logIdx           uint32
		loghash          common.Hash
		containsResponse containsResponse
		expected         bool
	}{
		{
			// confirm that checking Unsafe uses the unsafe head,
			// and that we can find logs even *at* the unsafe head index
			"Unsafe Log at Head",
			Unsafe,
			types.ChainIDFromUInt64(1),
			1,
			1,
			common.Hash{1, 2, 3},
			containsResponse{entrydb.EntryIdx(6), nil},
			true,
		},
		{
			// confirm that checking the Safe head works
			"Safe Log",
			Safe,
			types.ChainIDFromUInt64(1),
			1,
			1,
			common.Hash{1, 2, 3},
			containsResponse{entrydb.EntryIdx(3), nil},
			true,
		},
		{
			// confirm that checking the Finalized head works
			"Finalized Log",
			Finalized,
			types.ChainIDFromUInt64(1),
			1,
			1,
			common.Hash{1, 2, 3},
			containsResponse{entrydb.EntryIdx(1), nil},
			true,
		},
		{
			// confirm that when exists is false, we return false
			"Does not Exist",
			Safe,
			types.ChainIDFromUInt64(1),
			1,
			1,
			common.Hash{1, 2, 3},
			containsResponse{entrydb.EntryIdx(1), logs.ErrConflict},
			false,
		},
		{
			// confirm that when a head is out of range, we return false
			"Unsafe Out of Range",
			Unsafe,
			types.ChainIDFromUInt64(1),
			1,
			1,
			common.Hash{1, 2, 3},
			containsResponse{entrydb.EntryIdx(100), nil},
			false,
		},
		{
			// confirm that when a head is out of range, we return false
			"Safe Out of Range",
			Safe,
			types.ChainIDFromUInt64(1),
			1,
			1,
			common.Hash{1, 2, 3},
			containsResponse{entrydb.EntryIdx(5), nil},
			false,
		},
		{
			// confirm that when a head is out of range, we return false
			"Finalized Out of Range",
			Finalized,
			types.ChainIDFromUInt64(1),
			1,
			1,
			common.Hash{1, 2, 3},
			containsResponse{entrydb.EntryIdx(3), nil},
			false,
		},
		{
			// confirm that when Contains returns an error, we return false
			"Error",
			Safe,
			types.ChainIDFromUInt64(1),
			1,
			1,
			common.Hash{1, 2, 3},
			containsResponse{entrydb.EntryIdx(0), errors.New("error")},
			false,
		},
	}

	for _, c := range tcases {
		t.Run(c.name, func(t *testing.T) {
			// rig the logStore to return the expected response
			logDB.containsResponse = c.containsResponse
			checker := NewSafetyChecker(c.checkerType, chainsDB)
			r := checker.Check(c.chainID, c.blockNum, c.logIdx, c.loghash)
			// confirm that the expected outcome is correct
			require.Equal(t, c.expected, r)
		})
	}
}
