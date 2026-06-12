package interop

import (
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	bolt "go.etcd.io/bbolt"
)

// openDafnyTestVerifiedDB opens a fresh VerifiedDB in a temp dir and closes it
// on cleanup.
func openDafnyTestVerifiedDB(t *testing.T) *VerifiedDB {
	t.Helper()
	v, err := OpenVerifiedDB(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, v.Close()) })
	return v
}

// dafnyVerifiedResult builds a VerifiedResult at ts with per-chain head
// numbers given as chainID->number.
func dafnyVerifiedResult(ts uint64, headNumbers map[uint64]uint64) VerifiedResult {
	heads := make(map[eth.ChainID]eth.BlockID, len(headNumbers))
	for c, n := range headNumbers {
		heads[dafnyChainID(c)] = eth.BlockID{Hash: common.HexToHash("0xaa"), Number: n}
	}
	return VerifiedResult{
		Timestamp: ts,
		L2Heads:   heads,
	}
}

// putRawVerified writes raw bytes into the verified bucket, bypassing Commit.
func putRawVerified(t *testing.T, v *VerifiedDB, key, value []byte) {
	t.Helper()
	require.NoError(t, v.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketName).Put(key, value)
	}))
}

// putVerified writes a JSON-encoded VerifiedResult at its own timestamp key,
// bypassing Commit's sequential/cache bookkeeping.
func putVerified(t *testing.T, v *VerifiedDB, result VerifiedResult) {
	t.Helper()
	value, err := json.Marshal(result)
	require.NoError(t, err)
	putRawVerified(t, v, timestampToKey(result.Timestamp), value)
}

func TestCheckSequential(t *testing.T) {
	t.Parallel()

	require.NoError(t, checkSequential(nil))
	require.NoError(t, checkSequential([]uint64{1000}))
	require.NoError(t, checkSequential([]uint64{1000, 1001, 1002}))
	require.NoError(t, checkSequential([]uint64{1002, 1000, 1001}))
	require.ErrorContains(t, checkSequential([]uint64{1000, 1002}), "gap at 1001")
	require.ErrorContains(t, checkSequential([]uint64{1003, 1000, 1001}), "gap at 1002")
}

func TestCheckVerifiedDBValidPass(t *testing.T) {
	t.Parallel()

	t.Run("empty db", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, CheckVerifiedDBValid(v))
	})

	t.Run("sequential commits with monotone heads", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		require.NoError(t, v.Commit(dafnyVerifiedResult(1001, map[uint64]uint64{1: 100, 2: 201})))
		require.NoError(t, v.Commit(dafnyVerifiedResult(1002, map[uint64]uint64{1: 102, 2: 202})))
		require.NoError(t, CheckVerifiedDBValid(v))
	})

	t.Run("after rewind", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100})))
		require.NoError(t, v.Commit(dafnyVerifiedResult(1001, map[uint64]uint64{1: 101})))
		require.NoError(t, v.Commit(dafnyVerifiedResult(1002, map[uint64]uint64{1: 102})))
		deleted, err := v.Rewind(1001)
		require.NoError(t, err)
		require.True(t, deleted)
		require.NoError(t, CheckVerifiedDBValid(v))
	})
}

func TestCheckVerifiedDBValidViolations(t *testing.T) {
	t.Parallel()

	t.Run("conjunct 0: nil VerifiedDB", func(t *testing.T) {
		t.Parallel()
		err := CheckVerifiedDBValid(nil)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 0: VerifiedDB without store", func(t *testing.T) {
		t.Parallel()
		err := CheckVerifiedDBValid(&VerifiedDB{})
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 0: malformed key length", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		putRawVerified(t, v, []byte("bad"), []byte("{}"))
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "big-endian timestamp")
	})

	t.Run("conjunct 0: undecodable value", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		putRawVerified(t, v, timestampToKey(1000), []byte("not json"))
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "not a VerifiedResult")
	})

	t.Run("conjunct 1: gap in committed timestamps", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100})))
		putVerified(t, v, dafnyVerifiedResult(1002, map[uint64]uint64{1: 102}))
		v.lastTimestamp = 1002 // keep the conjunct (3) cache consistent
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "gap at 1001")
		require.NotContains(t, err.Error(), "conjunct (3)")
	})

	t.Run("conjunct 2: timestamp field differs from key", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100})))
		mismatched, err := json.Marshal(dafnyVerifiedResult(999, map[uint64]uint64{1: 100}))
		require.NoError(t, err)
		putRawVerified(t, v, timestampToKey(1000), mismatched)
		err = CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (2)")
		require.ErrorContains(t, err, "timestamp field 999")
	})

	t.Run("conjunct 3: empty db but cache initialized", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		v.initialized = true
		v.lastTimestamp = 1000
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "Some(1000)")
	})

	t.Run("conjunct 3: non-empty db but cache uninitialized", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		putVerified(t, v, dafnyVerifiedResult(1000, map[uint64]uint64{1: 100}))
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "lastTimestamp is None")
	})

	t.Run("conjunct 3: cached lastTimestamp behind MaxKey", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100})))
		require.NoError(t, v.Commit(dafnyVerifiedResult(1001, map[uint64]uint64{1: 101})))
		v.lastTimestamp = 1000
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "lastTimestamp 1000 != MaxKey(db) 1001")
	})

	t.Run("conjunct 4: per-chain head number decreases", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		putVerified(t, v, dafnyVerifiedResult(1001, map[uint64]uint64{1: 99, 2: 201}))
		v.lastTimestamp = 1001
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (4)")
		require.ErrorContains(t, err, "decreases from 100 at ts 1000 to 99 at ts 1001")
		require.NotContains(t, err.Error(), "conjunct (3)")
	})

	t.Run("conjunct 4: decrease across a timestamp missing the chain", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		putVerified(t, v, dafnyVerifiedResult(1001, map[uint64]uint64{2: 201}))
		putVerified(t, v, dafnyVerifiedResult(1002, map[uint64]uint64{1: 99, 2: 202}))
		v.lastTimestamp = 1002
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (4)")
		require.ErrorContains(t, err, "decreases from 100 at ts 1000 to 99 at ts 1002")
	})

	t.Run("joins violations across conjuncts", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100})))
		putVerified(t, v, dafnyVerifiedResult(1002, map[uint64]uint64{1: 99}))
		err := CheckVerifiedDBValid(v)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "conjunct (4)")
	})
}

func TestAssertVerifiedDBValid(t *testing.T) {
	t.Parallel()

	t.Run("pass leaves t untouched", func(t *testing.T) {
		t.Parallel()
		v := openDafnyTestVerifiedDB(t)
		require.NoError(t, v.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100})))
		ft := &fakeDafnyT{}
		AssertVerifiedDBValid(ft, v)
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	t.Run("violation reports and fails", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertVerifiedDBValid(ft, nil)
		require.True(t, ft.errorfCalled)
		require.True(t, ft.failNowCalled)
	})
}
