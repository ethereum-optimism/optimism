package interop

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop/raftwallogdb"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// sealsMockLogsDB is a stateful LogsDB mock exposing only the sealed-block
// views the LogsDB.dfy checker reads; other LogsDB methods panic (nil embed).
type sealsMockLogsDB struct {
	LogsDB
	first     suptypes.BlockSeal
	firstErr  error
	latest    eth.BlockID
	hasLatest bool
	seals     map[uint64]suptypes.BlockSeal
	findErrs  map[uint64]error
}

func (m *sealsMockLogsDB) LatestSealedBlock() (eth.BlockID, bool) {
	return m.latest, m.hasLatest
}

func (m *sealsMockLogsDB) FirstSealedBlock() (suptypes.BlockSeal, error) {
	if m.firstErr != nil {
		return suptypes.BlockSeal{}, m.firstErr
	}
	return m.first, nil
}

func (m *sealsMockLogsDB) FindSealedBlock(number uint64) (suptypes.BlockSeal, error) {
	if err, ok := m.findErrs[number]; ok {
		return suptypes.BlockSeal{}, err
	}
	if seal, ok := m.seals[number]; ok {
		return seal, nil
	}
	return suptypes.BlockSeal{}, suptypes.ErrFuture
}

var _ LogsDB = (*sealsMockLogsDB)(nil)

// dafnySeal builds a BlockSeal at the given number and timestamp with a
// number-derived hash.
func dafnySeal(number, timestamp uint64) suptypes.BlockSeal {
	return suptypes.BlockSeal{
		Hash:      common.Hash{0xd0, byte(number)},
		Number:    number,
		Timestamp: timestamp,
	}
}

// dafnySealedMock builds a mock whose first/latest/find views agree over the
// given seals, which must be in ascending block-number order.
func dafnySealedMock(seals ...suptypes.BlockSeal) *sealsMockLogsDB {
	m := &sealsMockLogsDB{
		firstErr: suptypes.ErrFuture,
		seals:    make(map[uint64]suptypes.BlockSeal, len(seals)),
		findErrs: make(map[uint64]error),
	}
	for _, s := range seals {
		m.seals[s.Number] = s
	}
	if len(seals) > 0 {
		m.first, m.firstErr = seals[0], nil
		m.latest, m.hasLatest = seals[len(seals)-1].ID(), true
	}
	return m
}

func TestCheckLogsDBSealsWellFormedPass(t *testing.T) {
	t.Parallel()

	t.Run("empty db", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckLogsDBSealsWellFormed(dafnySealedMock()))
	})

	t.Run("single sealed block", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckLogsDBSealsWellFormed(dafnySealedMock(dafnySeal(5, 1000))))
	})

	t.Run("contiguous sealed range", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1001), dafnySeal(7, 1003))
		require.NoError(t, CheckLogsDBSealsWellFormed(db))
	})

	t.Run("gap inside sealed range is not excluded by the model", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(7, 1002))
		db.findErrs[6] = suptypes.ErrSkipped
		require.NoError(t, CheckLogsDBSealsWellFormed(db))
	})

	t.Run("real raftwallogdb instance", func(t *testing.T) {
		t.Parallel()
		db, err := raftwallogdb.Open(t.TempDir(), dafnyChainID(1))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, db.Close()) })

		require.NoError(t, CheckLogsDBSealsWellFormed(db))

		prev := eth.BlockID{Hash: common.Hash{0xd0, 5}, Number: 5}
		require.NoError(t, db.SealBlock(common.Hash{}, prev, 1000))
		for n := uint64(6); n <= 8; n++ {
			blk := eth.BlockID{Hash: common.Hash{0xd0, byte(n)}, Number: n}
			require.NoError(t, db.SealBlock(prev.Hash, blk, 1000+(n-5)))
			prev = blk
		}
		require.NoError(t, CheckLogsDBSealsWellFormed(db))

		require.NoError(t, db.Rewind(eth.BlockID{Hash: common.Hash{0xd0, 6}, Number: 6}))
		require.NoError(t, CheckLogsDBSealsWellFormed(db))
	})
}

func TestCheckLogsDBSealsWellFormedViolations(t *testing.T) {
	t.Parallel()

	t.Run("conjunct 0: nil db", func(t *testing.T) {
		t.Parallel()
		err := CheckLogsDBSealsWellFormed(nil)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 0: FirstSealedBlock fails with a non-sentinel error", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000))
		db.firstErr = errors.New("disk corrupted")
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "disk corrupted")
	})

	t.Run("conjunct 0: FindSealedBlock fails with a non-sentinel error", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1001), dafnySeal(7, 1002))
		db.findErrs[6] = errors.New("disk corrupted")
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "FindSealedBlock(6) failed")
	})

	t.Run("conjunct E1: latest present but first absent", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000))
		db.firstErr = suptypes.ErrFuture
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (E1)")
	})

	t.Run("conjunct E1: first present but latest absent", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000))
		db.hasLatest = false
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (E1)")
	})

	t.Run("conjunct B1: first number above latest number", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(7, 1002))
		db.first = db.seals[7]
		db.latest = db.seals[5].ID()
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (B1)")
	})

	t.Run("conjunct F1: FindSealedBlock disagrees with FirstSealedBlock hash", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1001))
		db.first.Hash = common.Hash{0xff}
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (F1)")
	})

	t.Run("conjunct F1: first number not found", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1001))
		db.findErrs[5] = suptypes.ErrSkipped
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (F1)")
		require.ErrorContains(t, err, "found=false")
	})

	t.Run("conjunct L1: FindSealedBlock disagrees with LatestSealedBlock hash", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1001))
		db.latest.Hash = common.Hash{0xff}
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (L1)")
	})

	t.Run("conjunct L1: latest number not found", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1001))
		db.findErrs[6] = suptypes.ErrFuture
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (L1)")
		require.ErrorContains(t, err, "found=false")
	})

	t.Run("conjunct N1: seal number differs from queried number", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1001), dafnySeal(7, 1002))
		db.seals[6] = dafnySeal(60, 1001)
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (N1)")
		require.ErrorContains(t, err, "FindSealedBlock(6) returned seal with number 60")
	})

	t.Run("conjunct T1: equal timestamps", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1000))
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (T1)")
		require.ErrorContains(t, err, "timestamp 1000 at block 6 does not exceed timestamp 1000 at block 5")
	})

	t.Run("conjunct T1: decreasing timestamps across a gap", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(7, 999))
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (T1)")
		require.ErrorContains(t, err, "timestamp 999 at block 7 does not exceed timestamp 1000 at block 5")
	})

	t.Run("joins violations across conjuncts", func(t *testing.T) {
		t.Parallel()
		db := dafnySealedMock(dafnySeal(5, 1000), dafnySeal(6, 1000))
		db.first.Hash = common.Hash{0xff}
		err := CheckLogsDBSealsWellFormed(db)
		require.ErrorContains(t, err, "conjunct (F1)")
		require.ErrorContains(t, err, "conjunct (T1)")
	})
}

func TestCheckFetchReceiptsPost(t *testing.T) {
	t.Parallel()

	blockID := eth.BlockID{Hash: common.Hash{0xd0, 5}, Number: 5}

	t.Run("pass", func(t *testing.T) {
		t.Parallel()
		info := &mockBlockInfo{hash: blockID.Hash, number: blockID.Number}
		require.NoError(t, CheckFetchReceiptsPost(blockID, info))
	})

	t.Run("conjunct 0: nil info", func(t *testing.T) {
		t.Parallel()
		err := CheckFetchReceiptsPost(blockID, nil)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 1: hash mismatch", func(t *testing.T) {
		t.Parallel()
		info := &mockBlockInfo{hash: common.Hash{0xff}, number: blockID.Number}
		err := CheckFetchReceiptsPost(blockID, info)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 1: number mismatch", func(t *testing.T) {
		t.Parallel()
		info := &mockBlockInfo{hash: blockID.Hash, number: blockID.Number + 1}
		err := CheckFetchReceiptsPost(blockID, info)
		require.ErrorContains(t, err, "conjunct (1)")
	})
}

func TestLogsDBAndFetchReceiptsAsserts(t *testing.T) {
	t.Parallel()

	t.Run("pass leaves t untouched", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertLogsDBSealsWellFormed(ft, dafnySealedMock(dafnySeal(5, 1000)))
		blockID := eth.BlockID{Hash: common.Hash{0xd0, 5}, Number: 5}
		AssertFetchReceiptsPost(ft, blockID, &mockBlockInfo{hash: blockID.Hash, number: blockID.Number})
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	t.Run("AssertLogsDBSealsWellFormed reports and fails", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertLogsDBSealsWellFormed(ft, nil)
		require.True(t, ft.errorfCalled)
		require.True(t, ft.failNowCalled)
	})

	t.Run("AssertFetchReceiptsPost reports and fails", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertFetchReceiptsPost(ft, eth.BlockID{Number: 5}, nil)
		require.True(t, ft.errorfCalled)
		require.True(t, ft.failNowCalled)
	})
}
