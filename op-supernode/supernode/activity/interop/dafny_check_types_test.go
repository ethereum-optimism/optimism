package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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
