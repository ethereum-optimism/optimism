package interop

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// dafnyTestInterop builds an in-package Interop over a temp-dir VerifiedDB
// and per-chain seal mocks for chains 1 and 2, with protocol activation 1000.
func dafnyTestInterop(t *testing.T) *Interop {
	t.Helper()
	return &Interop{
		activationTimestamp: 1000,
		chains: map[eth.ChainID]cc.InteropChain{
			dafnyChainID(1): nil,
			dafnyChainID(2): nil,
		},
		verifiedDB: openDafnyTestVerifiedDB(t),
		logsDBs: map[eth.ChainID]LogsDB{
			dafnyChainID(1): dafnySealedMock(),
			dafnyChainID(2): dafnySealedMock(),
		},
	}
}

// dafnyHeadSeal builds a seal whose ID matches a dafnyVerifiedResult head at
// the given number.
func dafnyHeadSeal(number, timestamp uint64) suptypes.BlockSeal {
	return suptypes.BlockSeal{Hash: common.HexToHash("0xaa"), Number: number, Timestamp: timestamp}
}

// dafnySyncedInterop extends dafnyTestInterop with verified results at
// 1000..1002 (chain 1 heads 100..102, chain 2 heads 200..202) and logsDB
// mocks sealing exactly those heads.
func dafnySyncedInterop(t *testing.T) *Interop {
	t.Helper()
	i := dafnyTestInterop(t)
	for off := uint64(0); off <= 2; off++ {
		require.NoError(t, i.verifiedDB.Commit(
			dafnyVerifiedResult(1000+off, map[uint64]uint64{1: 100 + off, 2: 200 + off})))
	}
	i.logsDBs[dafnyChainID(1)] = dafnySealedMock(
		dafnyHeadSeal(100, 1000), dafnyHeadSeal(101, 1001), dafnyHeadSeal(102, 1002))
	i.logsDBs[dafnyChainID(2)] = dafnySealedMock(
		dafnyHeadSeal(200, 1000), dafnyHeadSeal(201, 1001), dafnyHeadSeal(202, 1002))
	return i
}

// mockLogsDBFor returns the seal mock backing the given chain's logsDB.
func mockLogsDBFor(t *testing.T, i *Interop, chain uint64) *sealsMockLogsDB {
	t.Helper()
	m, ok := i.logsDBs[dafnyChainID(chain)].(*sealsMockLogsDB)
	require.True(t, ok)
	return m
}

func TestCheckInteropValidPass(t *testing.T) {
	t.Parallel()

	t.Run("fresh instance", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckInteropValid(dafnyTestInterop(t)))
	})

	t.Run("synced instance", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckInteropValid(dafnySyncedInterop(t)))
	})

	t.Run("with valid pending transition", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionAdvance,
			Result: &Result{
				Timestamp: 1003,
				L2Heads:   dafnyTestHeads(1, 2),
			},
		}))
		require.NoError(t, CheckInteropValid(i))
	})
}

func TestCheckInteropValidViolations(t *testing.T) {
	t.Parallel()

	t.Run("conjunct 0: nil Interop", func(t *testing.T) {
		t.Parallel()
		err := CheckInteropValid(nil)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 2: logsDBs missing a chain", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (2)")
		require.ErrorContains(t, err, "logsDBs.Keys != CHAIN_IDS")
	})

	t.Run("conjunct 2: logsDBs has an extra chain", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[dafnyChainID(3)] = dafnySealedMock()
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 3: two chains share one logsDB", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[dafnyChainID(2)] = i.logsDBs[dafnyChainID(1)]
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "same instance")
	})

	t.Run("conjunct 4: invalid verifiedDB skips db conjuncts", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, i.verifiedDB.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		putVerified(t, i.verifiedDB, dafnyVerifiedResult(1002, map[uint64]uint64{1: 102, 2: 202}))
		i.verifiedDB.lastTimestamp = 1002 // keep the VerifiedDB cache conjunct quiet
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (4)")
		require.ErrorContains(t, err, "VerifiedDB.dfy Valid()")
		require.NotContains(t, err.Error(), "conjunct (5)")
		require.NotContains(t, err.Error(), "conjunct (6)")
		require.NotContains(t, err.Error(), "conjunct (7)")
	})

	t.Run("conjunct 5: first-verifiable timestamp not in db", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.firstTimestamp = 999 // stale cache: ACTIVATION_TIMESTAMP maps to 999
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (5)")
		require.ErrorContains(t, err, "ACTIVATION_TIMESTAMP 999 not in db")
		require.NotContains(t, err.Error(), "conjunct (6)")
	})

	t.Run("conjunct 6: committed timestamp below activation", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.firstTimestamp = 1001 // stale cache: ACTIVATION_TIMESTAMP maps to 1001
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (6)")
		require.ErrorContains(t, err, "committed timestamp 1000 below ACTIVATION_TIMESTAMP 1001")
		require.NotContains(t, err.Error(), "conjunct (5)")
	})

	t.Run("conjunct 7: committed result misses a chain", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, i.verifiedDB.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100})))
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (7)")
		require.ErrorContains(t, err, "db[1000].l2Heads.Keys != CHAIN_IDS")
	})

	t.Run("conjunct 8: stored pending transition invalid", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{Decision: DecisionWait}))
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (8)")
		require.ErrorContains(t, err, "Types.dfy ValidPendingTransition")
	})
}

func TestCheckDBsInSyncUpTo(t *testing.T) {
	t.Parallel()

	t.Run("pass: synced range on both chains", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckDBsInSyncUpTo(i, dafnyChainID(1), 1002))
		require.NoError(t, CheckDBsInSyncUpTo(i, dafnyChainID(2), 1002))
		require.NoError(t, CheckDBsInSyncUpTo(i, dafnyChainID(1), 1001))
	})

	t.Run("pass: vacuous when upper below activation", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckDBsInSyncUpTo(i, dafnyChainID(1), 999))
	})

	t.Run("conjunct 0: unknown chain", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckDBsInSyncUpTo(i, dafnyChainID(9), 1002)
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "has no logsDB")
	})

	t.Run("conjunct 1: timestamp missing from verifiedDB", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckDBsInSyncUpTo(i, dafnyChainID(1), 1003)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "verifiedDB.Has(1003) is false")
	})

	t.Run("conjunct 2: verified entry misses the chain", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		putVerified(t, i.verifiedDB, dafnyVerifiedResult(1001, map[uint64]uint64{2: 201}))
		err := CheckDBsInSyncUpTo(i, dafnyChainID(1), 1002)
		require.ErrorContains(t, err, "conjunct (2)")
		require.ErrorContains(t, err, "not in verifiedDB.Get(1001).l2Heads")
	})

	t.Run("conjunct 3: verified head not sealed in logsDB", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101)
		err := CheckDBsInSyncUpTo(i, dafnyChainID(1), 1002)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "no sealed block 101 for verified head at ts 1001")
	})

	t.Run("conjunct 4: sealed block diverges from verified head", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).seals[101] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 101, Timestamp: 1001,
		}
		err := CheckDBsInSyncUpTo(i, dafnyChainID(1), 1002)
		require.ErrorContains(t, err, "conjunct (4)")
		require.ErrorContains(t, err, "at ts 1001")
	})
}

func TestCheckDBsInSync(t *testing.T) {
	t.Parallel()

	t.Run("pass: empty verifiedDB and empty logsDB", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckDBsInSync(i, dafnyChainID(1)))
	})

	t.Run("pass: synced instance", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckDBsInSync(i, dafnyChainID(1)))
		require.NoError(t, CheckDBsInSync(i, dafnyChainID(2)))
	})

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.lastTimestamp = 999 // break the VerifiedDB cache conjunct
		err := CheckDBsInSync(i, dafnyChainID(1))
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
		require.ErrorContains(t, err, "VerifiedDB.dfy Valid()")
		require.NotContains(t, err.Error(), "DBsInSync conjunct")
	})

	t.Run("conjunct N1: empty verifiedDB but sealed logsDB", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[dafnyChainID(1)] = dafnySealedMock(dafnyHeadSeal(100, 1000))
		err := CheckDBsInSync(i, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (N1)")
		require.ErrorContains(t, err, "verifiedDB is empty")
	})

	t.Run("conjunct S1: last verified entry misses the chain", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		putVerified(t, i.verifiedDB, dafnyVerifiedResult(1002, map[uint64]uint64{2: 202}))
		err := CheckDBsInSync(i, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S1)")
	})

	t.Run("conjunct S2: latest sealed block beyond last verified head", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).latest = eth.BlockID{Hash: common.HexToHash("0xaa"), Number: 103}
		err := CheckDBsInSync(i, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S2)")
		require.ErrorContains(t, err, "!= last verified head")
		require.NotContains(t, err.Error(), "conjunct (S3)")
	})

	t.Run("conjunct S3: interior logsDB/verifiedDB divergence", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).seals[101] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 101, Timestamp: 1001,
		}
		err := CheckDBsInSync(i, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S3)")
		require.ErrorContains(t, err, "DBsInSyncUpTo conjunct (4)")
		require.NotContains(t, err.Error(), "conjunct (S2)")
	})
}

func TestCheckAllDBsInSyncUpTo(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckAllDBsInSyncUpTo(dafnySyncedInterop(t), 1002))
	})

	t.Run("conjunct 0: nil Interop", func(t *testing.T) {
		t.Parallel()
		err := CheckAllDBsInSyncUpTo(nil, 1002)
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("reports only the diverged chain", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 2).seals, 201)
		err := CheckAllDBsInSyncUpTo(i, 1002)
		require.ErrorContains(t, err, "chain 2:")
		require.ErrorContains(t, err, "conjunct (3)")
		require.NotContains(t, err.Error(), "chain 1:")
	})
}

func TestCheckAllDBsInSync(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckAllDBsInSync(dafnySyncedInterop(t)))
	})

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.lastTimestamp = 999
		err := CheckAllDBsInSync(i)
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("reports only the diverged chain", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 2).seals[201] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 201, Timestamp: 1001,
		}
		err := CheckAllDBsInSync(i)
		require.ErrorContains(t, err, "chain 2:")
		require.ErrorContains(t, err, "conjunct (S3)")
		require.NotContains(t, err.Error(), "chain 1:")
	})
}

func TestInteropDBSyncAsserts(t *testing.T) {
	t.Parallel()

	t.Run("pass leaves t untouched", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		ft := &fakeDafnyT{}
		AssertInteropValid(ft, i)
		AssertDBsInSyncUpTo(ft, i, dafnyChainID(1), 1002)
		AssertDBsInSync(ft, i, dafnyChainID(1))
		AssertAllDBsInSyncUpTo(ft, i, 1002)
		AssertAllDBsInSync(ft, i)
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	asserts := map[string]func(dafnyT){
		"AssertInteropValid":     func(ft dafnyT) { AssertInteropValid(ft, nil) },
		"AssertDBsInSyncUpTo":    func(ft dafnyT) { AssertDBsInSyncUpTo(ft, nil, dafnyChainID(1), 1002) },
		"AssertDBsInSync":        func(ft dafnyT) { AssertDBsInSync(ft, nil, dafnyChainID(1)) },
		"AssertAllDBsInSyncUpTo": func(ft dafnyT) { AssertAllDBsInSyncUpTo(ft, nil, 1002) },
		"AssertAllDBsInSync":     func(ft dafnyT) { AssertAllDBsInSync(ft, nil) },
	}
	for name, assert := range asserts {
		t.Run(name+" reports and fails", func(t *testing.T) {
			t.Parallel()
			ft := &fakeDafnyT{}
			assert(ft)
			require.True(t, ft.errorfCalled)
			require.True(t, ft.failNowCalled)
		})
	}
}
