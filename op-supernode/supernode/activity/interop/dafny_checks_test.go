package interop

// Tests for the Dafny model checkers (see dafny_checks.go).

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop/raftwallogdb"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"

	bolt "go.etcd.io/bbolt"
)

// === types ===

var _ dafnyT = (*testing.T)(nil)

// fakeDafnyT records Assert* failure interactions.
type fakeDafnyT struct {
	helperCalled  bool
	errorfCalled  bool
	failNowCalled bool
	lastMessage   string
}

func (f *fakeDafnyT) Helper() { f.helperCalled = true }

func (f *fakeDafnyT) Errorf(format string, args ...any) {
	f.errorfCalled = true
	f.lastMessage = format
}

func (f *fakeDafnyT) FailNow() { f.failNowCalled = true }

func dafnyChainID(n uint64) eth.ChainID {
	return eth.ChainIDFromUInt64(n)
}

// dafnyTestParams returns ModelParams with ACTIVATION_TIMESTAMP 1000 and
// CHAIN_IDS {1, 2}.
func dafnyTestParams() ModelParams {
	return ModelParams{
		ActivationTimestamp: 1000,
		ChainIDs: map[eth.ChainID]struct{}{
			dafnyChainID(1): {},
			dafnyChainID(2): {},
		},
	}
}

// dafnyTestHeads returns a block map keyed by the given chain IDs.
func dafnyTestHeads(chains ...uint64) map[eth.ChainID]eth.BlockID {
	heads := make(map[eth.ChainID]eth.BlockID, len(chains))
	for n, c := range chains {
		heads[dafnyChainID(c)] = eth.BlockID{Hash: common.HexToHash("0xaa"), Number: uint64(100 + n)}
	}
	return heads
}

func ptrUint64(v uint64) *uint64 { return &v }

func TestModelParamsFromInterop(t *testing.T) {
	t.Parallel()

	t.Run("uninitialized falls back to protocol activation timestamp", func(t *testing.T) {
		t.Parallel()
		i := &Interop{
			activationTimestamp: 1000,
			chains: map[eth.ChainID]cc.InteropChain{
				dafnyChainID(1): nil,
				dafnyChainID(2): nil,
			},
		}
		p := modelParamsFromInterop(i)
		require.Equal(t, uint64(1000), p.ActivationTimestamp)
		require.Equal(t, map[eth.ChainID]struct{}{
			dafnyChainID(1): {},
			dafnyChainID(2): {},
		}, p.ChainIDs)
	})

	t.Run("initialized uses first verifiable timestamp", func(t *testing.T) {
		t.Parallel()
		i := &Interop{
			activationTimestamp:        1000,
			verificationStartTimestamp: 1234,
		}
		i.initialized.Store(true)
		p := modelParamsFromInterop(i)
		require.Equal(t, uint64(1234), p.ActivationTimestamp)
		require.Empty(t, p.ChainIDs)
	})
}

func TestCheckValidRewindPlan(t *testing.T) {
	t.Parallel()

	p := dafnyTestParams()
	tests := []struct {
		name    string
		plan    RewindPlan
		wantErr string // empty: predicate holds
	}{
		{
			name: "pass: None at activation timestamp",
			plan: RewindPlan{RewindAtOrAfter: 1000},
		},
		{
			name: "pass: None below activation timestamp",
			plan: RewindPlan{RewindAtOrAfter: 500},
		},
		{
			name: "pass: Some with matching reset target and full head coverage",
			plan: RewindPlan{
				RewindAtOrAfter:  1005,
				ResetAllChainsTo: ptrUint64(1004),
				TargetHeads:      dafnyTestHeads(1, 2),
			},
		},
		{
			name:    "violation N1: None above activation timestamp",
			plan:    RewindPlan{RewindAtOrAfter: 1001},
			wantErr: "conjunct (N1)",
		},
		{
			name: "violation S1: Some at or below activation timestamp",
			plan: RewindPlan{
				RewindAtOrAfter:  1000,
				ResetAllChainsTo: ptrUint64(999),
				TargetHeads:      dafnyTestHeads(1, 2),
			},
			wantErr: "conjunct (S1)",
		},
		{
			name: "violation S1 without underflow: Some with rewindAtOrAfter zero",
			plan: RewindPlan{
				RewindAtOrAfter:  0,
				ResetAllChainsTo: ptrUint64(0),
				TargetHeads:      dafnyTestHeads(1, 2),
			},
			wantErr: "conjunct (S1)",
		},
		{
			name: "violation S2: reset target is not rewindAtOrAfter-1",
			plan: RewindPlan{
				RewindAtOrAfter:  1005,
				ResetAllChainsTo: ptrUint64(1000),
				TargetHeads:      dafnyTestHeads(1, 2),
			},
			wantErr: "conjunct (S2)",
		},
		{
			name: "violation S3: target heads missing a chain",
			plan: RewindPlan{
				RewindAtOrAfter:  1005,
				ResetAllChainsTo: ptrUint64(1004),
				TargetHeads:      dafnyTestHeads(1),
			},
			wantErr: "conjunct (S3)",
		},
		{
			name: "violation S3: target heads contain an unknown chain",
			plan: RewindPlan{
				RewindAtOrAfter:  1005,
				ResetAllChainsTo: ptrUint64(1004),
				TargetHeads:      dafnyTestHeads(1, 2, 3),
			},
			wantErr: "conjunct (S3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckValidRewindPlan(p, tt.plan)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
			require.ErrorContains(t, err, "Types.dfy ValidRewindPlan")
		})
	}
}

func TestCheckValidPendingTransition(t *testing.T) {
	t.Parallel()

	p := dafnyTestParams()
	validResult := &Result{Timestamp: 1001, L2Heads: dafnyTestHeads(1, 2)}
	validPlan := &RewindPlan{RewindAtOrAfter: 1000}
	tests := []struct {
		name    string
		pending PendingTransition
		wantErr string
	}{
		{
			name:    "pass: Advance with full result",
			pending: PendingTransition{Decision: DecisionAdvance, Result: validResult},
		},
		{
			name:    "pass: Invalidate with full result",
			pending: PendingTransition{Decision: DecisionInvalidate, Result: validResult},
		},
		{
			name:    "pass: Rewind with valid plan",
			pending: PendingTransition{Decision: DecisionRewind, Rewind: validPlan},
		},
		{
			name:    "violation 0: decision outside the model",
			pending: PendingTransition{Decision: Decision(42)},
			wantErr: "conjunct (0)",
		},
		{
			name:    "violation 1: decision Wait",
			pending: PendingTransition{Decision: DecisionWait},
			wantErr: "conjunct (1)",
		},
		{
			name:    "violation 2: Rewind without plan",
			pending: PendingTransition{Decision: DecisionRewind},
			wantErr: "conjunct (2)",
		},
		{
			name: "violation 3: Rewind with invalid plan",
			pending: PendingTransition{
				Decision: DecisionRewind,
				Rewind:   &RewindPlan{RewindAtOrAfter: 1001},
			},
			wantErr: "conjunct (3)",
		},
		{
			name:    "violation 4: Advance without result",
			pending: PendingTransition{Decision: DecisionAdvance},
			wantErr: "conjunct (4)",
		},
		{
			name:    "violation 5: Invalidate without result",
			pending: PendingTransition{Decision: DecisionInvalidate},
			wantErr: "conjunct (5)",
		},
		{
			name: "violation 6: result heads missing a chain",
			pending: PendingTransition{
				Decision: DecisionAdvance,
				Result:   &Result{Timestamp: 1001, L2Heads: dafnyTestHeads(1)},
			},
			wantErr: "conjunct (6)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckValidPendingTransition(p, tt.pending)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
			require.ErrorContains(t, err, "Types.dfy ValidPendingTransition")
		})
	}
}

func TestCheckValidPendingTransitionJoinsViolations(t *testing.T) {
	t.Parallel()

	// Rewind without plan, with a result that misses a chain: conjuncts (2)
	// and (6) must both be reported.
	err := CheckValidPendingTransition(dafnyTestParams(), PendingTransition{
		Decision: DecisionRewind,
		Result:   &Result{Timestamp: 1001, L2Heads: dafnyTestHeads(1)},
	})
	require.ErrorContains(t, err, "conjunct (2)")
	require.ErrorContains(t, err, "conjunct (6)")
}

func TestCheckValidStepOutput(t *testing.T) {
	t.Parallel()

	p := dafnyTestParams()
	heads := dafnyTestHeads(1, 2)
	obs := RoundObservation{
		LastVerifiedTS: ptrUint64(1000),
		NextTimestamp:  1001,
		ChainsReady:    true,
		BlocksAtTS:     heads,
	}
	tests := []struct {
		name    string
		output  StepOutput
		obs     RoundObservation
		wantErr string
	}{
		{
			name:   "pass: WaitOutput",
			output: StepOutput{Decision: DecisionWait},
			obs:    RoundObservation{},
		},
		{
			name:   "pass: RewindOutput with last verified timestamp",
			output: StepOutput{Decision: DecisionRewind},
			obs:    obs,
		},
		{
			name:   "pass: AdvanceOutput matching observation",
			output: StepOutput{Decision: DecisionAdvance, Result: Result{Timestamp: 1001, L2Heads: heads}},
			obs:    obs,
		},
		{
			name:   "pass: InvalidateOutput matching observation",
			output: StepOutput{Decision: DecisionInvalidate, Result: Result{Timestamp: 1001, L2Heads: heads}},
			obs:    obs,
		},
		{
			name:    "violation 0: decision outside the model",
			output:  StepOutput{Decision: Decision(42)},
			obs:     obs,
			wantErr: "conjunct (0)",
		},
		{
			name:    "violation R1: RewindOutput without last verified timestamp",
			output:  StepOutput{Decision: DecisionRewind},
			obs:     RoundObservation{NextTimestamp: 1001},
			wantErr: "conjunct (R1)",
		},
		{
			name:    "violation 1: Advance result timestamp mismatch",
			output:  StepOutput{Decision: DecisionAdvance, Result: Result{Timestamp: 999, L2Heads: heads}},
			obs:     obs,
			wantErr: "conjunct (1)",
		},
		{
			name: "violation 2: Advance heads differ from observed blocks",
			output: StepOutput{Decision: DecisionAdvance, Result: Result{
				Timestamp: 1001,
				L2Heads: map[eth.ChainID]eth.BlockID{
					dafnyChainID(1): {Hash: common.HexToHash("0xbb"), Number: 100},
					dafnyChainID(2): heads[dafnyChainID(2)],
				},
			}},
			obs:     obs,
			wantErr: "conjunct (2)",
		},
		{
			name: "violation 3: Invalidate heads equal observed blocks but miss a chain",
			output: StepOutput{Decision: DecisionInvalidate, Result: Result{
				Timestamp: 1001,
				L2Heads:   dafnyTestHeads(1),
			}},
			obs: RoundObservation{
				LastVerifiedTS: ptrUint64(1000),
				NextTimestamp:  1001,
				ChainsReady:    true,
				BlocksAtTS:     dafnyTestHeads(1),
			},
			wantErr: "conjunct (3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckValidStepOutput(p, tt.output, tt.obs)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
			require.ErrorContains(t, err, "Types.dfy ValidStepOutput")
		})
	}
}

func TestCheckValidRoundObservation(t *testing.T) {
	t.Parallel()

	p := dafnyTestParams()
	tests := []struct {
		name    string
		obs     RoundObservation
		wantErr string
	}{
		{
			name: "pass: zero observation",
			obs:  RoundObservation{},
		},
		{
			name: "pass: rewind-bound observation with last verified timestamp",
			obs: RoundObservation{
				LastVerifiedTS: ptrUint64(1000),
				ChainsReady:    true,
				BlocksAtTS:     dafnyTestHeads(1, 2),
				L1NeedsRewind:  true,
			},
		},
		{
			name: "pass: paused observation skips model conjuncts",
			obs: RoundObservation{
				Paused:        true,
				ChainsReady:   true,
				L1NeedsRewind: true,
			},
		},
		{
			name:    "violation 1: L1 needs rewind without last verified timestamp",
			obs:     RoundObservation{L1NeedsRewind: true},
			wantErr: "conjunct (1)",
		},
		{
			name: "violation 2: chains ready with incomplete block map",
			obs: RoundObservation{
				ChainsReady: true,
				BlocksAtTS:  dafnyTestHeads(1),
			},
			wantErr: "conjunct (2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckValidRoundObservation(p, tt.obs)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
			require.ErrorContains(t, err, "Types.dfy ValidRoundObservation")
		})
	}
}

func TestAssertWrappers(t *testing.T) {
	t.Parallel()

	p := dafnyTestParams()

	t.Run("pass leaves t untouched", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertValidRewindPlan(ft, p, RewindPlan{RewindAtOrAfter: 1000})
		AssertValidPendingTransition(ft, p, PendingTransition{
			Decision: DecisionAdvance,
			Result:   &Result{Timestamp: 1001, L2Heads: dafnyTestHeads(1, 2)},
		})
		AssertValidStepOutput(ft, p, StepOutput{Decision: DecisionWait}, RoundObservation{})
		AssertValidRoundObservation(ft, p, RoundObservation{})
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	t.Run("violation reports and fails", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertValidRewindPlan(ft, p, RewindPlan{RewindAtOrAfter: 1001})
		require.True(t, ft.errorfCalled)
		require.True(t, ft.failNowCalled)
	})
}

// === verifieddb ===

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

// === logsdb ===

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

// === round ===

// dafnySyncedObs is the observation of dafnySyncedInterop right before
// verification: last verified 1002, next 1003, frontier heads advancing by
// one block, L1 consistent.
func dafnySyncedObs() RoundObservation {
	return RoundObservation{
		LastVerifiedTS: ptrUint64(1002),
		NextTimestamp:  1003,
		ChainsReady:    true,
		BlocksAtTS:     dafnyHeads(map[uint64]uint64{1: 103, 2: 203}),
		L1Consistent:   true,
	}
}

// dafnyRewindObs is the observation of dafnySyncedInterop after the accepted
// L1 inclusion turned non-canonical (model !l1Consistent, Go L1NeedsRewind).
func dafnyRewindObs() RoundObservation {
	obs := dafnySyncedObs()
	obs.L1Consistent = false
	obs.L1NeedsRewind = true
	return obs
}

// dafnyAdvanceOutput is the advancing step output matching dafnySyncedObs.
func dafnyAdvanceOutput() StepOutput {
	return StepOutput{
		Decision: DecisionAdvance,
		Result:   Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
	}
}

func TestCheckOutputConsistentWithVerified(t *testing.T) {
	t.Parallel()

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.concrete().lastTimestamp = 999
		err := CheckOutputConsistentWithVerified(i, StepOutput{Decision: DecisionWait}, dafnySyncedObs())
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("requires: invalid step output short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = nil
		err := CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, obs)
		require.ErrorContains(t, err, "requires ValidStepOutput")
	})

	t.Run("pass: wait output is vacuous", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionWait}, dafnySyncedObs()))
	})

	t.Run("pass: rewind with lastVerifiedTS - 1 present", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, dafnyRewindObs()))
	})

	t.Run("pass: rewind at activation timestamp skips the guarded conjunct", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, i.verifiedDB.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = ptrUint64(1000)
		require.NoError(t, CheckOutputConsistentWithVerified(i, StepOutput{Decision: DecisionRewind}, obs))
	})

	t.Run("conjunct R1: rewind with lastVerifiedTS - 1 absent", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = ptrUint64(1005)
		err := CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, obs)
		require.ErrorContains(t, err, "conjunct (R1)")
		require.ErrorContains(t, err, "1004 not in verifiedDB")
	})

	t.Run("pass: advancing output", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			dafnyAdvanceOutput(), dafnySyncedObs()))
	})

	t.Run("conjunct A1: advance result does not advance the db", func(t *testing.T) {
		t.Parallel()
		output := dafnyAdvanceOutput()
		output.Result.Timestamp = 1005
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1005
		err := CheckOutputConsistentWithVerified(dafnySyncedInterop(t), output, obs)
		require.ErrorContains(t, err, "conjunct (A1)")
	})

	t.Run("pass: invalidate at the next timestamp", func(t *testing.T) {
		t.Parallel()
		output := dafnyAdvanceOutput()
		output.Decision = DecisionInvalidate
		require.NoError(t, CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			output, dafnySyncedObs()))
	})

	t.Run("conjunct I1: invalidate not at the next timestamp", func(t *testing.T) {
		t.Parallel()
		output := dafnyAdvanceOutput()
		output.Decision = DecisionInvalidate
		output.Result.Timestamp = 1005
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1005
		err := CheckOutputConsistentWithVerified(dafnySyncedInterop(t), output, obs)
		require.ErrorContains(t, err, "conjunct (I1)")
		require.ErrorContains(t, err, "!= NextTimestamp() 1003")
	})
}

func TestCheckOutputConsistentWithLogs(t *testing.T) {
	t.Parallel()

	t.Run("requires: Valid() failure short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckOutputConsistentWithLogs(i, StepOutput{Decision: DecisionWait}, dafnySyncedObs())
		require.ErrorContains(t, err, "requires Valid()")
	})

	t.Run("requires: invalid step output short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = nil
		err := CheckOutputConsistentWithLogs(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, obs)
		require.ErrorContains(t, err, "requires ValidStepOutput")
	})

	t.Run("requires: OutputConsistentWithVerified failure short-circuits", func(t *testing.T) {
		t.Parallel()
		output := dafnyAdvanceOutput()
		output.Result.Timestamp = 1005
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1005
		err := CheckOutputConsistentWithLogs(dafnySyncedInterop(t), output, obs)
		require.ErrorContains(t, err, "requires OutputConsistentWithVerified")
	})

	t.Run("pass: wait and invalidate outputs are vacuous", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckOutputConsistentWithLogs(i,
			StepOutput{Decision: DecisionWait}, dafnySyncedObs()))
		output := dafnyAdvanceOutput()
		output.Decision = DecisionInvalidate
		require.NoError(t, CheckOutputConsistentWithLogs(i, output, dafnySyncedObs()))
	})

	t.Run("pass: rewind with target heads sealed on all chains", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithLogs(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, dafnyRewindObs()))
	})

	t.Run("conjunct R1: rewind target head not sealed", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101)
		err := CheckOutputConsistentWithLogs(i, StepOutput{Decision: DecisionRewind}, dafnyRewindObs())
		require.ErrorContains(t, err, "conjunct (R1)")
		require.ErrorContains(t, err, "chain 1")
		require.NotContains(t, err.Error(), "chain 2")
	})

	t.Run("conjunct R2: sealed block differs from verified head", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).seals[101] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 101, Timestamp: 1001,
		}
		err := CheckOutputConsistentWithLogs(i, StepOutput{Decision: DecisionRewind}, dafnyRewindObs())
		require.ErrorContains(t, err, "conjunct (R2)")
	})

	t.Run("pass: advancing output", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithLogs(dafnySyncedInterop(t),
			dafnyAdvanceOutput(), dafnySyncedObs()))
	})

	t.Run("conjunct A1: advance result does not advance the logsDBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).latest = dafnyBlock(105)
		err := CheckOutputConsistentWithLogs(i, dafnyAdvanceOutput(), dafnySyncedObs())
		require.ErrorContains(t, err, "conjunct (A1)")
		require.ErrorContains(t, err, "chain 1")
	})
}

func TestCheckObservationConsistentWithVerified(t *testing.T) {
	t.Parallel()

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.concrete().lastTimestamp = 999
		err := CheckObservationConsistentWithVerified(i, dafnySyncedObs())
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("requires: invalid round observation short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = nil
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "requires ValidRoundObservation")
	})

	t.Run("pass: synced observation", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckObservationConsistentWithVerified(dafnySyncedInterop(t), dafnySyncedObs()))
	})

	t.Run("pass: empty db observation at activation", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.initialized.Store(true)
		i.verificationStartTimestamp = 1000
		obs := RoundObservation{NextTimestamp: 1000}
		require.NoError(t, CheckObservationConsistentWithVerified(i, obs))
	})

	t.Run("conjunct 1: lastVerifiedTS does not mirror the db", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)

		obs := dafnySyncedObs()
		obs.LastVerifiedTS = ptrUint64(1001)
		err := CheckObservationConsistentWithVerified(i, obs)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "1001 != verifiedDB.lastTimestamp 1002")

		obs.LastVerifiedTS = nil
		err = CheckObservationConsistentWithVerified(i, obs)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "obs.lastVerifiedTS is None")
	})

	t.Run("conjunct 2: nextTimestamp does not mirror the db", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1004
		obs.ChainsReady = false // keep conjunct (4) vacuous
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "conjunct (2)")
		require.ErrorContains(t, err, "!= NextTimestamp() 1003")
	})

	t.Run("pass: rewind observation with lastVerifiedTS - 1 present", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckObservationConsistentWithVerified(dafnySyncedInterop(t), dafnyRewindObs()))
	})

	t.Run("pass: rewind observation at activation timestamp skips the guarded conjunct", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, i.verifiedDB.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = ptrUint64(1000)
		obs.NextTimestamp = 1001
		require.NoError(t, CheckObservationConsistentWithVerified(i, obs))
	})

	t.Run("conjunct 3: rewind observation with lastVerifiedTS - 1 absent", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = ptrUint64(1005) // also violates conjunct (1); 1004 is not committed
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "1004 not in verifiedDB")
	})

	t.Run("conjunct 4: ready frontier does not advance the db", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.BlocksAtTS = dafnyHeads(map[uint64]uint64{1: 105, 2: 203})
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "conjunct (4)")
	})

	t.Run("paused: round-flag conjuncts skipped", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.Paused = true
		obs.BlocksAtTS = dafnyHeads(map[uint64]uint64{1: 105, 2: 203}) // would violate (4)
		require.NoError(t, CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs))
	})

	t.Run("paused: db-mirror conjuncts still checked", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.Paused = true
		obs.NextTimestamp = 999
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "conjunct (2)")
	})
}

func TestCheckObservationConsistentWithLogs(t *testing.T) {
	t.Parallel()

	t.Run("requires: Valid() failure short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckObservationConsistentWithLogs(i, dafnySyncedObs())
		require.ErrorContains(t, err, "requires Valid()")
	})

	t.Run("requires: invalid round observation short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = nil
		err := CheckObservationConsistentWithLogs(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "requires ValidRoundObservation")
	})

	t.Run("requires: ObservationConsistentWithVerified failure short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1004
		err := CheckObservationConsistentWithLogs(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "requires ObservationConsistentWithVerified")
	})

	t.Run("pass: synced advancing observation", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckObservationConsistentWithLogs(dafnySyncedInterop(t), dafnySyncedObs()))
	})

	t.Run("pass: rewind observation with target heads sealed", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckObservationConsistentWithLogs(dafnySyncedInterop(t), dafnyRewindObs()))
	})

	t.Run("conjunct 1: rewind target head not sealed", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 2).seals, 201)
		err := CheckObservationConsistentWithLogs(i, dafnyRewindObs())
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "chain 2")
		require.NotContains(t, err.Error(), "chain 1")
	})

	t.Run("conjunct 2: sealed block differs from verified head", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 2).seals[201] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 201, Timestamp: 1001,
		}
		err := CheckObservationConsistentWithLogs(i, dafnyRewindObs())
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 3: ready frontier does not advance the logsDBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 2).latest = dafnyBlock(205)
		err := CheckObservationConsistentWithLogs(i, dafnySyncedObs())
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "chain 2")
	})

	t.Run("paused: all conjuncts skipped", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101) // would violate (1) if not paused
		obs := dafnyRewindObs()
		obs.Paused = true
		require.NoError(t, CheckObservationConsistentWithLogs(i, obs))
	})
}

func TestRoundAsserts(t *testing.T) {
	t.Parallel()

	t.Run("pass leaves t untouched", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		ft := &fakeDafnyT{}
		AssertOutputConsistentWithVerified(ft, i, dafnyAdvanceOutput(), dafnySyncedObs())
		AssertOutputConsistentWithLogs(ft, i, dafnyAdvanceOutput(), dafnySyncedObs())
		AssertObservationConsistentWithVerified(ft, i, dafnySyncedObs())
		AssertObservationConsistentWithLogs(ft, i, dafnySyncedObs())
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	asserts := map[string]func(dafnyT){
		"AssertOutputConsistentWithVerified": func(ft dafnyT) {
			AssertOutputConsistentWithVerified(ft, nil, StepOutput{}, RoundObservation{})
		},
		"AssertOutputConsistentWithLogs": func(ft dafnyT) {
			AssertOutputConsistentWithLogs(ft, nil, StepOutput{}, RoundObservation{})
		},
		"AssertObservationConsistentWithVerified": func(ft dafnyT) {
			AssertObservationConsistentWithVerified(ft, nil, RoundObservation{})
		},
		"AssertObservationConsistentWithLogs": func(ft dafnyT) {
			AssertObservationConsistentWithLogs(ft, nil, RoundObservation{})
		},
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

// === transition ===

// dafnyBlock builds a BlockID matching a dafnyVerifiedResult head at number.
func dafnyBlock(number uint64) eth.BlockID {
	return eth.BlockID{Hash: common.HexToHash("0xaa"), Number: number}
}

// dafnyHeads builds a chainID->head map from chainID->number pairs, with
// dafnyBlock hashes.
func dafnyHeads(numbers map[uint64]uint64) map[eth.ChainID]eth.BlockID {
	heads := make(map[eth.ChainID]eth.BlockID, len(numbers))
	for c, n := range numbers {
		heads[dafnyChainID(c)] = dafnyBlock(n)
	}
	return heads
}

// dafnySyncedPlan is a rewind plan consistent with dafnySyncedInterop:
// rewind at 1002, reset to 1001, target heads = verified heads at 1001.
func dafnySyncedPlan() RewindPlan {
	return RewindPlan{
		RewindAtOrAfter:  1002,
		ResetAllChainsTo: ptrUint64(1001),
		TargetHeads:      dafnyHeads(map[uint64]uint64{1: 101, 2: 201}),
	}
}

// dafnySyncedAdvance is a pending Advance transition consistent with
// dafnySyncedInterop (next ts 1003, heads advancing by one block).
func dafnySyncedAdvance() PendingTransition {
	return PendingTransition{
		Decision: DecisionAdvance,
		Result: &Result{
			Timestamp: 1003,
			L2Heads:   dafnyHeads(map[uint64]uint64{1: 103, 2: 203}),
		},
	}
}

// dafnyRewoundInterop builds the state right after applyRewindPlan: verified
// results at 1000..1001, logsDBs sealed to the 1001 heads, and the rewind
// pending transition still stored. Returns the instance and the stored plan.
func dafnyRewoundInterop(t *testing.T) (*Interop, RewindPlan) {
	t.Helper()
	i := dafnyTestInterop(t)
	for off := uint64(0); off <= 1; off++ {
		require.NoError(t, i.verifiedDB.Commit(
			dafnyVerifiedResult(1000+off, map[uint64]uint64{1: 100 + off, 2: 200 + off})))
	}
	i.logsDBs[dafnyChainID(1)] = dafnySealedMock(dafnyHeadSeal(100, 1000), dafnyHeadSeal(101, 1001))
	i.logsDBs[dafnyChainID(2)] = dafnySealedMock(dafnyHeadSeal(200, 1000), dafnyHeadSeal(201, 1001))
	plan := RewindPlan{
		RewindAtOrAfter:  1002,
		ResetAllChainsTo: ptrUint64(1001),
		TargetHeads:      dafnyHeads(map[uint64]uint64{1: 101, 2: 201}),
	}
	require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
		Decision: DecisionRewind,
		Rewind:   &plan,
	}))
	return i, plan
}

func TestCheckAdvancesVerifiedDB(t *testing.T) {
	t.Parallel()

	t.Run("pass: empty db at activation timestamp", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckAdvancesVerifiedDB(dafnyTestInterop(t), 1000, nil))
	})

	t.Run("pass: next timestamp with heads advancing by at most one", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 202})))
	})

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.concrete().lastTimestamp = 999
		err := CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 203}))
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("conjunct N1: empty db but ts not at activation", func(t *testing.T) {
		t.Parallel()
		err := CheckAdvancesVerifiedDB(dafnyTestInterop(t), 1001, nil)
		require.ErrorContains(t, err, "conjunct (N1)")
	})

	t.Run("conjunct S1: ts not the successor of the last timestamp", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesVerifiedDB(i, 1005, dafnyHeads(map[uint64]uint64{1: 103, 2: 203}))
		require.ErrorContains(t, err, "conjunct (S1)")
		require.NotContains(t, err.Error(), "conjunct (S2)")
	})

	t.Run("conjunct S2: blocks key set differs from last heads", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 103}))
		require.ErrorContains(t, err, "conjunct (S2)")
	})

	t.Run("conjunct S3: head number jumps by more than one", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 105, 2: 203}))
		require.ErrorContains(t, err, "conjunct (S3)")
		require.ErrorContains(t, err, "not in [102, 103]")
	})

	t.Run("conjunct S3: head number decreases", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 101, 2: 203}))
		require.ErrorContains(t, err, "conjunct (S3)")
	})
}

func TestCheckAdvancesLogsDB(t *testing.T) {
	t.Parallel()

	t.Run("pass: empty logsDB at activation timestamp", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckAdvancesLogsDB(dafnyTestInterop(t), 1000, dafnyChainID(1), dafnyBlock(100)))
	})

	t.Run("pass: next block and repeated tip block", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), dafnyBlock(103)))
		require.NoError(t, CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), dafnyBlock(102)))
	})

	t.Run("conjunct 0: unknown chain", func(t *testing.T) {
		t.Parallel()
		err := CheckAdvancesLogsDB(dafnySyncedInterop(t), 1003, dafnyChainID(9), dafnyBlock(103))
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "has no logsDB")
	})

	t.Run("conjunct N1: empty logsDB but ts not at activation", func(t *testing.T) {
		t.Parallel()
		err := CheckAdvancesLogsDB(dafnyTestInterop(t), 1234, dafnyChainID(1), dafnyBlock(100))
		require.ErrorContains(t, err, "conjunct (N1)")
	})

	t.Run("conjunct S1: block number out of range", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), dafnyBlock(104))
		require.ErrorContains(t, err, "conjunct (S1)")
		err = CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), dafnyBlock(101))
		require.ErrorContains(t, err, "conjunct (S1)")
	})

	t.Run("conjunct S2: same-number block differs from latest", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), eth.BlockID{Hash: common.Hash{0xff}, Number: 102})
		require.ErrorContains(t, err, "conjunct (S2)")
		require.NotContains(t, err.Error(), "conjunct (S1)")
	})
}

func TestCheckAdvancesAllLogsDBs(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckAdvancesAllLogsDBs(i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 203})))
	})

	t.Run("conjunct 0: key set mismatch", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesAllLogsDBs(i, 1003, dafnyHeads(map[uint64]uint64{1: 103}))
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "requires blocksAtTS.Keys == logsDBs.Keys")
	})

	t.Run("reports only the failing chain", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesAllLogsDBs(i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 205}))
		require.ErrorContains(t, err, "chain 2:")
		require.ErrorContains(t, err, "conjunct (S1)")
		require.NotContains(t, err.Error(), "chain 1:")
	})
}

func TestCheckPlanConsistentWithVerified(t *testing.T) {
	t.Parallel()

	t.Run("pass: None plan is vacuous", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckPlanConsistentWithVerified(i, RewindPlan{RewindAtOrAfter: 1000}))
	})

	t.Run("pass: consistent Some plan", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckPlanConsistentWithVerified(dafnySyncedInterop(t), dafnySyncedPlan()))
	})

	t.Run("conjunct S1: reset target not in db", func(t *testing.T) {
		t.Parallel()
		plan := RewindPlan{
			RewindAtOrAfter:  1005,
			ResetAllChainsTo: ptrUint64(1004),
			TargetHeads:      dafnyHeads(map[uint64]uint64{1: 101, 2: 201}),
		}
		err := CheckPlanConsistentWithVerified(dafnySyncedInterop(t), plan)
		require.ErrorContains(t, err, "conjunct (S1)")
		require.NotContains(t, err.Error(), "conjunct (S3)")
	})

	t.Run("conjunct S2: surviving timestamp above reset target", func(t *testing.T) {
		t.Parallel()
		plan := RewindPlan{
			RewindAtOrAfter:  1002,
			ResetAllChainsTo: ptrUint64(1000),
			TargetHeads:      dafnyHeads(map[uint64]uint64{1: 100, 2: 200}),
		}
		err := CheckPlanConsistentWithVerified(dafnySyncedInterop(t), plan)
		require.ErrorContains(t, err, "conjunct (S2)")
		require.ErrorContains(t, err, "committed timestamp 1001")
	})

	t.Run("conjunct S3: target heads diverge from verified result", func(t *testing.T) {
		t.Parallel()
		plan := dafnySyncedPlan()
		plan.TargetHeads = dafnyHeads(map[uint64]uint64{1: 999, 2: 201})
		err := CheckPlanConsistentWithVerified(dafnySyncedInterop(t), plan)
		require.ErrorContains(t, err, "conjunct (S3)")
	})
}

func TestCheckPlanConsistentWithLogs(t *testing.T) {
	t.Parallel()

	t.Run("pass: None plan is always consistent", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckPlanConsistentWithLogs(i, RewindPlan{RewindAtOrAfter: 1000}, dafnyChainID(1)))
	})

	t.Run("pass: target heads sealed on both chains", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckPlanConsistentWithLogs(i, dafnySyncedPlan(), dafnyChainID(1)))
		require.NoError(t, CheckPlanConsistentWithLogs(i, dafnySyncedPlan(), dafnyChainID(2)))
	})

	t.Run("conjunct 0: unknown chain", func(t *testing.T) {
		t.Parallel()
		err := CheckPlanConsistentWithLogs(dafnySyncedInterop(t), dafnySyncedPlan(), dafnyChainID(9))
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 0: chain missing from target heads", func(t *testing.T) {
		t.Parallel()
		plan := dafnySyncedPlan()
		delete(plan.TargetHeads, dafnyChainID(1))
		err := CheckPlanConsistentWithLogs(dafnySyncedInterop(t), plan, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "requires chain")
	})

	t.Run("conjunct S1: target head not sealed", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101)
		err := CheckPlanConsistentWithLogs(i, dafnySyncedPlan(), dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S1)")
	})

	t.Run("conjunct S2: sealed block differs from target head", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).seals[101] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 101, Timestamp: 1001,
		}
		err := CheckPlanConsistentWithLogs(i, dafnySyncedPlan(), dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S2)")
	})
}

func TestCheckRewoundVerifiedDB(t *testing.T) {
	t.Parallel()

	t.Run("pass: rewound state with stored plan", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		require.NoError(t, CheckRewoundVerifiedDB(i, plan))
	})

	t.Run("pass: full rewind on empty db", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		plan := RewindPlan{RewindAtOrAfter: 1000}
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		require.NoError(t, CheckRewoundVerifiedDB(i, plan))
	})

	t.Run("requires: inconsistent plan short-circuits", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		plan.ResetAllChainsTo = ptrUint64(1005)
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "requires PlanConsistentWithVerified")
	})

	t.Run("conjunct 1: no pending transition", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		require.NoError(t, i.verifiedDB.ClearPendingTransition())
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 2: stored decision is not Rewind", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1002, L2Heads: dafnyHeads(map[uint64]uint64{1: 102, 2: 202})},
		}))
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 3: stored plan differs from given plan", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		stored := plan
		stored.RewindAtOrAfter = 1003
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &stored,
		}))
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (3)")
	})

	t.Run("conjunct N1: db not empty after full rewind", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, i.verifiedDB.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		plan := RewindPlan{RewindAtOrAfter: 1000}
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (N1)")
	})

	t.Run("conjunct S1: last timestamp beyond reset target", func(t *testing.T) {
		t.Parallel()
		i, _ := dafnyRewoundInterop(t)
		plan := RewindPlan{
			RewindAtOrAfter:  1001,
			ResetAllChainsTo: ptrUint64(1000),
			TargetHeads:      dafnyHeads(map[uint64]uint64{1: 100, 2: 200}),
		}
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (S1)")
		require.ErrorContains(t, err, "LastTimestamp() 1001 != resetAllChainsTo 1000")
	})
}

func TestCheckRewoundLogsDB(t *testing.T) {
	t.Parallel()

	t.Run("pass: rewound state on both chains", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		require.NoError(t, CheckRewoundLogsDB(i, plan, dafnyChainID(1)))
		require.NoError(t, CheckRewoundLogsDB(i, plan, dafnyChainID(2)))
	})

	t.Run("pass: cleared logsDB after full rewind", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckRewoundLogsDB(i, RewindPlan{RewindAtOrAfter: 1000}, dafnyChainID(1)))
	})

	t.Run("requires: plan inconsistent with logs short-circuits", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101)
		err := CheckRewoundLogsDB(i, plan, dafnyChainID(1))
		require.ErrorContains(t, err, "requires PlanConsistentWithLogs")
	})

	t.Run("conjunct 0: chain missing from target heads", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		delete(plan.TargetHeads, dafnyChainID(2))
		err := CheckRewoundLogsDB(i, plan, dafnyChainID(2))
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct N1: logsDB not cleared after full rewind", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[dafnyChainID(1)] = dafnySealedMock(dafnyHeadSeal(100, 1000))
		err := CheckRewoundLogsDB(i, RewindPlan{RewindAtOrAfter: 1000}, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (N1)")
	})

	t.Run("conjunct S1: latest sealed block beyond target head", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		mockLogsDBFor(t, i, 1).latest = dafnyBlock(102)
		err := CheckRewoundLogsDB(i, plan, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S1)")
	})
}

func TestCheckTransitionConsistentWithVerified(t *testing.T) {
	t.Parallel()

	t.Run("requires: invalid pending transition short-circuits", func(t *testing.T) {
		t.Parallel()
		err := CheckTransitionConsistentWithVerified(dafnySyncedInterop(t),
			PendingTransition{Decision: DecisionWait})
		require.ErrorContains(t, err, "requires ValidPendingTransition")
	})

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.concrete().lastTimestamp = 999
		err := CheckTransitionConsistentWithVerified(i, dafnySyncedAdvance())
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("pass: consistent rewind, invalidate, and advance", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		plan := dafnySyncedPlan()
		require.NoError(t, CheckTransitionConsistentWithVerified(i,
			PendingTransition{Decision: DecisionRewind, Rewind: &plan}))
		require.NoError(t, CheckTransitionConsistentWithVerified(i, PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
		}))
		require.NoError(t, CheckTransitionConsistentWithVerified(i, dafnySyncedAdvance()))
	})

	t.Run("conjunct R1: rewind plan inconsistent with verifiedDB", func(t *testing.T) {
		t.Parallel()
		plan := RewindPlan{
			RewindAtOrAfter:  1005,
			ResetAllChainsTo: ptrUint64(1004),
			TargetHeads:      dafnyHeads(map[uint64]uint64{1: 101, 2: 201}),
		}
		err := CheckTransitionConsistentWithVerified(dafnySyncedInterop(t),
			PendingTransition{Decision: DecisionRewind, Rewind: &plan})
		require.ErrorContains(t, err, "conjunct (R1)")
	})

	t.Run("conjunct A1: advance result does not advance the db", func(t *testing.T) {
		t.Parallel()
		pending := dafnySyncedAdvance()
		pending.Result.Timestamp = 1005
		err := CheckTransitionConsistentWithVerified(dafnySyncedInterop(t), pending)
		require.ErrorContains(t, err, "conjunct (A1)")
	})
}

func TestCheckTransitionConsistentWithLogs(t *testing.T) {
	t.Parallel()

	t.Run("requires: invalid pending transition short-circuits", func(t *testing.T) {
		t.Parallel()
		err := CheckTransitionConsistentWithLogs(dafnySyncedInterop(t),
			PendingTransition{Decision: DecisionWait})
		require.ErrorContains(t, err, "requires ValidPendingTransition")
	})

	t.Run("pass: consistent rewind, invalidate, and advance", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		plan := dafnySyncedPlan()
		require.NoError(t, CheckTransitionConsistentWithLogs(i,
			PendingTransition{Decision: DecisionRewind, Rewind: &plan}))
		require.NoError(t, CheckTransitionConsistentWithLogs(i, PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
		}))
		require.NoError(t, CheckTransitionConsistentWithLogs(i, dafnySyncedAdvance()))
	})

	t.Run("conjunct R1: logsDB chain missing from target heads", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.logsDBs[dafnyChainID(3)] = dafnySealedMock(dafnyHeadSeal(300, 1000))
		plan := dafnySyncedPlan()
		err := CheckTransitionConsistentWithLogs(i,
			PendingTransition{Decision: DecisionRewind, Rewind: &plan})
		require.ErrorContains(t, err, "conjunct (R1)")
		require.ErrorContains(t, err, "chain 3 not in plan.targetHeads")
	})

	t.Run("conjunct R2: rewind plan inconsistent with one chain's logs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).seals[101] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 101, Timestamp: 1001,
		}
		plan := dafnySyncedPlan()
		err := CheckTransitionConsistentWithLogs(i,
			PendingTransition{Decision: DecisionRewind, Rewind: &plan})
		require.ErrorContains(t, err, "conjunct (R2)")
		require.ErrorContains(t, err, "chain 1")
		require.NotContains(t, err.Error(), "chain 2")
	})

	t.Run("conjunct A1: result heads key set differs from logsDBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckTransitionConsistentWithLogs(i, dafnySyncedAdvance())
		require.ErrorContains(t, err, "conjunct (A1)")
		require.NotContains(t, err.Error(), "conjunct (A2)")
	})

	t.Run("conjunct A2: advance result does not advance the logsDBs", func(t *testing.T) {
		t.Parallel()
		pending := dafnySyncedAdvance()
		pending.Result.L2Heads = dafnyHeads(map[uint64]uint64{1: 105, 2: 203})
		err := CheckTransitionConsistentWithLogs(dafnySyncedInterop(t), pending)
		require.ErrorContains(t, err, "conjunct (A2)")
		require.ErrorContains(t, err, "chain 1")
	})
}

func TestCheckPendingTransitionIsConsistent(t *testing.T) {
	t.Parallel()

	t.Run("requires: Valid() failure short-circuits", func(t *testing.T) {
		t.Parallel()
		err := CheckPendingTransitionIsConsistent(nil)
		require.ErrorContains(t, err, "requires Valid()")
	})

	t.Run("pass: no pending transition with synced DBs", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckPendingTransitionIsConsistent(dafnySyncedInterop(t)))
	})

	t.Run("conjunct N1: no pending transition with diverged DBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 2).seals[201] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 201, Timestamp: 1001,
		}
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (N1)")
		require.ErrorContains(t, err, "chain 2")
	})

	t.Run("pass: stored advance, invalidate, and rewind transitions", func(t *testing.T) {
		t.Parallel()
		advance := dafnySyncedInterop(t)
		require.NoError(t, advance.verifiedDB.SetPendingTransition(dafnySyncedAdvance()))
		require.NoError(t, CheckPendingTransitionIsConsistent(advance))

		invalidate := dafnySyncedInterop(t)
		require.NoError(t, invalidate.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
		}))
		require.NoError(t, CheckPendingTransitionIsConsistent(invalidate))

		rewind := dafnySyncedInterop(t)
		plan := dafnySyncedPlan()
		require.NoError(t, rewind.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		require.NoError(t, CheckPendingTransitionIsConsistent(rewind))
	})

	t.Run("conjunct S1: stored advance inconsistent with verifiedDB", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		pending := dafnySyncedAdvance()
		pending.Result.Timestamp = 1005
		require.NoError(t, i.verifiedDB.SetPendingTransition(pending))
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (S1)")
		require.NotContains(t, err.Error(), "conjunct (S2)")
	})

	t.Run("conjunct S2: stored advance inconsistent with logsDBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		pending := dafnySyncedAdvance()
		pending.Result.L2Heads = map[eth.ChainID]eth.BlockID{
			dafnyChainID(1): {Hash: common.Hash{0xbb}, Number: 102},
			dafnyChainID(2): dafnyBlock(203),
		}
		require.NoError(t, i.verifiedDB.SetPendingTransition(pending))
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (S2)")
		require.NotContains(t, err.Error(), "conjunct (S1)")
	})

	t.Run("conjunct S3: stored rewind with desynced prefix", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		plan := dafnySyncedPlan()
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		delete(mockLogsDBFor(t, i, 1).seals, 100)
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (S3)")
		require.NotContains(t, err.Error(), "conjunct (S1)")
		require.NotContains(t, err.Error(), "conjunct (S2)")
	})

	t.Run("conjunct S3: stored invalidate with desynced tip", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
		}))
		mockLogsDBFor(t, i, 2).latest = dafnyBlock(205)
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (S3)")
	})
}

func TestCheckInvariants(t *testing.T) {
	t.Parallel()

	t.Run("pass: fresh and synced instances", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckInvariants(dafnyTestInterop(t)))
		require.NoError(t, CheckInvariants(dafnySyncedInterop(t)))
	})

	t.Run("conjunct 1: Valid() violation", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckInvariants(i)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "Interop.dfy Valid()")
		require.NotContains(t, err.Error(), "PendingTransitionIsConsistent")
	})

	t.Run("conjunct 2: stored pending transition inconsistent", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		pending := dafnySyncedAdvance()
		pending.Result.Timestamp = 1005
		require.NoError(t, i.verifiedDB.SetPendingTransition(pending))
		err := CheckInvariants(i)
		require.ErrorContains(t, err, "conjunct (2)")
		require.ErrorContains(t, err, "PendingTransitionIsConsistent")
	})
}

func TestAssertInvariants(t *testing.T) {
	t.Parallel()

	t.Run("healthy instance passes on a real t", func(t *testing.T) {
		t.Parallel()
		AssertInvariants(t, dafnySyncedInterop(t))
	})

	t.Run("healthy instance leaves t untouched", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertInvariants(ft, dafnySyncedInterop(t))
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	t.Run("stored inconsistent pending transition fails t", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		pending := dafnySyncedAdvance()
		pending.Result.Timestamp = 1005
		require.NoError(t, i.verifiedDB.SetPendingTransition(pending))
		ft := &fakeDafnyT{}
		AssertInvariants(ft, i)
		require.True(t, ft.errorfCalled)
		require.True(t, ft.failNowCalled)
	})
}

func TestTransitionAsserts(t *testing.T) {
	t.Parallel()

	t.Run("pass leaves t untouched", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		r, plan := dafnyRewoundInterop(t)
		ft := &fakeDafnyT{}
		AssertAdvancesVerifiedDB(ft, i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 203}))
		AssertAdvancesLogsDB(ft, i, 1003, dafnyChainID(1), dafnyBlock(103))
		AssertAdvancesAllLogsDBs(ft, i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 203}))
		AssertPlanConsistentWithVerified(ft, i, dafnySyncedPlan())
		AssertPlanConsistentWithLogs(ft, i, dafnySyncedPlan(), dafnyChainID(1))
		AssertRewoundVerifiedDB(ft, r, plan)
		AssertRewoundLogsDB(ft, r, plan, dafnyChainID(1))
		AssertTransitionConsistentWithVerified(ft, i, dafnySyncedAdvance())
		AssertTransitionConsistentWithLogs(ft, i, dafnySyncedAdvance())
		AssertPendingTransitionIsConsistent(ft, i)
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	asserts := map[string]func(dafnyT){
		"AssertAdvancesVerifiedDB": func(ft dafnyT) { AssertAdvancesVerifiedDB(ft, nil, 1000, nil) },
		"AssertAdvancesLogsDB":     func(ft dafnyT) { AssertAdvancesLogsDB(ft, nil, 1000, dafnyChainID(1), dafnyBlock(100)) },
		"AssertAdvancesAllLogsDBs": func(ft dafnyT) { AssertAdvancesAllLogsDBs(ft, nil, 1000, nil) },
		"AssertPlanConsistentWithVerified": func(ft dafnyT) {
			AssertPlanConsistentWithVerified(ft, nil, RewindPlan{})
		},
		"AssertPlanConsistentWithLogs": func(ft dafnyT) {
			AssertPlanConsistentWithLogs(ft, nil, RewindPlan{}, dafnyChainID(1))
		},
		"AssertRewoundVerifiedDB": func(ft dafnyT) { AssertRewoundVerifiedDB(ft, nil, RewindPlan{}) },
		"AssertRewoundLogsDB":     func(ft dafnyT) { AssertRewoundLogsDB(ft, nil, RewindPlan{}, dafnyChainID(1)) },
		"AssertTransitionConsistentWithVerified": func(ft dafnyT) {
			AssertTransitionConsistentWithVerified(ft, nil, PendingTransition{})
		},
		"AssertTransitionConsistentWithLogs": func(ft dafnyT) {
			AssertTransitionConsistentWithLogs(ft, nil, PendingTransition{})
		},
		"AssertPendingTransitionIsConsistent": func(ft dafnyT) { AssertPendingTransitionIsConsistent(ft, nil) },
		"AssertInvariants":                    func(ft dafnyT) { AssertInvariants(ft, nil) },
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

// === interop ===

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
		putVerified(t, i.verifiedDB.concrete(), dafnyVerifiedResult(1002, map[uint64]uint64{1: 102, 2: 202}))
		i.verifiedDB.concrete().lastTimestamp = 1002 // keep the VerifiedDB cache conjunct quiet
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
		i.verifiedDB.concrete().firstTimestamp = 999 // stale cache: ACTIVATION_TIMESTAMP maps to 999
		err := CheckInteropValid(i)
		require.ErrorContains(t, err, "conjunct (5)")
		require.ErrorContains(t, err, "ACTIVATION_TIMESTAMP 999 not in db")
		require.NotContains(t, err.Error(), "conjunct (6)")
	})

	t.Run("conjunct 6: committed timestamp below activation", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.concrete().firstTimestamp = 1001 // stale cache: ACTIVATION_TIMESTAMP maps to 1001
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
		putVerified(t, i.verifiedDB.concrete(), dafnyVerifiedResult(1001, map[uint64]uint64{2: 201}))
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
		i.verifiedDB.concrete().lastTimestamp = 999 // break the VerifiedDB cache conjunct
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
		putVerified(t, i.verifiedDB.concrete(), dafnyVerifiedResult(1002, map[uint64]uint64{2: 202}))
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
		i.verifiedDB.concrete().lastTimestamp = 999
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
