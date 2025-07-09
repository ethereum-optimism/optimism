package fromda

import (
	"fmt"
	"io/fs"
	"math/rand" // nosemgrep
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type stubMetrics struct {
	DBDerivedEntryCount int64
}

func (s *stubMetrics) RecordDBDerivedEntryCount(count int64) {
	s.DBDerivedEntryCount = count
}

var _ Metrics = (*stubMetrics)(nil)

type setupFn func(t *testing.T, db *DB, m *stubMetrics)

type assertFn func(t *testing.T, db *DB, m *stubMetrics)

func runDBTest(t *testing.T, setup setupFn, assert assertFn) {
	createDb := func(t *testing.T, dir string) (*DB, *stubMetrics, string) {
		logger := testlog.Logger(t, log.LvlTrace)
		path := filepath.Join(dir, "test.db")
		m := &stubMetrics{}
		db, err := NewFromFile(logger, m, path)
		require.NoError(t, err, "Failed to create database")
		t.Cleanup(func() {
			err := db.Close()
			if err != nil {
				require.ErrorIs(t, err, fs.ErrClosed)
			}
		})
		return db, m, path
	}

	t.Run("New", func(t *testing.T) {
		db, m, _ := createDb(t, t.TempDir())
		setup(t, db, m)
		assert(t, db, m)
	})

	t.Run("Existing", func(t *testing.T) {
		dir := t.TempDir()
		db, m, path := createDb(t, dir)
		setup(t, db, m)
		// Close and recreate the database
		require.NoError(t, db.Close())
		checkDBInvariants(t, path, m)

		db2, m, path := createDb(t, dir)
		assert(t, db2, m)
		checkDBInvariants(t, path, m)
	})
}

func TestEmptyDB(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {},
		func(t *testing.T, db *DB, m *stubMetrics) {
			_, err := db.Last()
			require.ErrorIs(t, err, types.ErrFuture)

			_, err = db.First()
			require.ErrorIs(t, err, types.ErrFuture)

			_, err = db.SourceToLastDerived(eth.BlockID{})
			require.ErrorIs(t, err, types.ErrFuture)

			_, err = db.DerivedToFirstSource(eth.BlockID{}, types.RevisionAny)
			require.ErrorIs(t, err, types.ErrFuture)

			_, err = db.PreviousDerived(eth.BlockID{}, types.RevisionAny)
			require.ErrorIs(t, err, types.ErrFuture)

			_, err = db.NextDerived(eth.BlockID{}, types.RevisionAny)
			require.ErrorIs(t, err, types.ErrFuture)

			_, err = db.PreviousSource(eth.BlockID{})
			require.ErrorIs(t, err, types.ErrFuture)

			_, err = db.NextSource(eth.BlockID{})
			require.ErrorIs(t, err, types.ErrFuture)

			_, err = db.Next(types.DerivedIDPair{
				Source:  eth.BlockID{},
				Derived: eth.BlockID{}})
			require.ErrorIs(t, err, types.ErrFuture)
		})
}

func mockL1(i uint64) types.BlockSeal {
	return types.BlockSeal{
		Hash:      crypto.Keccak256Hash([]byte(fmt.Sprintf("L1 block %d", i))),
		Number:    i,
		Timestamp: 1000_000 + i*12,
	}
}

func mockL1Ref(i uint64) eth.BlockRef {
	v := mockL1(i)
	if i == 0 {
		v.MustWithParent(eth.BlockID{})
	}
	return v.MustWithParent(mockL1(i - 1).ID())
}

func mockL2(i uint64) types.BlockSeal {
	var h common.Hash
	if i != 0 {
		h = crypto.Keccak256Hash([]byte(fmt.Sprintf("L2 block %d", i)))
	}
	return types.BlockSeal{
		Hash:      h,
		Number:    i,
		Timestamp: 1000_000 + i*2,
	}
}

func mockL2Ref(i uint64) eth.BlockRef {
	v := mockL2(i)
	if i == 0 {
		v.MustWithParent(eth.BlockID{})
	}
	return v.MustWithParent(mockL2(i - 1).ID())
}

func toRef(seal types.BlockSeal, parentHash common.Hash) eth.BlockRef {
	return eth.BlockRef{
		Hash:       seal.Hash,
		Number:     seal.Number,
		ParentHash: parentHash,
		Time:       seal.Timestamp,
	}
}

func dbAddDerivedAny(db *DB, source eth.BlockRef, derived eth.BlockRef) error {
	return db.AddDerived(source, derived, types.RevisionAny)
}

func TestSingleEntryDB(t *testing.T) {
	expectedSource := mockL1(0)
	expectedDerived := mockL2(2)
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, dbAddDerivedAny(db, toRef(expectedSource, mockL1(0).Hash), toRef(expectedDerived, mockL2(0).Hash)))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			// First
			pair, err := db.First()
			require.NoError(t, err)
			require.Equal(t, expectedSource, pair.Source)
			require.Equal(t, expectedDerived, pair.Derived)

			// Last
			pair, err = db.Last()
			require.NoError(t, err)
			require.Equal(t, expectedSource, pair.Source)
			require.Equal(t, expectedDerived, pair.Derived)

			// Next after Last
			_, err = db.Next(types.DerivedIDPair{
				Source:  pair.Source.ID(),
				Derived: pair.Derived.ID()})
			require.ErrorIs(t, err, types.ErrFuture)

			// Last Derived
			derived, err := db.SourceToLastDerived(expectedSource.ID())
			require.NoError(t, err)
			require.Equal(t, expectedDerived, derived)

			// Last Derived with a non-existent Source
			_, err = db.SourceToLastDerived(eth.BlockID{Hash: common.Hash{0xaa}, Number: expectedSource.Number})
			require.ErrorIs(t, err, types.ErrConflict)

			// Next with a non-existent block (derived and source)
			_, err = db.Next(types.DerivedIDPair{
				Source:  eth.BlockID{Hash: common.Hash{0xaa}, Number: expectedSource.Number},
				Derived: expectedDerived.ID()})
			require.ErrorIs(t, err, types.ErrConflict)
			_, err = db.Next(types.DerivedIDPair{
				Source:  expectedSource.ID(),
				Derived: eth.BlockID{Hash: common.Hash{0xaa}, Number: expectedDerived.Number}})
			require.ErrorIs(t, err, types.ErrConflict)

			// First Source
			source, err := db.DerivedToFirstSource(expectedDerived.ID(), types.RevisionAny)
			require.NoError(t, err)
			require.Equal(t, expectedSource, source)

			// Source with a non-existent Derived
			_, err = db.DerivedToFirstSource(eth.BlockID{Hash: common.Hash{0xbb}, Number: expectedDerived.Number}, types.RevisionAny)
			require.ErrorIs(t, err, types.ErrConflict)

			// PreviousDerived
			prev, err := db.PreviousDerived(expectedDerived.ID(), types.RevisionAny)
			require.NoError(t, err)
			require.Equal(t, types.BlockSeal{}, prev, "zeroed seal before first entry")

			// PreviousSource
			prev, err = db.PreviousSource(expectedSource.ID())
			require.NoError(t, err)
			require.Equal(t, types.BlockSeal{}, prev, "zeroed seal before first entry")

			// NextDerived
			_, err = db.NextDerived(expectedDerived.ID(), types.RevisionAny)
			require.ErrorIs(t, err, types.ErrFuture)

			// NextSource
			_, err = db.NextSource(expectedSource.ID())
			require.ErrorIs(t, err, types.ErrFuture)

			// Next
			_, err = db.Next(types.DerivedIDPair{
				Source:  expectedSource.ID(),
				Derived: expectedDerived.ID()})
			require.ErrorIs(t, err, types.ErrFuture)
		})
}

func TestGap(t *testing.T) {
	// mockL1 starts at block 1 to produce a gap
	expectedSource := mockL1(1)
	// mockL2 starts at block 2 to produce a gap
	expectedDerived := mockL2(2)
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, dbAddDerivedAny(db, toRef(expectedSource, mockL1(0).Hash), toRef(expectedDerived, mockL2(0).Hash)))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			_, err := db.NextDerived(mockL2(0).ID(), types.RevisionAny)
			require.ErrorIs(t, err, types.ErrSkipped)

			_, err = db.NextSource(mockL1(0).ID())
			require.ErrorIs(t, err, types.ErrSkipped)
		})
}

func TestThreeBlocksDB(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block0, common.Hash{}), toRef(l2Block0, common.Hash{})))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block1, l1Block0.Hash), toRef(l2Block0, common.Hash{}))) // bump scope
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block1, l1Block0.Hash), toRef(l2Block1, l2Block0.Hash)))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block2, l1Block1.Hash), toRef(l2Block1, l2Block0.Hash))) // bump scope
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block2, l1Block1.Hash), toRef(l2Block2, l2Block1.Hash)))
	}, func(t *testing.T, db *DB, m *stubMetrics) {

		pair, err := db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block2, pair.Source)
		require.Equal(t, l2Block2, pair.Derived)

		pair, err = db.First()
		require.NoError(t, err)
		require.Equal(t, l1Block0, pair.Source)
		require.Equal(t, l2Block0, pair.Derived)

		derived, err := db.SourceToLastDerived(l1Block2.ID())
		require.NoError(t, err)
		require.Equal(t, l2Block2, derived)

		_, err = db.SourceToLastDerived(eth.BlockID{Hash: common.Hash{0xaa}, Number: l1Block2.Number})
		require.ErrorIs(t, err, types.ErrConflict)

		source, err := db.DerivedToFirstSource(l2Block2.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block2, source)

		_, err = db.DerivedToFirstSource(eth.BlockID{Hash: common.Hash{0xbb}, Number: l2Block2.Number}, types.RevisionAny)
		require.ErrorIs(t, err, types.ErrConflict)

		derived, err = db.SourceToLastDerived(l1Block1.ID())
		require.NoError(t, err)
		require.Equal(t, l2Block1, derived)

		source, err = db.DerivedToFirstSource(l2Block1.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block1, source)

		derived, err = db.SourceToLastDerived(l1Block0.ID())
		require.NoError(t, err)
		require.Equal(t, l2Block0, derived)

		source, err = db.DerivedToFirstSource(l2Block0.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block0, source)

		derived, err = db.PreviousDerived(l2Block0.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, types.BlockSeal{}, derived)

		derived, err = db.PreviousDerived(l2Block1.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block0, derived)

		derived, err = db.PreviousDerived(l2Block2.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block1, derived)

		next, err := db.NextDerived(l2Block0.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block1, next.Derived)
		require.Equal(t, l1Block1, next.Source)

		next, err = db.NextDerived(l2Block1.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block2, next.Derived)
		require.Equal(t, l1Block2, next.Source)

		_, err = db.NextDerived(l2Block2.ID(), types.RevisionAny)
		require.ErrorIs(t, err, types.ErrFuture)

		source, err = db.PreviousSource(l1Block0.ID())
		require.NoError(t, err)
		require.Equal(t, types.BlockSeal{}, source)

		source, err = db.PreviousSource(l1Block1.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block0, source)

		source, err = db.PreviousSource(l1Block2.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block1, source)

		source, err = db.NextSource(l1Block0.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block1, source)

		source, err = db.NextSource(l1Block1.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block2, source)

		_, err = db.NextSource(l1Block2.ID())
		require.ErrorIs(t, err, types.ErrFuture)

		_, err = db.Next(types.DerivedIDPair{
			Source:  l1Block2.ID(),
			Derived: l2Block2.ID()})
		require.ErrorIs(t, err, types.ErrFuture)

		next, err = db.Next(types.DerivedIDPair{
			Source:  l1Block0.ID(),
			Derived: l2Block0.ID()})
		require.NoError(t, err)
		require.Equal(t, l1Block1, next.Source) // bump in scope first
		require.Equal(t, l2Block0, next.Derived)

		next, err = db.Next(types.DerivedIDPair{
			Source:  l1Block1.ID(),
			Derived: l2Block0.ID()})
		require.NoError(t, err)
		require.Equal(t, l1Block1, next.Source)
		require.Equal(t, l2Block1, next.Derived)

		next, err = db.Next(types.DerivedIDPair{
			Source:  l1Block1.ID(),
			Derived: l2Block1.ID()})
		require.NoError(t, err)
		require.Equal(t, l1Block2, next.Source) // bump in scope first
		require.Equal(t, l2Block1, next.Derived)

		next, err = db.Next(types.DerivedIDPair{
			Source:  l1Block2.ID(),
			Derived: l2Block1.ID()})
		require.NoError(t, err)
		require.Equal(t, l1Block2, next.Source)
		require.Equal(t, l2Block2, next.Derived)
	})
}

// TestFastL2Batcher tests what happens if we have derived multiple L2 blocks from the same L1 block
func TestFastL2Batcher(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)
	l2Block3 := mockL2(3)
	l2Block4 := mockL2(4)
	l2Block5 := mockL2(5)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		// L2 genesis derived from L1 genesis
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block0, common.Hash{}), toRef(l2Block0, common.Hash{})))
		// Many L2 blocks all derived from same L1 block
		l1Ref1 := toRef(l1Block1, l1Block0.Hash)
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, toRef(l2Block0, common.Hash{}))) // bump scope
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, toRef(l2Block1, l2Block0.Hash)))
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, toRef(l2Block2, l2Block1.Hash)))
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, toRef(l2Block3, l2Block2.Hash)))
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, toRef(l2Block4, l2Block3.Hash)))
		// Last L2 block derived from later L1 block
		l1Ref2 := toRef(l1Block2, l1Block1.Hash)
		require.NoError(t, dbAddDerivedAny(db, l1Ref2, toRef(l2Block4, l2Block3.Hash))) // bump scope
		require.NoError(t, dbAddDerivedAny(db, l1Ref2, toRef(l2Block5, l2Block4.Hash)))
	}, func(t *testing.T, db *DB, m *stubMetrics) {

		pair, err := db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block2, pair.Source)
		require.Equal(t, l2Block5, pair.Derived)

		derived, err := db.SourceToLastDerived(l1Block2.ID())
		require.NoError(t, err)
		require.Equal(t, l2Block5, derived)

		// test what tip was derived from
		source, err := db.DerivedToFirstSource(l2Block5.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block2, source)

		// Multiple L2 blocks all derived from same older L1 block
		for _, b := range []types.BlockSeal{l2Block1, l2Block2, l2Block3, l2Block4} {
			source, err = db.DerivedToFirstSource(b.ID(), types.RevisionAny)
			require.NoError(t, err)
			require.Equal(t, l1Block1, source)
		}

		// test that the Last L2 counts, not the intermediate
		derived, err = db.SourceToLastDerived(l1Block1.ID())
		require.NoError(t, err)
		require.Equal(t, l2Block4, derived)

		derived, err = db.PreviousDerived(l2Block5.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block4, derived)
		derived, err = db.PreviousDerived(l2Block4.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block3, derived)
		derived, err = db.PreviousDerived(l2Block3.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block2, derived)
		derived, err = db.PreviousDerived(l2Block2.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block1, derived)
		derived, err = db.PreviousDerived(l2Block1.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block0, derived)

		next, err := db.NextDerived(l2Block0.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block1, next.Source)
		require.Equal(t, l2Block1, next.Derived)
		next, err = db.NextDerived(l2Block1.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block1, next.Source)
		require.Equal(t, l2Block2, next.Derived)
		next, err = db.NextDerived(l2Block2.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block1, next.Source)
		require.Equal(t, l2Block3, next.Derived)
		next, err = db.NextDerived(l2Block3.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block1, next.Source)
		require.Equal(t, l2Block4, next.Derived)
		next, err = db.NextDerived(l2Block4.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block2, next.Source) // derived from later L1 block
		require.Equal(t, l2Block5, next.Derived)
		_, err = db.NextDerived(l2Block5.ID(), types.RevisionAny)
		require.ErrorIs(t, err, types.ErrFuture)

		source, err = db.PreviousSource(l1Block2.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block1, source)
		source, err = db.PreviousSource(l1Block1.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block0, source)

		source, err = db.NextSource(l1Block0.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block1, source)
		source, err = db.NextSource(l1Block1.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block2, source)
		_, err = db.NextSource(l1Block2.ID())
		require.ErrorIs(t, err, types.ErrFuture)

		next, err = db.Next(types.DerivedIDPair{
			Source:  l1Block1.ID(),
			Derived: l2Block2.ID()})
		require.NoError(t, err)
		require.Equal(t, l1Block1, next.Source) // no increment in L1 yet, the next after is L2 block 3
		require.Equal(t, l2Block3, next.Derived)
	})
}

// TestSlowL2Batcher tests what happens if we have derived no new L2 blocks in a L1 block
func TestSlowL2Batcher(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)
	l1Block3 := mockL1(3)
	l1Block4 := mockL1(4)
	l1Block5 := mockL1(5)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		// L2 genesis derived from L1 genesis
		l2Ref0 := toRef(l2Block0, common.Hash{})
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block0, common.Hash{}), l2Ref0))
		// Many L1 blocks all repeating the same L2 block
		l2Ref1 := toRef(l2Block1, l2Block0.Hash)
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block1, l1Block0.Hash), l2Ref0)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block1, l1Block0.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block2, l1Block1.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block3, l1Block2.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block4, l1Block3.Hash), l2Ref1))
		// New L1 block that finally produces a new L2 block
		l1Ref5 := toRef(l1Block5, l1Block4.Hash)
		require.NoError(t, dbAddDerivedAny(db, l1Ref5, l2Ref1)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, l1Ref5, toRef(l2Block2, l2Block1.Hash)))
	}, func(t *testing.T, db *DB, m *stubMetrics) {

		pair, err := db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block5, pair.Source)
		require.Equal(t, l2Block2, pair.Derived)

		// test what we last derived at the tip
		derived, err := db.SourceToLastDerived(l1Block5.ID())
		require.NoError(t, err)
		require.Equal(t, l2Block2, derived)

		// Multiple L1 blocks all copying the last known derived L2 block
		for _, b := range []types.BlockSeal{l1Block1, l1Block2, l1Block3, l1Block4} {
			derived, err = db.SourceToLastDerived(b.ID())
			require.NoError(t, err)
			require.Equal(t, l2Block1, derived)
		}

		// test that the first L1 counts, not the ones that repeat the L2 info
		source, err := db.DerivedToFirstSource(l2Block1.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block1, source)

		derived, err = db.PreviousDerived(l2Block2.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block1, derived)
		derived, err = db.PreviousDerived(l2Block1.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Block0, derived)

		next, err := db.NextDerived(l2Block0.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block1, next.Source)
		require.Equal(t, l2Block1, next.Derived)
		next, err = db.NextDerived(l2Block1.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l1Block5, next.Source)
		require.Equal(t, l2Block2, next.Derived)
		_, err = db.NextDerived(l2Block2.ID(), types.RevisionAny)
		require.ErrorIs(t, err, types.ErrFuture)

		source, err = db.PreviousSource(l1Block5.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block4, source)
		source, err = db.PreviousSource(l1Block4.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block3, source)
		source, err = db.PreviousSource(l1Block3.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block2, source)
		source, err = db.PreviousSource(l1Block2.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block1, source)
		source, err = db.PreviousSource(l1Block1.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block0, source)

		source, err = db.NextSource(l1Block0.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block1, source)
		source, err = db.NextSource(l1Block1.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block2, source)
		source, err = db.NextSource(l1Block2.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block3, source)
		source, err = db.NextSource(l1Block4.ID())
		require.NoError(t, err)
		require.Equal(t, l1Block5, source)
		_, err = db.NextSource(l1Block5.ID())
		require.ErrorIs(t, err, types.ErrFuture)

		next, err = db.Next(types.DerivedIDPair{
			Source:  l1Block2.ID(),
			Derived: l2Block1.ID()})
		require.NoError(t, err)
		require.Equal(t, l1Block3, next.Source)
		require.Equal(t, l2Block1, next.Derived) // no increment in L2 yet, the next after is L1 block 3
	})
}

func TestManyEntryDB(t *testing.T) {
	t.Run("at genesis", func(t *testing.T) {
		testManyEntryDB(t, 0, 0)
	})
	t.Run("start at arbitrary L1 block", func(t *testing.T) {
		testManyEntryDB(t, 30000, 0)
	})
	t.Run("start at arbitrary L2 block", func(t *testing.T) {
		testManyEntryDB(t, 0, 30000)
	})
	t.Run("start at arbitrary L1 and L2 block", func(t *testing.T) {
		testManyEntryDB(t, 2000000, 1000000)
	})
}

func testManyEntryDB(t *testing.T, offsetL1 uint64, offsetL2 uint64) {
	// L2 -> first L1 occurrence
	firstSource := make(map[eth.BlockID]types.BlockSeal)
	// L1 -> last L2 occurrence
	lastDerived := make(map[eth.BlockID]types.BlockSeal)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		// Insert genesis
		require.NoError(t, dbAddDerivedAny(db, toRef(mockL1(offsetL1), common.Hash{}), toRef(mockL2(offsetL2), common.Hash{})))
		firstSource[mockL2(offsetL2).ID()] = mockL1(offsetL1)
		lastDerived[mockL1(offsetL1).ID()] = mockL2(offsetL2)

		rng := rand.New(rand.NewSource(1234))
		// Insert 1000 randomly generated entries, derived at random bumps in L1
		for i := uint64(0); i < 1000; i++ {
			pair, err := db.Last()
			require.NoError(t, err)

			switch rng.Intn(2) {
			case 0: // bump L1
				pair.Source = mockL1(pair.Source.Number + 1)
			case 1: // bump L2
				pair.Derived = mockL2(pair.Derived.Number + 1)
			}
			sourceRef := toRef(pair.Source, mockL1(pair.Source.Number-1).Hash)
			derivedRef := toRef(pair.Derived, mockL2(pair.Derived.Number-1).Hash)
			lastDerived[sourceRef.ID()] = pair.Derived
			if _, ok := firstSource[derivedRef.ID()]; !ok {
				firstSource[derivedRef.ID()] = pair.Source
			}
			require.NoError(t, dbAddDerivedAny(db, sourceRef, derivedRef))
		}
	}, func(t *testing.T, db *DB, m *stubMetrics) {
		// Now assert we can find what they are all derived from, and match the expectations.
		pair, err := db.Last()
		require.NoError(t, err)
		require.NotZero(t, pair.Source.Number-offsetL1)
		require.NotZero(t, pair.Derived.Number-offsetL2)

		for i := offsetL1; i <= pair.Source.Number; i++ {
			l1ID := mockL1(i).ID()
			derived, err := db.SourceToLastDerived(l1ID)
			require.NoError(t, err)
			require.Contains(t, lastDerived, l1ID)
			require.Equal(t, lastDerived[l1ID], derived)
		}

		for i := offsetL2; i <= pair.Derived.Number; i++ {
			l2ID := mockL2(i).ID()
			source, err := db.DerivedToFirstSource(l2ID, types.RevisionAny)
			require.NoError(t, err)
			require.Contains(t, firstSource, l2ID)
			require.Equal(t, firstSource[l2ID], source)
		}

		// if not started at genesis, try to read older data, assert it's unavailable.
		if offsetL1 > 0 {
			_, err := db.SourceToLastDerived(mockL1(0).ID())
			require.ErrorIs(t, err, types.ErrSkipped)

			_, err = db.SourceToLastDerived(mockL1(offsetL1 - 1).ID())
			require.ErrorIs(t, err, types.ErrSkipped)
		}
		if offsetL2 > 0 {
			_, err := db.DerivedToFirstSource(mockL2(0).ID(), types.RevisionAny)
			require.ErrorIs(t, err, types.ErrSkipped)

			_, err = db.DerivedToFirstSource(mockL2(offsetL2-1).ID(), types.RevisionAny)
			require.ErrorIs(t, err, types.ErrSkipped)
		}
	})
}

// TestRewindToSourceEmptied tests what happens if we rewind based on derived-from source,
// and rewind to before the first block, emptying the DB.
func TestRewindToSourceEmptied(t *testing.T) {
	l1Block2 := mockL1(2)
	l1Block3 := mockL1(3)
	l1Block4 := mockL1(4)
	l1Block5 := mockL1(5)

	l2Block6 := mockL2(6)
	l2Block7 := mockL2(7)
	l2Block8 := mockL2(8)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		l2Ref7 := toRef(l2Block7, l2Block6.Hash)
		l2Ref8 := toRef(l2Block8, l2Block7.Hash)
		// Start from L1 block 3, and L2 block 7
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block3, l1Block2.Hash), l2Ref7))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block4, l1Block3.Hash), l2Ref7))
		// New L1 block that finally produces a new L2 block
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block5, l1Block4.Hash), l2Ref7)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block5, l1Block4.Hash), l2Ref8))
	}, func(t *testing.T, db *DB, m *stubMetrics) {

		pair, err := db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block5, pair.Source)
		require.Equal(t, l2Block8, pair.Derived)

		inv := &reads.TestInvalidator{}
		// Rewind to the block before DB started
		require.NoError(t, db.RewindToSource(inv, l1Block2.ID()))
		require.True(t, inv.Invalidated)
		require.Equal(t, uint64(0), inv.InvalidatedSourceNum)
		require.Equal(t, uint64(0), inv.InvalidatedDerivedTimestamp)
		pair, err = db.Last()
		require.ErrorIs(t, err, types.ErrFuture)
		require.True(t, db.IsEmpty())

		// Test that we can add a new starting point
		altL1Ref := mockL1Ref(10000)
		altL2Ref := mockL2Ref(20000)
		require.NoError(t, dbAddDerivedAny(db, altL1Ref, altL2Ref))
		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, altL1Ref.ID(), pair.Source.ID())
		require.Equal(t, altL2Ref.ID(), pair.Derived.ID())
	})
}

// TestRewindToSource tests what happens if we rewind based on derived-from source.
func TestRewindToSource(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)
	l1Block3 := mockL1(3)
	l1Block4 := mockL1(4)
	l1Block5 := mockL1(5)
	l1Block6 := mockL1(6)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		// L2 genesis derived from L1 genesis
		l2Ref0 := toRef(l2Block0, common.Hash{})
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block0, common.Hash{}), l2Ref0))
		// Many L1 blocks all repeating the same L2 block
		l2Ref1 := toRef(l2Block1, l2Block0.Hash)
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block1, l1Block0.Hash), l2Ref0)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block1, l1Block0.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block2, l1Block1.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block3, l1Block2.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block4, l1Block3.Hash), l2Ref1))
		// New L1 block that finally produces a new L2 block
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block5, l1Block4.Hash), l2Ref1)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block5, l1Block4.Hash), toRef(l2Block2, l2Block1.Hash)))
	}, func(t *testing.T, db *DB, m *stubMetrics) {

		pair, err := db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block5, pair.Source)
		require.Equal(t, l2Block2, pair.Derived)

		inv := &reads.TestInvalidator{}
		// Rewind to the future
		require.ErrorIs(t, db.RewindToSource(inv, l1Block6.ID()), types.ErrFuture)
		require.False(t, inv.Invalidated)

		inv = &reads.TestInvalidator{}
		// Rewind to the exact block we're at
		require.NoError(t, db.RewindToSource(inv, l1Block5.ID()))
		require.True(t, inv.Invalidated)
		require.Equal(t, l1Block5.Number, inv.InvalidatedSourceNum)
		require.Equal(t, l2Block2.Timestamp, inv.InvalidatedDerivedTimestamp)
		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block5, pair.Source)
		require.Equal(t, l2Block2, pair.Derived)

		inv = &reads.TestInvalidator{}
		// Now rewind to L1 block 3 (inclusive).
		require.NoError(t, db.RewindToSource(inv, l1Block3.ID()))
		require.True(t, inv.Invalidated)
		require.Equal(t, l1Block3.Number, inv.InvalidatedSourceNum)
		require.Equal(t, l2Block1.Timestamp, inv.InvalidatedDerivedTimestamp)

		// See if we find consistent data
		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block3, pair.Source)
		require.Equal(t, l2Block1, pair.Derived)

		inv = &reads.TestInvalidator{}
		// Rewind further to L1 block 1 (inclusive).
		require.NoError(t, db.RewindToSource(inv, l1Block1.ID()))
		require.True(t, inv.Invalidated)
		require.Equal(t, l1Block1.Number, inv.InvalidatedSourceNum)
		require.Equal(t, l2Block1.Timestamp, inv.InvalidatedDerivedTimestamp)
		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block1, pair.Source)
		require.Equal(t, l2Block1, pair.Derived)

		// Rewind further to L1 block 0 (inclusive).
		require.NoError(t, db.RewindToSource(inv, l1Block0.ID()))
		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block0, pair.Source)
		require.Equal(t, l2Block0, pair.Derived)
	})
}

// TestRewindToFirstDerived tests what happens if we rewind based on when a block was first derived.
func TestRewindToFirstDerived(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)
	l1Block3 := mockL1(3)
	l1Block4 := mockL1(4)
	l1Block5 := mockL1(5)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)
	l2Block3 := mockL2(3)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		// L2 genesis derived from L1 genesis
		l2Ref0 := toRef(l2Block0, common.Hash{})
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block0, common.Hash{}), l2Ref0))
		// Many L1 blocks all repeating the same L2 block
		l2Ref1 := toRef(l2Block1, l2Block0.Hash)
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block1, l1Block0.Hash), l2Ref0)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block1, l1Block0.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block2, l1Block1.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block3, l1Block2.Hash), l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block4, l1Block3.Hash), l2Ref1))
		// New L1 block that finally produces a new L2 block
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block5, l1Block4.Hash), l2Ref1)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, toRef(l1Block5, l1Block4.Hash), toRef(l2Block2, l2Block1.Hash)))
	}, func(t *testing.T, db *DB, m *stubMetrics) {

		pair, err := db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block5, pair.Source)
		require.Equal(t, l2Block2, pair.Derived)

		inv := &reads.TestInvalidator{}
		// Rewind to the future
		require.ErrorIs(t, db.RewindToFirstDerived(inv, l2Block3.ID(), types.RevisionAny), types.ErrFuture)

		// Rewind to the exact block we're at
		require.NoError(t, db.RewindToFirstDerived(inv, l2Block2.ID(), types.RevisionAny))
		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block5, pair.Source)
		require.Equal(t, l2Block2, pair.Derived)

		// Now rewind to L2 block 1
		require.NoError(t, db.RewindToFirstDerived(inv, l2Block1.ID(), types.RevisionAny))

		// See if we went back to the first occurrence of L2 block 1.
		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block1, pair.Source)
		require.Equal(t, l2Block1, pair.Derived)

		// Rewind further to L2 block 0 (inclusive).
		require.NoError(t, db.RewindToFirstDerived(inv, l2Block0.ID(), types.RevisionAny))
		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, l1Block0, pair.Source)
		require.Equal(t, l2Block0, pair.Derived)
	})
}

func TestInvalidateAndReplace(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)

	l1Ref0 := toRef(l1Block0, common.Hash{})
	l1Ref1 := toRef(l1Block1, l1Block0.Hash)
	l1Ref2 := toRef(l1Block1, l1Block0.Hash)
	l1Ref3 := toRef(l1Block1, l1Block0.Hash)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)
	l2Block3 := mockL2(3)

	l2Ref0 := toRef(l2Block0, common.Hash{})
	l2Ref1 := toRef(l2Block1, l2Block0.Hash)
	l2Ref2 := toRef(l2Block2, l2Block1.Hash)
	l2Ref3 := toRef(l2Block3, l2Block2.Hash)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref0)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref1)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref2))
		require.NoError(t, dbAddDerivedAny(db, l1Ref3, l2Ref2)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, l1Ref3, l2Ref3))
	}, func(t *testing.T, db *DB, m *stubMetrics) {
		pair, err := db.Last()
		require.NoError(t, err)
		require.Equal(t, l2Ref3.ID(), pair.Derived.ID())
		require.Equal(t, l1Block1.ID(), pair.Source.ID())

		_, err = db.Invalidated()
		require.ErrorIs(t, err, types.ErrConflict)

		replacement := l2Ref2
		replacement.Hash = common.Hash{0xff, 0xff, 0xff}
		require.NotEqual(t, l2Ref2.Hash, replacement.Hash) // different L2 block as replacement

		inv := &reads.TestInvalidator{}
		_, err = db.ReplaceInvalidatedBlock(inv, replacement, l2Ref2.Hash)
		require.ErrorIs(t, err, types.ErrConflict, "cannot replace what has not been invalidated")

		invalidated := types.DerivedBlockRefPair{
			Source:  l1Ref1,
			Derived: l2Ref2,
		}
		require.NoError(t, db.RewindAndInvalidate(inv, invalidated))
		_, err = db.Last()
		require.ErrorIs(t, err, types.ErrAwaitReplacementBlock)

		pair, err = db.Invalidated()
		require.NoError(t, err)
		require.Equal(t, invalidated.Source.ID(), pair.Source.ID())
		require.Equal(t, invalidated.Derived.ID(), pair.Derived.ID())

		_, err = db.ReplaceInvalidatedBlock(inv, replacement, common.Hash{0xba, 0xd})
		require.ErrorIs(t, err, types.ErrConflict, "must point at the right invalidated block")

		result, err := db.ReplaceInvalidatedBlock(inv, replacement, invalidated.Derived.Hash)
		require.NoError(t, err)
		require.Equal(t, replacement.ID(), result.Derived.ID())
		require.Equal(t, l1Block1.ID(), result.Source.ID())

		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, replacement.ID(), pair.Derived.ID())
		require.Equal(t, l1Block1.ID(), pair.Source.ID())
	})
}

// TestInvalidateAndReplaceNonFirst covers an edge-case where we invalidate an L2 block,
// but only at a later L1 scope, after the L2 block has already been derived from previous L1 blocks.
// At previous L1 blocks, the original L2 block is still needed, for accurate local-safe information,
// as future L1 data does not retroactively change the interpretation of past data within that past scope.
func TestInvalidateAndReplaceNonFirst(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)

	l1Ref0 := toRef(l1Block0, common.Hash{})
	l1Ref1 := toRef(l1Block1, l1Block0.Hash)
	l1Ref2 := toRef(l1Block2, l1Block1.Hash)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)
	l2Block3 := mockL2(3)
	l2Block4 := mockL2(4)

	l2Ref0 := toRef(l2Block0, common.Hash{})
	l2Ref1 := toRef(l2Block1, l2Block0.Hash)
	l2Ref2 := toRef(l2Block2, l2Block1.Hash)
	l2Ref3 := toRef(l2Block3, l2Block2.Hash)
	l2Ref4 := toRef(l2Block4, l2Block3.Hash)

	runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {
		require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref0)) // bump scope
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref1))
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref2))
		require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref3))
		// note the repeat of the L2 block with the bump in L1 scope
		require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref3)) // to be invalidated and replaced
		require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref4))
	}, func(t *testing.T, db *DB, m *stubMetrics) {
		pair, err := db.Last()
		require.NoError(t, err)
		require.Equal(t, l2Ref4.ID(), pair.Derived.ID())
		require.Equal(t, l1Block2.ID(), pair.Source.ID())

		_, err = db.Invalidated()
		require.ErrorIs(t, err, types.ErrConflict)

		inv := &reads.TestInvalidator{}
		invalidated := types.DerivedBlockRefPair{
			Source:  l1Ref2,
			Derived: l2Ref3,
		}
		require.NoError(t, db.RewindAndInvalidate(inv, invalidated))
		_, err = db.Last()
		require.ErrorIs(t, err, types.ErrAwaitReplacementBlock)

		pair, err = db.Invalidated()
		require.NoError(t, err)
		require.Equal(t, invalidated.Source.ID(), pair.Source.ID())
		require.Equal(t, invalidated.Derived.ID(), pair.Derived.ID())

		replacement := l2Ref3
		replacement.Hash = common.Hash{0xff, 0xff, 0xff}
		require.NotEqual(t, l2Ref3.Hash, replacement.Hash) // different L2 block as replacement
		result, err := db.ReplaceInvalidatedBlock(inv, replacement, invalidated.Derived.Hash)
		require.NoError(t, err)
		require.Equal(t, replacement.ID(), result.Derived.ID())
		require.Equal(t, l1Block2.ID(), result.Source.ID())

		pair, err = db.Last()
		require.NoError(t, err)
		require.Equal(t, replacement.ID(), pair.Derived.ID())
		require.Equal(t, l1Block2.ID(), pair.Source.ID())

		// The L2 block before the replacement should point to 2
		prev, err := db.PreviousDerived(replacement.ID(), types.RevisionAny)
		require.NoError(t, err)
		require.Equal(t, l2Ref2.ID(), prev.ID())

		lastFrom1, err := db.SourceToLastDerived(l1Block1.ID())
		require.NoError(t, err)
		// while invalidated, at this point in L1, it was still the local-safe block
		require.Equal(t, l2Ref3.ID(), lastFrom1.ID())

		// This should point to the original, since we traverse based on L1 scope
		entryBlock3, err := db.Next(types.DerivedIDPair{
			Source:  l1Block1.ID(),
			Derived: l2Ref2.ID()})
		require.NoError(t, err)
		require.Equal(t, l2Ref3.ID(), entryBlock3.Derived.ID())
		require.Equal(t, l1Block1.ID(), entryBlock3.Source.ID())

		// And then find the replacement, once we traverse further
		entryBlockRepl, err := db.Next(types.DerivedIDPair{
			Source:  l1Block1.ID(),
			Derived: l2Ref3.ID()})
		require.NoError(t, err)
		require.Equal(t, replacement.ID(), entryBlockRepl.Derived.ID())
		require.Equal(t, l1Block2.ID(), entryBlockRepl.Source.ID())

		// Check if canonical chain is represented accurately
		require.NoError(t, db.ContainsDerived(l2Ref2.ID(), types.RevisionAny), "common block 2 is valid part of canonical chain")
		require.NoError(t, db.ContainsDerived(replacement.ID(), types.RevisionAny), "replacement is valid part of canonical chain")
		require.ErrorIs(t, db.ContainsDerived(l2Ref3.ID(), types.RevisionAny), types.ErrConflict, "invalidated block is not valid in canonical chain")
	})
}

func TestNoInvalidatedFirst(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {},
		func(t *testing.T, db *DB, m *stubMetrics) {
			l1Block20 := toRef(mockL1(20), mockL1(19).Hash)
			l2Block100 := toRef(mockL1(100), mockL2(99).Hash)
			require.ErrorIs(t, db.addLink(l1Block20, l2Block100, l2Block100.Hash, FirstRevision), types.ErrConflict)
		},
	)
}

func TestNoReplaceFirst(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {},
		func(t *testing.T, db *DB, m *stubMetrics) {
			l2Block0 := mockL1(0)
			l2Ref0 := toRef(l2Block0, common.Hash{})
			inv := &reads.TestInvalidator{}
			_, err := db.ReplaceInvalidatedBlock(inv, l2Ref0, common.Hash{0xff})
			require.ErrorIs(t, err, types.ErrFuture)
			require.True(t, inv.Invalidated)
			require.Equal(t, l2Ref0.Time, inv.InvalidatedDerivedTimestamp)
			require.Equal(t, uint64(0), inv.InvalidatedSourceNum, "source not affected")
		},
	)
}

// After a block is invalidated, we cannot build on top of it like nothing happened.
func TestNotOntoInvalidated(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)

	l1Ref0 := toRef(l1Block0, common.Hash{})
	l1Ref1 := toRef(l1Block1, l1Block0.Hash)
	l1Ref2 := toRef(l1Block2, l1Block1.Hash)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)
	l2Block3 := mockL2(3)

	l2Ref0 := toRef(l2Block0, common.Hash{})
	l2Ref1 := toRef(l2Block1, l2Block0.Hash)
	l2Ref2 := toRef(l2Block2, l2Block1.Hash)
	l2Ref3 := toRef(l2Block3, l2Block2.Hash)

	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref0))
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref1))
			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref1))
			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref2))
			inv := &reads.TestInvalidator{}
			require.NoError(t, db.RewindAndInvalidate(inv, types.DerivedBlockRefPair{
				Source:  l1Ref2,
				Derived: l2Ref2,
			}))
			require.True(t, inv.Invalidated)
			require.Equal(t, l1Ref2.Number, inv.InvalidatedSourceNum)
			require.Equal(t, l2Ref2.Time, inv.InvalidatedDerivedTimestamp)
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.ErrorIs(t, dbAddDerivedAny(db, l1Ref2, l2Ref3), types.ErrAwaitReplacementBlock)
		},
	)
}

// When invalidating a block, we must have a matching hash, to not invalidate the wrong thing
func TestMismatchedInvalidate(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)
	l1Block3 := mockL1(3)

	l1Ref0 := toRef(l1Block0, common.Hash{})
	l1Ref1 := toRef(l1Block1, l1Block0.Hash)
	l1Ref2 := toRef(l1Block2, l1Block1.Hash)
	l1Ref3 := toRef(l1Block3, l1Block2.Hash)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)

	l2Ref0 := toRef(l2Block0, common.Hash{})
	l2Ref1 := toRef(l2Block1, l2Block0.Hash)
	l2Ref2 := toRef(l2Block2, l2Block1.Hash)
	l2Ref2Alt := l2Ref2
	l2Ref2Alt.Hash = common.Hash{0xff}

	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref0))
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref1))
			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref1))
			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref2))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			inv := &reads.TestInvalidator{}
			// This will detect the issue upon rewinding
			require.ErrorIs(t, db.RewindAndInvalidate(inv, types.DerivedBlockRefPair{
				Source:  l1Ref2,
				Derived: l2Ref2Alt,
			}), types.ErrConflict)
			require.True(t, inv.Invalidated) // even though no effect, we still pick up the signal as clue something is wrong.
			require.Equal(t, l1Ref2.Number, inv.InvalidatedSourceNum)
			require.Equal(t, l2Ref2Alt.Time, inv.InvalidatedDerivedTimestamp)
			// This will detect the issue upon insertion of the new invalidated-entry
			require.ErrorIs(t, db.addLink(l1Ref3, l2Ref2Alt, l2Ref2Alt.Hash, FirstRevision), types.ErrConflict)
		},
	)
}

// L1 and L2 blocks cannot be bumped up at once.
func TestNoParallelBump(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)

	l1Ref0 := toRef(l1Block0, common.Hash{})
	l1Ref1 := toRef(l1Block1, l1Block0.Hash)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)

	l2Ref0 := toRef(l2Block0, common.Hash{})
	l2Ref1 := toRef(l2Block1, l2Block0.Hash)

	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.ErrorIs(t, dbAddDerivedAny(db, l1Ref1, l2Ref1), types.ErrOutOfOrder)
		},
	)
}

func TestLookupDetectIfCorruptDB(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)

	l1Ref0 := toRef(l1Block0, common.Hash{})

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)

	l2Ref0 := toRef(l2Block0, common.Hash{})

	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			// Skip L1 block 1 and L2 block 1, force it into the DB
			e := LinkEntry{
				source:      l1Block2,
				derived:     l2Block2,
				invalidated: false,
			}
			require.NoError(t, db.store.Append(e.encode()))
			db.m.RecordDBDerivedEntryCount(db.store.Size())

			// Look for L1 block 1
			_, err := db.SourceToLastDerived(l1Block1.ID())
			require.ErrorIs(t, err, types.ErrNotExact)

			// Look for L2 block 1
			_, err = db.DerivedToFirstSource(l2Block1.ID(), types.RevisionAny)
			require.ErrorIs(t, err, types.ErrNotExact)

			inv := &reads.TestInvalidator{}
			// Rewind, corrupt data cannot be left at end of test, otherwise invariant checks fail
			require.NoError(t, db.Rewind(inv, types.DerivedBlockSealPair{
				Source:  l1Block0,
				Derived: l2Block0,
			}, true))
		},
	)
}

// Do not rewind to a block that does not actually match the target to rewind to.
func TestRewindToDifferent(t *testing.T) {
	l1Block0 := mockL1(0)
	l1Block1 := mockL1(1)
	l1Block2 := mockL1(2)

	l1Ref0 := toRef(l1Block0, common.Hash{})
	l1Ref1 := toRef(l1Block1, l1Block0.Hash)
	l1Ref2 := toRef(l1Block2, l1Block1.Hash)

	l2Block0 := mockL2(0)
	l2Block1 := mockL2(1)
	l2Block2 := mockL2(2)

	l2Ref0 := toRef(l2Block0, common.Hash{})
	l2Ref1 := toRef(l2Block1, l2Block0.Hash)
	l2Ref2 := toRef(l2Block2, l2Block1.Hash)

	l1ID1Alt := l1Ref1.ID()
	l1ID1Alt.Hash = common.Hash{0xff}

	l2ID1Alt := l2Ref1.ID()
	l2ID1Alt.Hash = common.Hash{0xff}

	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref0))
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref1))
			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref1))
			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref2))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {

			t.Run("Bad derived target", func(t *testing.T) {
				inv := &reads.TestInvalidator{}
				// try to rewind, but towards a mismatching block
				require.ErrorIs(t, db.RewindToFirstDerived(inv, l2ID1Alt, types.RevisionAny), types.ErrConflict)
				require.False(t, inv.Invalidated)
				last, err := db.Last()
				require.NoError(t, err)
				// assert we didn't rewind anything
				require.Equal(t, l2Block2, last.Derived)
				require.Equal(t, l1Block2, last.Source)
			})

			t.Run("Bad source target", func(t *testing.T) {
				inv := &reads.TestInvalidator{}
				// try to rewind, but towards a mismatching block
				require.ErrorIs(t, db.RewindToSource(inv, l1ID1Alt), types.ErrConflict)
				require.False(t, inv.Invalidated)
				last, err := db.Last()
				require.NoError(t, err)
				// assert we didn't rewind anything
				require.Equal(t, l2Block2, last.Derived)
				require.Equal(t, l1Block2, last.Source)
			})
		},
	)
}

// TestDeepInvalidate tests that a local-safe DB can invalidate and rewind entries,
// even if multiple new local-safe entries with new source blocks or derived blocks have already been added.
// The blocks stay attached to these sources, but the derived number will fall back and rebuild.
// This causes different derived blocks with the same block-number to show up multiple times in the DB,
// and is why we need revisions to distinguish them and search them.
func TestDeepInvalidate(t *testing.T) {
	l1Ref0 := mockL1Ref(0)
	l1Ref1 := mockL1Ref(1)
	l1Ref2 := mockL1Ref(2)
	l1Ref3 := mockL1Ref(3)
	l1Ref4 := mockL1Ref(4)
	l1Ref5 := mockL1Ref(5)
	l1Ref6 := mockL1Ref(6) // candidate cross-safe scope where we learn prior derived blocks need to be invalidated
	l1Ref7 := mockL1Ref(7)
	l1Ref8 := mockL1Ref(8)

	l2Ref0 := mockL2Ref(0) // genesis
	l2Ref1 := mockL2Ref(1) // canonical, from L1 1
	l2Ref2 := mockL2Ref(2) // canonical, from L1 2
	l2Ref3 := mockL2Ref(3) // optimistic, from L1 3, to be non-canonical once we reach L1 6
	l2Ref4 := mockL2Ref(4) // optimistic, from L1 3, to be non-canonical once we reach L1 6
	l2Ref5 := mockL2Ref(5) // optimistic, from L1 3, to be non-canonical once we reach L1 6
	l2Ref6 := mockL2Ref(6) // optimistic, from L1 4, to be non-canonical once we reach L1 6
	l2Ref7 := mockL2Ref(7) // optimistic, from L1 4, to be non-canonical once we reach L1 6
	l2Ref8 := mockL2Ref(8) // optimistic, from L1 5, to be non-canonical once we reach L1 6
	l2Ref9 := mockL2Ref(9) // optimistic, from L1 6, to be non-canonical once we reach L1 6 (cross-verification is after local derivation)

	secondRevision := types.Revision(3)
	thirdRevision := types.Revision(5)

	l2Ref3p := l2Ref3 // canonical later, replacement block, from L1 6
	l2Ref3p.Hash[0] ^= 0xff

	l2Ref4p := l2Ref4 // canonical later, block after replacement block, from L1 7
	l2Ref4p.ParentHash = l2Ref3p.Hash
	l2Ref4p.Hash[0] ^= 0xff

	l2Ref5p := l2Ref5 // canonical later, block after replacement block, from L1 7, but replaced again!
	l2Ref5p.ParentHash = l2Ref4p.Hash
	l2Ref5p.Hash[0] ^= 0xff

	l2Ref6p := l2Ref6 // canonical later, block after replacement block, from L1 7, replaced again
	l2Ref6p.ParentHash = l2Ref5p.Hash
	l2Ref6p.Hash[0] ^= 0xff

	l2Ref5pp := l2Ref5 // canonical later, block after replacement block, from L1 8
	l2Ref5pp.ParentHash = l2Ref4p.Hash
	l2Ref5pp.Hash[0] ^= 0xff
	l2Ref5pp.Hash[1] ^= 0xff

	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))

			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref0)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref1))

			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref1)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref2))

			require.NoError(t, dbAddDerivedAny(db, l1Ref3, l2Ref2)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref3, l2Ref3))
			require.NoError(t, dbAddDerivedAny(db, l1Ref3, l2Ref4))
			require.NoError(t, dbAddDerivedAny(db, l1Ref3, l2Ref5))

			require.NoError(t, dbAddDerivedAny(db, l1Ref4, l2Ref5)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref4, l2Ref6))
			require.NoError(t, dbAddDerivedAny(db, l1Ref4, l2Ref7))

			require.NoError(t, dbAddDerivedAny(db, l1Ref5, l2Ref7)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref5, l2Ref8))

			require.NoError(t, dbAddDerivedAny(db, l1Ref6, l2Ref8)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref6, l2Ref9))

			_, _, err := db.lookup(6, 3) // 3 was never derived from 6 at this point, we didn't apply the invalidation yet
			require.ErrorIs(t, err, types.ErrNotExact)

			// If we scan for the first thing at or after (6, 3) we should find (6, 8)
			_, v, err := db.lookupOrAfter(6, 3)
			require.NoError(t, err)
			require.Equal(t, l1Ref6.ID(), v.source.ID())
			require.Equal(t, l2Ref8.ID(), v.derived.ID())
			require.Equal(t, FirstRevision, v.revision)

			inv := &reads.TestInvalidator{}
			invalidated := types.DerivedBlockRefPair{
				Source:  l1Ref6,
				Derived: l2Ref3,
			}
			require.NoError(t, db.RewindAndInvalidate(inv, invalidated))
			require.True(t, inv.Invalidated)

			_, v, err = db.lookup(6, 3) // now we have an invalidated place-holder that matches the (6, 3) lookup
			require.NoError(t, err)
			require.True(t, v.invalidated)
			require.Equal(t, l1Ref6.ID(), v.source.ID())
			require.Equal(t, l2Ref3.ID(), v.derived.ID())
			require.Equal(t, secondRevision, v.revision)

			// Invalidated, but with old revision before invalidation, is considered valid
			require.NoError(t, db.ContainsDerived(l2Ref3.ID(), FirstRevision))
			// Invalidated entries are not considered by ContainsDerived as valid
			require.ErrorIs(t, db.ContainsDerived(l2Ref3.ID(), secondRevision), types.ErrAwaitReplacementBlock)
			// Older non-invalidated entries are considered valid still
			require.NoError(t, db.ContainsDerived(l2Ref2.ID(), FirstRevision))

			// We cannot invalidate what is already invalidated
			require.ErrorIs(t, db.RewindAndInvalidate(inv, invalidated), types.ErrAwaitReplacementBlock)

			// we cannot proceed in any way until actually replacing the invalidated placeholder block
			require.ErrorIs(t, dbAddDerivedAny(db, l1Ref6, l2Ref4p), types.ErrAwaitReplacementBlock)
			require.ErrorIs(t, dbAddDerivedAny(db, l1Ref7, l2Ref3p), types.ErrAwaitReplacementBlock)
			require.ErrorIs(t, dbAddDerivedAny(db, l1Ref7, l2Ref4p), types.ErrAwaitReplacementBlock)

			replacement, err := db.ReplaceInvalidatedBlock(inv, l2Ref3p, l2Ref3.Hash)
			require.NoError(t, err)
			require.Equal(t, l1Ref6.ID(), replacement.Source.ID())
			require.Equal(t, l2Ref3p.ID(), replacement.Derived.ID())

			_, v, err = db.lookup(6, 3)
			require.NoError(t, err)
			require.False(t, v.invalidated) // not invalid, this is canonical now
			require.Equal(t, l1Ref6.ID(), v.source.ID())
			require.Equal(t, l2Ref3p.ID(), v.derived.ID())
			require.Equal(t, secondRevision, v.revision) // replacement matches invalidated-placeholder revision

			require.NoError(t, db.ContainsDerived(l2Ref3.ID(), FirstRevision), types.ErrConflict)
			require.ErrorIs(t, db.ContainsDerived(l2Ref3.ID(), secondRevision), types.ErrConflict)
			require.NoError(t, db.ContainsDerived(l2Ref3p.ID(), secondRevision))

			require.NoError(t, dbAddDerivedAny(db, l1Ref7, l2Ref3p)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref7, l2Ref4p))
			require.NoError(t, dbAddDerivedAny(db, l1Ref7, l2Ref5p))
			require.NoError(t, dbAddDerivedAny(db, l1Ref7, l2Ref6p))

			require.NoError(t, dbAddDerivedAny(db, l1Ref8, l2Ref6p)) // scope bump
		},
		func(t *testing.T, db *DB, m *stubMetrics) {

			_, link, err := db.lookup(5, 7)
			require.NoError(t, err)
			require.Equal(t, l1Ref5.ID(), link.source.ID())
			require.Equal(t, l2Ref7.ID(), link.derived.ID()) // not canonical in L2, but available, attached to older L1 scope

			_, link, err = db.lookup(5, 3) // block 3 was passed already in scope 5, but returned back to in scope 6, and should thus not be a valid lookup
			require.ErrorIs(t, err, types.ErrNotExact)

			_, link, err = db.lookup(6, 3)
			require.NoError(t, err)
			require.Equal(t, l1Ref6.ID(), link.source.ID())
			require.Equal(t, l2Ref3p.ID(), link.derived.ID())

			_, link, err = db.lookup(7, 3)
			require.NoError(t, err)
			require.Equal(t, l1Ref7.ID(), link.source.ID())
			require.Equal(t, l2Ref3p.ID(), link.derived.ID())

			_, link, err = db.lookup(7, 4)
			require.NoError(t, err)
			require.Equal(t, l1Ref7.ID(), link.source.ID())
			require.Equal(t, l2Ref4p.ID(), link.derived.ID())

			// Now that we have a chain that jumps from (source=5, derived=8) to (source=6, derived=3),
			// let's do some searches and assert we get the expected results.

			checkRevisionsOf3 := func(t *testing.T) {
				_, v, err := db.derivedNumToLastSource(3, FirstRevision)
				require.NoError(t, err)
				require.Equal(t, l1Ref3.ID(), v.source.ID())
				require.Equal(t, l2Ref3.ID(), v.derived.ID())
				_, v, err = db.derivedNumToLastSource(3, secondRevision)
				require.NoError(t, err)
				require.Equal(t, l1Ref7.ID(), v.source.ID())
				require.Equal(t, l2Ref3p.ID(), v.derived.ID())

				_, v, err = db.derivedNumToFirstSource(3, FirstRevision)
				require.NoError(t, err)
				require.Equal(t, l1Ref3.ID(), v.source.ID())
				require.Equal(t, l2Ref3.ID(), v.derived.ID())
				_, v, err = db.derivedNumToFirstSource(3, secondRevision)
				require.NoError(t, err)
				require.Equal(t, l1Ref6.ID(), v.source.ID())
				require.Equal(t, l2Ref3p.ID(), v.derived.ID())
			}
			checkRevisionsOf3(t)
			checkRevisionsOf4 := func(t *testing.T) {
				_, v, err := db.derivedNumToLastSource(4, FirstRevision)
				require.NoError(t, err)
				require.Equal(t, l1Ref3.ID(), v.source.ID())
				require.Equal(t, l2Ref4.ID(), v.derived.ID())

				_, v, err = db.derivedNumToLastSource(4, secondRevision)
				require.NoError(t, err)
				require.Equal(t, l1Ref7.ID(), v.source.ID())
				require.Equal(t, l2Ref4p.ID(), v.derived.ID())

				_, v, err = db.derivedNumToFirstSource(4, FirstRevision)
				require.NoError(t, err)
				require.Equal(t, l1Ref3.ID(), v.source.ID())
				require.Equal(t, l2Ref4.ID(), v.derived.ID())
				_, v, err = db.derivedNumToFirstSource(4, secondRevision)
				require.NoError(t, err)
				require.Equal(t, l1Ref7.ID(), v.source.ID())
				require.Equal(t, l2Ref4p.ID(), v.derived.ID())
			}
			checkRevisionsOf4(t)

			inv := &reads.TestInvalidator{}
			// Invalidate again, now replace 5p with 5pp, at L1 8
			invalidated2 := types.DerivedBlockRefPair{
				Source:  l1Ref8,
				Derived: l2Ref5p,
			}
			require.NoError(t, db.RewindAndInvalidate(inv, invalidated2))
			require.True(t, inv.Invalidated)
			require.Equal(t, l1Ref8.Number, inv.InvalidatedSourceNum)
			require.Equal(t, l2Ref5p.Time, inv.InvalidatedDerivedTimestamp)

			_, v, err := db.lookup(8, 5) // now we have an invalidated place-holder that matches the (8, 5) lookup
			require.NoError(t, err)
			require.True(t, v.invalidated)
			require.Equal(t, l1Ref8.ID(), v.source.ID())
			require.Equal(t, l2Ref5p.ID(), v.derived.ID())
			require.Equal(t, thirdRevision, v.revision)

			inv = &reads.TestInvalidator{}
			replacement2, err := db.ReplaceInvalidatedBlock(inv, l2Ref5pp, l2Ref5p.Hash)
			require.NoError(t, err)
			require.Equal(t, l1Ref8.ID(), replacement2.Source.ID())
			require.Equal(t, l2Ref5pp.ID(), replacement2.Derived.ID())

			require.True(t, inv.Invalidated)
			require.Equal(t, uint64(0), inv.InvalidatedSourceNum, "source stays the same, unaffected")
			require.Equal(t, l2Ref5pp.Time, inv.InvalidatedDerivedTimestamp)

			checkRevisionsOf3(t)
			checkRevisionsOf4(t)

			// We have 3 occurrences of block 5 in the DB now. Let's search for them all.
			_, v, err = db.derivedNumToLastSource(5, FirstRevision)
			require.NoError(t, err)
			require.Equal(t, l1Ref4.ID(), v.source.ID())
			require.Equal(t, l2Ref5.ID(), v.derived.ID())
			_, v, err = db.derivedNumToLastSource(5, secondRevision)
			require.NoError(t, err)
			require.Equal(t, l1Ref7.ID(), v.source.ID())
			require.Equal(t, l2Ref5p.ID(), v.derived.ID())
			_, v, err = db.derivedNumToLastSource(5, thirdRevision)
			require.NoError(t, err)
			require.Equal(t, l1Ref8.ID(), v.source.ID())
			require.Equal(t, l2Ref5pp.ID(), v.derived.ID())

			_, v, err = db.derivedNumToFirstSource(5, FirstRevision)
			require.NoError(t, err)
			require.Equal(t, l1Ref3.ID(), v.source.ID())
			require.Equal(t, l2Ref5.ID(), v.derived.ID())
			_, v, err = db.derivedNumToFirstSource(5, secondRevision)
			require.NoError(t, err)
			require.Equal(t, l1Ref7.ID(), v.source.ID())
			require.Equal(t, l2Ref5p.ID(), v.derived.ID())
			_, v, err = db.derivedNumToFirstSource(5, thirdRevision)
			require.NoError(t, err)
			require.Equal(t, l1Ref8.ID(), v.source.ID())
			require.Equal(t, l2Ref5pp.ID(), v.derived.ID())

			require.NoError(t, db.ContainsDerived(l2Ref2.ID(), FirstRevision))
			require.ErrorIs(t, db.ContainsDerived(l2Ref3.ID(), secondRevision), types.ErrConflict)
			require.NoError(t, db.ContainsDerived(l2Ref3p.ID(), secondRevision))

			require.NoError(t, db.ContainsDerived(l2Ref5.ID(), FirstRevision))
			require.ErrorIs(t, db.ContainsDerived(l2Ref5p.ID(), FirstRevision), types.ErrConflict)
			require.NoError(t, db.ContainsDerived(l2Ref5p.ID(), secondRevision))
			require.ErrorIs(t, db.ContainsDerived(l2Ref5p.ID(), thirdRevision), types.ErrConflict)
			require.NoError(t, db.ContainsDerived(l2Ref5pp.ID(), thirdRevision))
		})
}

// TestDerivedToRevision matches what we see in the cross-safe DB:
// some revised data is added, without rewinding and invalidating,
// and leaving no alternative derived entries with the same number.
func TestDerivedToRevision(t *testing.T) {
	l1Ref0 := mockL1Ref(0)
	l1Ref1 := mockL1Ref(1)
	l1Ref2 := mockL1Ref(2)
	l1Ref3 := mockL1Ref(3)
	l1Ref4 := mockL1Ref(4)

	l2Ref0 := mockL2Ref(0)
	l2Ref1 := mockL2Ref(1)
	l2Ref2 := mockL2Ref(2)
	l2Ref3 := mockL2Ref(3)
	l2Ref4 := mockL2Ref(4)
	l2Ref5 := mockL2Ref(5)
	l2Ref6 := mockL2Ref(6)
	l2Ref7 := mockL2Ref(7)

	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			_, err := db.DerivedToRevision(l2Ref0.ID())
			require.ErrorIs(t, err, types.ErrFuture)

			require.NoError(t, dbAddDerivedAny(db, l1Ref0, l2Ref0))

			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref0)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref1))
			require.NoError(t, dbAddDerivedAny(db, l1Ref1, l2Ref2))

			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref2)) // scope bump
			require.NoError(t, db.AddDerived(l1Ref2, l2Ref3, 3))
			require.NoError(t, dbAddDerivedAny(db, l1Ref2, l2Ref4))

			require.NoError(t, dbAddDerivedAny(db, l1Ref3, l2Ref4)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref3, l2Ref4))
			// when something is revised, the number of the revised data is used as revision number
			require.ErrorIs(t, db.AddDerived(l1Ref3, l2Ref5, 6), types.ErrDataCorruption)
			require.NoError(t, db.AddDerived(l1Ref3, l2Ref5, 5))

			require.NoError(t, dbAddDerivedAny(db, l1Ref4, l2Ref5)) // scope bump
			require.NoError(t, dbAddDerivedAny(db, l1Ref4, l2Ref6))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			v, err := db.DerivedToRevision(l2Ref0.ID())
			require.NoError(t, err)
			require.Equal(t, FirstRevision, v)

			v, err = db.DerivedToRevision(l2Ref1.ID())
			require.NoError(t, err)
			require.Equal(t, FirstRevision, v)

			v, err = db.DerivedToRevision(l2Ref2.ID())
			require.NoError(t, err)
			require.Equal(t, FirstRevision, v)

			v, err = db.DerivedToRevision(l2Ref3.ID()) // this was a revised entry
			require.NoError(t, err)
			require.Equal(t, types.Revision(3), v)

			v, err = db.DerivedToRevision(l2Ref4.ID())
			require.NoError(t, err)
			require.Equal(t, types.Revision(3), v)

			v, err = db.DerivedToRevision(l2Ref5.ID())
			require.NoError(t, err)
			require.Equal(t, types.Revision(5), v)

			v, err = db.DerivedToRevision(l2Ref6.ID())
			require.NoError(t, err)
			require.Equal(t, types.Revision(5), v)

			_, err = db.DerivedToRevision(l2Ref7.ID())
			require.ErrorIs(t, err, types.ErrFuture)
		})
}
