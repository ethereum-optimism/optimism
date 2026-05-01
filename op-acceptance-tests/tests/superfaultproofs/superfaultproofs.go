package superfaultproofs

import (
	"context"
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-program/client/interop"
	interopTypes "github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	stepsPerTimestamp = super.StepsPerTimestamp
	consolidateStep   = stepsPerTimestamp - 1
)

// chain bundles the DSL handles for one L2 chain, ordered by chain ID.
type chain struct {
	ID      eth.ChainID
	Cfg     *rollup.Config
	Rollup  apis.RollupClient
	EL      *dsl.L2ELNode
	CLNode  *dsl.L2CLNode
	Batcher *dsl.L2Batcher
}

// freezeChains takes exclusive control of L2 block production: stops every
// chain's real sequencer and batcher, and waits for both safety levels to
// stall. After this returns no real-sequencer or batcher activity advances
// the chain — block production is the caller's via TestSequencer.SequenceBlock.
func freezeChains(chains []*chain) {
	for _, c := range chains {
		c.CLNode.StopSequencer()
	}
	for _, c := range chains {
		c.Batcher.Stop()
	}
	for _, c := range chains {
		c.CLNode.WaitForStall(types.LocalUnsafe)
		c.CLNode.WaitForStall(types.LocalSafe)
	}
}

// advanceUnsafeToTimestamp drives every chain's UnsafeL2 head to its latest
// scheduled block at-or-before target via repeated TestSequencer.SequenceBlock
// calls. Chains whose schedule produces a block exactly at target advance to
// target; chains whose schedule does not produce a block at target are left at
// the previous scheduled block (target - blockTime + offset). Errors if any
// chain's head is already past target.
//
// Callers must ensure the real sequencer is stopped on every chain before
// invoking this — otherwise SequenceBlock can race with the real sequencer.
func advanceUnsafeToTimestamp(t devtest.T, sys *presets.SimpleInterop, chains []*chain, target uint64) {
	for _, c := range chains {
		for {
			head := c.EL.BlockRefByLabel(eth.Unsafe)
			t.Require().LessOrEqualf(head.Time, target,
				"chain %s head %d is already past target %d", c.ID, head.Time, target)
			nextTime := head.Time + c.Cfg.BlockTime
			if nextTime > target {
				break
			}
			sys.TestSequencer.SequenceBlock(t, c.ID, head.Hash)
			next := c.EL.BlockRefByLabel(eth.Unsafe)
			t.Require().Equalf(nextTime, next.Time,
				"chain %s SequenceBlock must produce a block at parent.Time+blockTime", c.ID)
			t.Require().Equalf(head.Number+1, next.Number,
				"chain %s SequenceBlock must produce exactly one new block", c.ID)
		}
	}
}

// advanceSafeToCurrentUnsafe starts c.Batcher, waits for c.LocalSafe to catch
// up to c.UnsafeL2, then stops the batcher. The batcher only ever submits up
// to current UnsafeL2; with sequencers stopped, that bounds what lands on L1.
//
// To advance safe to a specific timestamp, pin UnsafeL2 to that timestamp via
// advanceUnsafeToTimestamp first.
func advanceSafeToCurrentUnsafe(t devtest.T, c *chain) {
	target := c.EL.BlockRefByLabel(eth.Unsafe).Number
	c.Batcher.Start()
	c.CLNode.Reached(types.LocalSafe, target, 60)
	c.Batcher.Stop()
}

// blockExpectedAt reports whether c's rollup schedule produces a block exactly
// at ts. Equivalent to: would the trace provider, transitioning to ts, demand
// batch data for c?
func blockExpectedAt(c *chain, ts uint64) bool {
	g := c.Cfg.Genesis.L2Time
	if ts < g {
		return false
	}
	return (ts-g)%c.Cfg.BlockTime == 0
}

// extendChainsExpectingBoundaryBlock produces a block at boundary on every
// chain whose rollup schedule expects one there, then batches it so the
// supernode picks it up via LocalSafe. l1HeadCurrent was already captured
// before this runs, so the boundary batch lands on L1 *after* l1HeadCurrent —
// the FPP can't derive the boundary block via l1HeadCurrent, and the
// supernode's response for the chain at boundary has RequiredL1 >
// l1HeadCurrent so the challenger top half also returns InvalidTransition.
//
// We need the supernode to actually have the boundary block in its
// OptimisticAtTimestamp data: that lookup goes through LocalSafe state, so an
// UnsafeL2-only block is invisible to it. Batching the new block fixes that
// without making l1HeadCurrent see it (l1HeadCurrent is fixed earlier in the
// flow).
func extendChainsExpectingBoundaryBlock(t devtest.T, sys *presets.SimpleInterop, chains []*chain, boundary uint64) {
	extended := make([]*chain, 0, len(chains))
	for _, c := range chains {
		if !blockExpectedAt(c, boundary) {
			continue
		}
		head := c.EL.BlockRefByLabel(eth.Unsafe)
		t.Require().Equalf(boundary-c.Cfg.BlockTime, head.Time,
			"chain %s head %d must be one block-time before boundary %d", c.ID, head.Time, boundary)
		sys.TestSequencer.SequenceBlock(t, c.ID, head.Hash)
		next := c.EL.BlockRefByLabel(eth.Unsafe)
		t.Require().Equalf(boundary, next.Time,
			"chain %s extension must land at boundary %d", c.ID, boundary)
		extended = append(extended, c)
	}
	// Batch the boundary blocks so the supernode picks them up via LocalSafe.
	// Each chain's batcher submits only its current UnsafeL2 (= boundary block).
	for _, c := range extended {
		advanceSafeToCurrentUnsafe(t, c)
	}
}

// transitionTest describes a single super-root transition test case.
type transitionTest struct {
	Name               string
	AgreedClaim        []byte
	DisputedClaim      []byte
	DisputedTraceIndex int64
	L1Head             eth.BlockID
	ClaimTimestamp     uint64
	ExpectValid        bool
}

// orderedChains returns the two interop chains sorted by chain ID.
func orderedChains(sys *presets.SimpleInterop) []*chain {
	chains := []*chain{
		{ID: sys.L2ChainA.ChainID(), Cfg: sys.L2ChainA.Escape().RollupConfig(), Rollup: sys.L2CLA.Escape().RollupAPI(), EL: sys.L2ELA, CLNode: sys.L2CLA, Batcher: sys.L2BatcherA},
		{ID: sys.L2ChainB.ChainID(), Cfg: sys.L2ChainB.Escape().RollupConfig(), Rollup: sys.L2CLB.Escape().RollupAPI(), EL: sys.L2ELB, CLNode: sys.L2CLB, Batcher: sys.L2BatcherB},
	}
	slices.SortFunc(chains, func(a, b *chain) int { return a.ID.Cmp(b.ID) })
	return chains
}

// nextTimestampAfterSafeHeads returns the next block timestamp after all chains' safe heads.
func nextTimestampAfterSafeHeads(t devtest.T, chains []*chain) uint64 {
	var ts uint64
	for _, c := range chains {
		status, err := c.Rollup.SyncStatus(t.Ctx())
		t.Require().NoError(err)
		// Use LocalSafeL2 when available, as it reflects the latest L1-derived
		// head before interop cross-validation. SafeL2 (cross-safe) may lag far
		// behind, causing endTimestamp to target blocks whose batch data is
		// already on L1.
		safeNum := status.SafeL2.Number
		if status.LocalSafeL2.Number > safeNum {
			safeNum = status.LocalSafeL2.Number
		}
		next := c.Cfg.TimestampForBlock(safeNum + 1)
		if next > ts {
			ts = next
		}
	}
	t.Require().NotZero(ts, "end timestamp must be non-zero")

	// Advance ts until every chain produces a new block at ts compared to ts-1.
	// With varied block times (e.g. 1s and 2s), the initial ts may land on a
	// no-op boundary for the slower chain. The L1-head-constrained subtests
	// assume every chain has a real transition to validate, so we need all
	// chains to have TargetBlockNumber(ts) > TargetBlockNumber(ts-1).
	for {
		allNew := true
		for _, c := range chains {
			curr, err := c.Cfg.TargetBlockNumber(ts)
			t.Require().NoError(err)
			prev, err := c.Cfg.TargetBlockNumber(ts - 1)
			t.Require().NoError(err)
			if curr <= prev {
				allNew = false
				break
			}
		}
		if allNew {
			break
		}
		ts++
	}
	return ts
}

// superRootAtTimestamp constructs a SuperV1 from each chain's output at the given timestamp.
func superRootAtTimestamp(t devtest.T, chains []*chain, timestamp uint64) eth.SuperV1 {
	sr := eth.SuperV1{Timestamp: timestamp, Chains: make([]eth.ChainIDAndOutput, len(chains))}
	for i, c := range chains {
		blockNum, err := c.Cfg.TargetBlockNumber(timestamp)
		t.Require().NoError(err)
		out, err := c.Rollup.OutputAtBlock(t.Ctx(), blockNum)
		t.Require().NoError(err)
		sr.Chains[i] = eth.ChainIDAndOutput{ChainID: c.ID, Output: out.OutputRoot}
	}
	return sr
}

// optimisticBlockAtTimestamp returns the optimistic block for a single chain at the given timestamp.
// It queries the supernode's super_atTimestamp API which returns the true optimistic output root,
// even after an invalid block has been replaced during cross-safe validation.
func optimisticBlockAtTimestamp(t devtest.T, queryAPI apis.SupernodeQueryAPI, chainID eth.ChainID, timestamp uint64) interopTypes.OptimisticBlock {
	resp, err := queryAPI.SuperRootAtTimestamp(t.Ctx(), timestamp)
	t.Require().NoError(err)
	out, ok := resp.OptimisticAtTimestamp[chainID]
	t.Require().Truef(ok, "no optimistic output for chain %v at timestamp %d", chainID, timestamp)
	return interopTypes.OptimisticBlock{BlockHash: out.Output.BlockHash, OutputRoot: out.OutputRoot}
}

// marshalTransition serializes a transition state with the given super root, step, and progress.
func marshalTransition(superRoot eth.SuperV1, step uint64, progress ...interopTypes.OptimisticBlock) []byte {
	return (&interopTypes.TransitionState{
		SuperRoot:       superRoot.Marshal(),
		PendingProgress: progress,
		Step:            step,
	}).Marshal()
}

// latestRequiredL1 returns the latest RequiredL1 across all optimistic outputs,
// i.e. the earliest L1 block at which all chains' data is derivable.
func latestRequiredL1(resp eth.SuperRootAtTimestampResponse) eth.BlockID {
	var latest eth.BlockID
	for _, out := range resp.OptimisticAtTimestamp {
		if out.RequiredL1.Number > latest.Number {
			latest = out.RequiredL1
		}
	}
	return latest
}

// runKonaInteropProgram runs the kona interop fault proof program and checks the result.
func runKonaInteropProgram(t devtest.T, cfg vm.Config, l1Head common.Hash, agreedPreState []byte, l2Claim common.Hash, claimTimestamp uint64, expectValid bool) {
	tmpDir := t.TempDir()
	inputs := utils.LocalGameInputs{
		L1Head:           l1Head,
		AgreedPreState:   agreedPreState,
		L2Claim:          l2Claim,
		L2SequenceNumber: new(big.Int).SetUint64(claimTimestamp),
	}

	argv, err := vm.NewNativeKonaSuperExecutor().OracleCommand(cfg, tmpDir, inputs)
	t.Require().NoError(err)

	exePath, err := filepath.Abs(argv[0])
	t.Require().NoError(err)
	t.Logf("Executing kona interop program: %s", strings.Join(argv, " "))
	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, exePath, argv[1:]...)
	cmd.Dir = tmpDir
	cmd.Env = append(append(cmd.Env, os.Environ()...), "NO_COLOR=1")
	// WaitDelay bounds how long CombinedOutput waits for I/O pipes to close
	// after the process exits or the context is cancelled. Without this, if
	// the context timeout fires and the process is killed, CombinedOutput
	// can block indefinitely waiting for pipe EOF (e.g. if a child process
	// or unclosed descriptor keeps the pipe open).
	cmd.WaitDelay = 60 * time.Second

	out, runErr := cmd.CombinedOutput()
	if expectValid {
		t.Require().NoErrorf(runErr, "kona interop program failed:\n%s", string(out))
		return
	}
	var exitErr *exec.ExitError
	t.Require().ErrorAsf(runErr, &exitErr, "expected kona interop program to fail, got: %v\n%s", runErr, string(out))
	t.Require().Equalf(1, exitErr.ExitCode(), "expected exit code 1 for invalid claim, got %d:\n%s", exitErr.ExitCode(), string(out))
}

// runChallengerProviderTest verifies the challenger trace provider agrees with the test expectations.
func runChallengerProviderTest(t devtest.T, queryAPI apis.SupernodeQueryAPI, gameDepth challengerTypes.Depth, startTimestamp, claimTimestamp uint64, test *transitionTest) {
	prestateProvider := super.NewSuperNodePrestateProvider(queryAPI, startTimestamp)
	traceProvider := super.NewSuperNodeTraceProvider(
		t.Logger().New("role", "challenger-provider"),
		prestateProvider,
		queryAPI,
		test.L1Head,
		gameDepth,
		startTimestamp,
		claimTimestamp,
	)

	var agreedPrestate []byte
	var err error
	if test.DisputedTraceIndex > 0 {
		agreedPrestate, err = traceProvider.GetPreimageBytes(t.Ctx(), challengerTypes.NewPosition(gameDepth, big.NewInt(test.DisputedTraceIndex-1)))
		t.Require().NoError(err)
	} else {
		superRoot, err := traceProvider.AbsolutePreState(t.Ctx())
		t.Require().NoError(err)
		agreedPrestate = superRoot.Marshal()
	}
	t.Require().Equal(test.AgreedClaim, agreedPrestate, "agreed prestate mismatch")

	disputedClaim, err := traceProvider.GetPreimageBytes(t.Ctx(), challengerTypes.NewPosition(gameDepth, big.NewInt(test.DisputedTraceIndex)))
	t.Require().NoError(err)
	if test.ExpectValid {
		t.Require().Equal(test.DisputedClaim, disputedClaim, "valid claim mismatch")
	} else {
		t.Require().NotEqual(test.DisputedClaim, disputedClaim, "invalid claim unexpectedly matched challenger provider output")
	}
}

// buildTransitionTests constructs the standard set of super-root transition test cases.
func buildTransitionTests(
	start, end eth.SuperV1,
	step1, step2 []byte,
	padding func(uint64) []byte,
	l1HeadCurrent, l1HeadBefore, l1HeadAfterFirst eth.BlockID,
	endTimestamp uint64,
) []*transitionTest {
	return []*transitionTest{
		{
			Name:               "ClaimDirectToNextTimestamp",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "FirstChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "FirstChainOptimisticBlock-InvalidNoChange",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      start.Marshal(),
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "SecondChainOptimisticBlock",
			AgreedClaim:        step1,
			DisputedClaim:      step2,
			DisputedTraceIndex: 1,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SecondChainOptimisticBlock-InvalidNoChange",
			AgreedClaim:        step1,
			DisputedClaim:      step1,
			DisputedTraceIndex: 1,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "FirstPaddingStep",
			AgreedClaim:        step2,
			DisputedClaim:      padding(3),
			DisputedTraceIndex: 2,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "FirstPaddingStep-InvalidNoChange",
			AgreedClaim:        step2,
			DisputedClaim:      step2,
			DisputedTraceIndex: 2,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "SecondPaddingStep",
			AgreedClaim:        padding(3),
			DisputedClaim:      padding(4),
			DisputedTraceIndex: 3,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SecondPaddingStep-InvalidNoChange",
			AgreedClaim:        padding(3),
			DisputedClaim:      padding(3),
			DisputedTraceIndex: 3,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "LastPaddingStep",
			AgreedClaim:        padding(consolidateStep - 1),
			DisputedClaim:      padding(consolidateStep),
			DisputedTraceIndex: consolidateStep - 1,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "FirstChainReachesL1Head",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      super.InvalidTransition,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadBefore,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SecondChainReachesL1Head",
			AgreedClaim:        step1,
			DisputedClaim:      super.InvalidTransition,
			DisputedTraceIndex: 1,
			L1Head:             l1HeadAfterFirst,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SuperRootInvalidIfUnsupportedByL1Data",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadBefore,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "FromInvalidTransitionHash",
			AgreedClaim:        super.InvalidTransition,
			DisputedClaim:      super.InvalidTransition,
			DisputedTraceIndex: 2,
			L1Head:             l1HeadBefore,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "ConsolidateStep",
			AgreedClaim:        padding(consolidateStep),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "ConsolidateStep-InvalidNoChange",
			AgreedClaim:        padding(consolidateStep),
			DisputedClaim:      padding(consolidateStep),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
	}
}

// buildAfterChainHeadTests constructs the 6 subtests that probe trace-provider
// behavior at trace indices around and beyond the chain head boundary
// (= endTimestamp + 1).
//
// Per-config branching is keyed on whether each chain's rollup schedule
// produces a block at the boundary (blockExpectedAt). Stage 2b extends the
// UnsafeL2 head of any chain whose schedule expects a boundary block (without
// batching), so the supernode response at the boundary for that chain has
// RequiredL1 > l1HeadCurrent. Three configurations are exercised by callers:
//
//   - Uniform 2s/2s with even endTimestamp: neither chain expects a boundary
//     block. Tests 1 / 2 / 3 all assert valid TransitionState (rollover); test 4
//     asserts a valid agreed (SuperRoot at boundary) with disputed = InvalidTransition
//     because chain A's expected block at boundary+1 isn't derivable.
//
//   - 1s/2s (chain A=1s): chain A's schedule expects a boundary block. Its
//     optimistic step at boundary returns InvalidTransition (block expected,
//     batch missing on L1). Once one step is invalid every subsequent step is
//     too, so test 2 is omitted here — including it would just duplicate the
//     cascade rule tests 5 and 6 already cover.
//
//   - 2s/1s (chain B=1s): chain A=2s rolls over at boundary (test 1 returns a
//     valid TransitionState); chain B's optimistic step at boundary returns
//     InvalidTransition (block expected on chain B, batch missing on L1).
//     Test 2 is omitted here too for symmetry with the 1s/2s config —
//     including it would be a unique probe of "chain B becomes invalid first"
//     but we don't currently cover that.
//
// Test 4's agreed claim is the trace state at idx 2*sPT-1 (consolidation at
// boundary). Whenever any chain expects a boundary block the trace cascades
// to InvalidTransition by then; otherwise it consolidates to SuperRoot(boundary).
//
// Tests 5 & 6 exercise the cascade rule (Invalid → Invalid) at deeper indices,
// independent of how the trace got invalid in the first place.
//
// claimTimestamp = endTimestamp + 100 ensures the trace covers idx
// 4*stepsPerTimestamp + 1 (test 6).
func buildAfterChainHeadTests(
	chains []*chain,
	end, endNext eth.SuperV1,
	endTimestamp uint64,
	l1HeadCurrent eth.BlockID,
	firstOptNext, secondOptNext interopTypes.OptimisticBlock,
) []*transitionTest {
	boundary := endTimestamp + 1
	const claimBuffer = uint64(100)
	claimTimestamp := endTimestamp + claimBuffer

	chainAExpects := blockExpectedAt(chains[0], boundary)
	chainBExpects := blockExpectedAt(chains[1], boundary)
	anyExpects := chainAExpects || chainBExpects

	tests := []*transitionTest{}

	// Test 1 — chain A's optimistic step at boundary.
	// If chain A's schedule expects a block at the boundary but l1HeadCurrent
	// doesn't have one, the trace provider returns InvalidTransition.
	// Otherwise it rolls over with chain A's previous-block data.
	test1Disputed := marshalTransition(end, 1, firstOptNext)
	if chainAExpects {
		test1Disputed = interop.InvalidTransition
	}
	tests = append(tests, &transitionTest{
		Name:               "DisputeTimestampAfterChainHeadChainA",
		AgreedClaim:        end.Marshal(),
		DisputedClaim:      test1Disputed,
		L1Head:             l1HeadCurrent,
		ClaimTimestamp:     claimTimestamp,
		DisputedTraceIndex: consolidateStep + 1,
		ExpectValid:        true,
	})

	// Test 2 — chain B's optimistic step at boundary.
	// Skipped whenever any chain expects a boundary block: in 1s/2s the trace
	// has already cascaded to InvalidTransition (test 5/6 cover the cascade),
	// and in 2s/1s we omit for symmetry rather than add new coverage here.
	if !anyExpects {
		tests = append(tests, &transitionTest{
			Name:               "DisputeTimestampAfterChainHeadChainB",
			AgreedClaim:        marshalTransition(end, 1, firstOptNext),
			DisputedClaim:      marshalTransition(end, 2, firstOptNext, secondOptNext),
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     claimTimestamp,
			DisputedTraceIndex: consolidateStep + 2,
			ExpectValid:        true,
		})
	}

	// Test 3 — Consolidate at boundary.
	// In uniform configs the trace at idx 2*sPT-2 builds a clean TransitionState
	// and idx 2*sPT-1 consolidates to SuperRoot(boundary).
	// In varied configs the trace cascades to InvalidTransition starting at
	// the optimistic step where the faster chain's boundary block isn't on
	// l1HeadCurrent, so by idx 2*sPT-1 the cascade rule says "always Invalid".
	test3Agreed := marshalTransition(end, consolidateStep, firstOptNext, secondOptNext)
	test3Disputed := endNext.Marshal()
	if anyExpects {
		test3Agreed = interop.InvalidTransition
		test3Disputed = interop.InvalidTransition
	}
	tests = append(tests, &transitionTest{
		Name:               "DisputeTimestampAfterChainHeadConsolidate",
		AgreedClaim:        test3Agreed,
		DisputedClaim:      test3Disputed,
		L1Head:             l1HeadCurrent,
		ClaimTimestamp:     claimTimestamp,
		DisputedTraceIndex: 2*stepsPerTimestamp - 1,
		ExpectValid:        true,
	})

	// Test 4 — chain A's optimistic step at boundary+1.
	// In uniform: agreed at idx 2*sPT-1 is SuperRoot(boundary) (consolidate
	// step), disputed at idx 2*sPT is InvalidTransition because chain A=2s
	// expects a block at boundary+1 but its LocalSafe is at endTimestamp.
	// In varied: cascade — agreed and disputed both InvalidTransition.
	test4Agreed := endNext.Marshal()
	if anyExpects {
		test4Agreed = interop.InvalidTransition
	}
	tests = append(tests, &transitionTest{
		Name:               "DisputeBlockAfterChainHead-FirstChain",
		AgreedClaim:        test4Agreed,
		DisputedClaim:      interop.InvalidTransition,
		L1Head:             l1HeadCurrent,
		ClaimTimestamp:     claimTimestamp,
		DisputedTraceIndex: 2 * stepsPerTimestamp,
		ExpectValid:        true,
	})

	// Tests 5 & 6 — far past chain head. Cascade.
	tests = append(tests,
		&transitionTest{
			Name:               "AgreedBlockAfterChainHead-Consolidate",
			AgreedClaim:        interop.InvalidTransition,
			DisputedClaim:      interop.InvalidTransition,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     claimTimestamp,
			DisputedTraceIndex: 4*stepsPerTimestamp - 1,
			ExpectValid:        true,
		},
		&transitionTest{
			Name:               "AgreedBlockAfterChainHead-Optimistic",
			AgreedClaim:        interop.InvalidTransition,
			DisputedClaim:      interop.InvalidTransition,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     claimTimestamp,
			DisputedTraceIndex: 4*stepsPerTimestamp + 1,
			ExpectValid:        true,
		},
	)

	return tests
}

// RunTraceExtensionActivationTest verifies that trace extension correctly
// activates (or not) based on whether the claim timestamp has been reached.
func RunTraceExtensionActivationTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	endTimestamp := uint64(time.Now().Unix())
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp + 1)
	l1Head := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp + 1))

	startTimestamp := endTimestamp - 1
	agreedSuperRoot := superRootAtTimestamp(t, chains, endTimestamp)
	agreedClaim := agreedSuperRoot.Marshal()

	// The disputed claim transitions to the next timestamp by including the
	// first chain's optimistic block at endTimestamp+1.
	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp+1)
	disputedClaim := marshalTransition(agreedSuperRoot, 1, firstOptimistic)
	disputedTraceIndex := int64(stepsPerTimestamp)

	tests := []*transitionTest{
		{
			Name:               "CorrectlyDidNotActivate",
			AgreedClaim:        agreedClaim,
			DisputedClaim:      disputedClaim,
			DisputedTraceIndex: disputedTraceIndex,
			L1Head:             l1Head,
			// Trace extension does not activate because we have not reached the proposal timestamp yet.
			ClaimTimestamp: endTimestamp + 1,
			ExpectValid:    true,
		},
		{
			Name:               "IncorrectlyDidNotActivate",
			AgreedClaim:        agreedClaim,
			DisputedClaim:      disputedClaim,
			DisputedTraceIndex: disputedTraceIndex,
			L1Head:             l1Head,
			// Trace extension should have activated because we have reached the proposal timestamp.
			ClaimTimestamp: endTimestamp,
			ExpectValid:    false,
		},
		{
			Name:               "CorrectlyActivated",
			AgreedClaim:        agreedClaim,
			DisputedClaim:      agreedClaim,
			DisputedTraceIndex: disputedTraceIndex,
			L1Head:             l1Head,
			// Trace extension activated at the proposal timestamp, claim stays the same.
			ClaimTimestamp: endTimestamp,
			ExpectValid:    true,
		},
		{
			Name:               "IncorrectlyActivated",
			AgreedClaim:        agreedClaim,
			DisputedClaim:      agreedClaim,
			DisputedTraceIndex: disputedTraceIndex,
			L1Head:             l1Head,
			// Trace extension should not have activated because we haven't reached the proposal timestamp.
			ClaimTimestamp: endTimestamp + 1,
			ExpectValid:    false,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()

	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})
		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

// RunUnsafeProposalTest verifies that proposing an unsafe block (one without
// batch data on L1) is correctly identified as invalid.
func RunUnsafeProposalTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// Stop chains[0]'s batcher first so its safe head stalls while chains[1]'s
	// batcher continues to advance. This deterministically guarantees chains[0]
	// has the lowest safe head — which is required because:
	//  1. Step 0 in the super root trace transitions chains[0]. We need step 0
	//     to produce InvalidTransition (no batch data for chains[0]'s block).
	//  2. The agreed prestate at (endTimestamp - 1) must be verified for ALL
	//     chains. Using chains[0]'s stalled safe head as the anchor ensures
	//     that timestamp maps to a block at or below every chain's safe head.
	chains[0].Batcher.Stop()
	defer chains[0].Batcher.Start()
	chains[0].CLNode.WaitForStall(types.LocalSafe)

	stalledStatus, err := chains[0].Rollup.SyncStatus(t.Ctx())
	t.Require().NoError(err)
	stalledSafeHead := stalledStatus.SafeL2.Number

	// Wait for chains[1]'s safe head to surpass chains[0]'s stalled safe head.
	// chains[1]'s batcher is still running, so this is guaranteed to happen.
	// We need strictly greater so that chains[1]'s block at endTimestamp
	// (= TimestampForBlock(stalledSafeHead + 1)) is safe.
	t.Require().Eventually(func() bool {
		status1, err := chains[1].Rollup.SyncStatus(t.Ctx())
		return err == nil && status1.SafeL2.Number > stalledSafeHead
	}, 2*time.Minute, 2*time.Second, "chains[1] safe head should advance past chains[0]'s stalled safe head")

	chains[1].Batcher.Stop()
	defer chains[1].Batcher.Start()
	chains[1].CLNode.WaitForStall(types.LocalSafe)

	endTimestamp := chains[0].Cfg.TimestampForBlock(stalledSafeHead + 1)
	agreedTimestamp := endTimestamp - 1

	// Ensure chains[0] has produced the target block as unsafe.
	target, err := chains[0].Cfg.TargetBlockNumber(endTimestamp)
	t.Require().NoError(err)
	chains[0].EL.Reached(eth.Unsafe, target, 60)

	sys.SuperRoots.AwaitValidatedTimestamp(agreedTimestamp)
	resp := sys.SuperRoots.SuperRootAtTimestamp(agreedTimestamp)
	l1Head := resp.CurrentL1

	startTimestamp := agreedTimestamp
	agreedSuperRoot := superRootAtTimestamp(t, chains, agreedTimestamp)
	agreedClaim := agreedSuperRoot.Marshal()

	// Disputed claim: transition state with step 1 but no optimistic blocks.
	// This claims a transition happened, but since chains[0]'s block at
	// endTimestamp is only unsafe (no batch data on L1), the correct answer
	// is InvalidTransition.
	disputedClaim := marshalTransition(agreedSuperRoot, 1)

	tests := []*transitionTest{
		{
			Name:               "ProposedUnsafeBlock-NotValid",
			AgreedClaim:        agreedClaim,
			DisputedClaim:      disputedClaim,
			DisputedTraceIndex: 0,
			L1Head:             l1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "ProposedUnsafeBlock-ShouldBeInvalid",
			AgreedClaim:        agreedClaim,
			DisputedClaim:      super.InvalidTransition,
			DisputedTraceIndex: 0,
			L1Head:             l1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()

	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})
		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

// RunSuperFaultProofTest encapsulates the basic super fault proof test flow.
func RunSuperFaultProofTest(t devtest.T, sys *presets.SimpleInterop) {
	runFaultProofTest(t, sys)
}

// RunVariedBlockTimesTest verifies that the super fault proof system works
// correctly when chains have different block times (e.g. 1s and 2s), exercising
// edge cases where not every chain produces a new block at every timestamp.
//
// The system must be configured with varied block times before calling this
// function (e.g. via presets.WithL2BlockTimes).
func RunVariedBlockTimesTest(t devtest.T, sys *presets.SimpleInterop) {
	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// Verify chains have different block times.
	t.Require().NotEqual(chains[0].Cfg.BlockTime, chains[1].Cfg.BlockTime,
		"this test requires chains with different block times")

	runFaultProofTest(t, sys)
}

// runFaultProofTest is the shared body for RunSuperFaultProofTest and
// RunVariedBlockTimesTest. It takes exclusive control of L2 block production
// via TestSequencer and per-chain Batcher.Start/Stop, then exercises the
// proof game over a deterministically-built chain state.
func runFaultProofTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// Take exclusive control of L2 block production. After this returns the
	// real sequencer and batchers are stopped on every chain — only
	// TestSequencer.SequenceBlock advances UnsafeL2, only explicit
	// Batcher.Start/Stop advances LocalSafe.
	freezeChains(chains)

	// Produce blocks deterministically up to endTimestamp on every chain.
	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	startTimestamp := endTimestamp - 1
	// Batch each chain's startTimestamp block on L1 before capturing the L1
	// heads below, so the prev super root at startTimestamp is fully verified
	// at every captured head.
	advanceUnsafeToTimestamp(t, sys, chains, startTimestamp)
	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	advanceUnsafeToTimestamp(t, sys, chains, endTimestamp)

	// Capture the three L1 heads via batcher choreography. With sequencers
	// stopped, batchers can only ever submit through current UnsafeL2 (=
	// endTimestamp), so each captured head is bounded exactly.
	l1HeadBefore := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()
	advanceSafeToCurrentUnsafe(t, chains[0])
	l1HeadAfterFirst := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()
	advanceSafeToCurrentUnsafe(t, chains[1])
	l1HeadCurrent := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)

	// Per-chain UnsafeL2 alignment to the boundary. For each chain whose
	// rollup schedule produces a block at the boundary (endTimestamp + 1),
	// advance UnsafeL2 to boundary via TestSequencer and batch after
	// l1HeadCurrent. The boundary block's batch lands on L1 only after
	// l1HeadCurrent was captured, so the supernode response at the boundary
	// for that chain has RequiredL1 > l1HeadCurrent — which makes the
	// top-half (challenger) trace provider return InvalidTransition, matching
	// what the bottom-half (FPP) returns from L1 alone.
	//
	// For chains whose schedule does NOT produce a block at the boundary,
	// UnsafeL2 stays at endTimestamp; the supernode rolls over previous
	// block data and both halves return the same valid TransitionState.
	boundaryTimestamp := endTimestamp + 1
	extendChainsExpectingBoundaryBlock(t, sys, chains, boundaryTimestamp)

	// Build expected transition states.
	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)
	endNext := superRootAtTimestamp(t, chains, boundaryTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)
	firstOptimisticNext := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, boundaryTimestamp)
	secondOptimisticNext := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, boundaryTimestamp)

	step1 := marshalTransition(start, 1, firstOptimistic)
	step2 := marshalTransition(start, 2, firstOptimistic, secondOptimistic)
	padding := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	// Transition test cases.
	tests := buildTransitionTests(start, end, step1, step2, padding,
		l1HeadCurrent, l1HeadBefore, l1HeadAfterFirst, endTimestamp)
	tests = append(tests, buildAfterChainHeadTests(
		chains, end, endNext, endTimestamp, l1HeadCurrent,
		firstOptimisticNext, secondOptimisticNext)...)

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()

	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})
		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

// RunPreForkActivationTest verifies that super-root transitions produce
// correct results when the interop fork is scheduled but not yet active.
// It sends an initiating message on chain A and a (reverting) executing
// message on chain B to ensure the proof system handles interop-related
// transactions correctly even before the fork activates.
func RunPreForkActivationTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	rng := rand.New(rand.NewSource(1234))

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)

	// Send an initiating message on chain A (just emits a log via EventLogger).
	eventLogger := aliceA.DeployEventLogger()
	initMsg := aliceA.SendRandomInitMessage(rng, eventLogger, 2, 10)

	// Execute the message on chain B. Interop is not active so the CrossL2Inbox
	// call reverts, but the tx is still included — the supervisor must not
	// re-org out reverted interop transactions when the fork is inactive.
	execMsg := aliceB.SendExecMessage(initMsg, dsl.WithFixedGasLimit(100_000), dsl.WithExpectRevert())

	endTimestamp := sys.L2ChainB.TimestampForBlockNum(bigs.Uint64Strict(execMsg.BlockNumber()))
	t.Require().False(chains[0].Cfg.IsInterop(endTimestamp), "Interop should not be active")

	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	l1Head := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))

	startTimestamp := endTimestamp - 1
	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)

	step1 := marshalTransition(start, 1, firstOptimistic)
	step2 := marshalTransition(start, 2, firstOptimistic, secondOptimistic)
	padding := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	tests := []*transitionTest{
		{
			Name:               "FirstChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1,
			DisputedTraceIndex: 0,
			L1Head:             l1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "SecondChainOptimisticBlock",
			AgreedClaim:        step1,
			DisputedClaim:      step2,
			DisputedTraceIndex: 1,
			L1Head:             l1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "FirstPaddingStep",
			AgreedClaim:        step2,
			DisputedClaim:      padding(3),
			DisputedTraceIndex: 2,
			L1Head:             l1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "Consolidate",
			AgreedClaim:        padding(consolidateStep),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()

	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})
		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

func RunConsolidateValidCrossChainMessageTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	rng := rand.New(rand.NewSource(1234))

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)

	eventLogger := aliceA.DeployEventLogger()
	initMsg := aliceA.SendRandomInitMessage(rng, eventLogger, 2, 10)
	execMsg := aliceB.SendExecMessage(initMsg)

	endTimestamp := sys.L2ChainB.TimestampForBlockNum(bigs.Uint64Strict(execMsg.BlockNumber()))
	startTimestamp := endTimestamp - 1

	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))

	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)
	paddingStep := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	tests := []*transitionTest{
		{
			Name:               "Consolidate-AllValid",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "Consolidate-AllValid-InvalidNoChange",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      paddingStep(consolidateStep),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        false,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()
	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})

		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

func RunInvalidBlockTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	rng := rand.New(rand.NewSource(1234))

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)

	l1BlockBeforeBatches := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	eventLogger := aliceA.DeployEventLogger()
	initMsg := aliceA.SendRandomInitMessage(rng, eventLogger, 2, 10)
	execMsg := aliceB.SendInvalidExecMessage(initMsg)

	endTimestamp := sys.L2ChainB.TimestampForBlockNum(bigs.Uint64Strict(execMsg.BlockNumber()))
	startTimestamp := endTimestamp - 1

	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	sys.L2CLB.Reached(types.CrossSafe, bigs.Uint64Strict(execMsg.BlockNumber()), 10)
	sys.L2ELB.AssertExecMessageNotInBlock(execMsg)

	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))

	start := superRootAtTimestamp(t, chains, startTimestamp)
	crossSafeSuperRootEnd := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)
	paddingStep := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	preReplacementSuperRoot := eth.NewSuperV1(endTimestamp,
		eth.ChainIDAndOutput{ChainID: chains[0].ID, Output: firstOptimistic.OutputRoot},
		eth.ChainIDAndOutput{ChainID: chains[1].ID, Output: secondOptimistic.OutputRoot})

	step1Expected := marshalTransition(start, 1, firstOptimistic)
	step2Expected := marshalTransition(start, 2, firstOptimistic, secondOptimistic)

	tests := []*transitionTest{
		{
			Name:               "FirstChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1Expected,
			DisputedTraceIndex: 0,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "SecondChainOptimisticBlock",
			AgreedClaim:        step1Expected,
			DisputedClaim:      step2Expected,
			DisputedTraceIndex: 1,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "FirstPaddingStep",
			AgreedClaim:        step2Expected,
			DisputedClaim:      paddingStep(3),
			DisputedTraceIndex: 2,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "SecondPaddingStep",
			AgreedClaim:        paddingStep(3),
			DisputedClaim:      paddingStep(4),
			DisputedTraceIndex: 3,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "LastPaddingStep",
			AgreedClaim:        paddingStep(consolidateStep - 1),
			DisputedClaim:      paddingStep(consolidateStep),
			DisputedTraceIndex: consolidateStep - 1,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "Consolidate-ExpectInvalidPendingBlock",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      preReplacementSuperRoot.Marshal(),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        false,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "Consolidate-ReplaceInvalidBlock",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      crossSafeSuperRootEnd.Marshal(),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "AlreadyAtClaimedTimestamp",
			AgreedClaim:        crossSafeSuperRootEnd.Marshal(),
			DisputedClaim:      crossSafeSuperRootEnd.Marshal(),
			DisputedTraceIndex: 5000,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},

		{
			Name:               "FirstChainReachesL1Head",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      interop.InvalidTransition,
			DisputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			L1Head:         l1BlockBeforeBatches.ID(),
			ExpectValid:    true,
			ClaimTimestamp: endTimestamp,
		},
		{
			Name:               "SuperRootInvalidIfUnsupportedByL1Data",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1Expected,
			DisputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			L1Head:         l1BlockBeforeBatches.ID(),
			ExpectValid:    false,
			ClaimTimestamp: endTimestamp,
		},
		{
			Name:               "FromInvalidTransitionHash",
			AgreedClaim:        interop.InvalidTransition,
			DisputedClaim:      interop.InvalidTransition,
			DisputedTraceIndex: 2,
			// The derivation reaches the L1 head before the next block can be created
			L1Head:         l1BlockBeforeBatches.ID(),
			ExpectValid:    true,
			ClaimTimestamp: endTimestamp,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()
	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})

		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

// RunMessageExpiryTest verifies that when a cross-chain message expires (the
// executing message's block timestamp exceeds the init message timestamp plus
// the message expiry window), the block containing the executing message is
// replaced during consolidation. The system must be configured with a short
// message expiry window via WithMessageExpiryWindow.
//
// msgExpiryWindow is the configured message expiry window in seconds; it must
// match the value passed to WithMessageExpiryWindow when creating the system.
func RunMessageExpiryTest(t devtest.T, sys *presets.SimpleInterop, msgExpiryWindow uint64) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	rng := rand.New(rand.NewSource(1234))

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)

	// Send an initiating message on chain A.
	eventLogger := aliceA.DeployEventLogger()
	initMsg := aliceA.SendRandomInitMessage(rng, eventLogger, 2, 10)

	// Record the init message's block timestamp.
	initBlockNum := bigs.Uint64Strict(initMsg.BlockNumber())
	initTimestamp := sys.L2ChainA.TimestampForBlockNum(initBlockNum)

	// Calculate target block numbers past expiry for each chain independently.
	// The message expires when: initTimestamp + expiryWindow < execTimestamp.
	// Add extra blocks for safety margin.
	blockTimeA := sys.L2ChainA.Escape().RollupConfig().BlockTime
	blockTimeB := sys.L2ChainB.Escape().RollupConfig().BlockTime
	t.Require().NotZero(blockTimeA, "block time A must be non-zero")
	t.Require().NotZero(blockTimeB, "block time B must be non-zero")

	currentBlockA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	currentBlockB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	blocksNeededA := (msgExpiryWindow / blockTimeA) + 2
	blocksNeededB := (msgExpiryWindow / blockTimeB) + 2
	targetBlockA := currentBlockA.Number + blocksNeededA
	targetBlockB := currentBlockB.Number + blocksNeededB

	// Wait for both chains to produce blocks past the expiry window.
	sys.L2ELA.Reached(eth.Unsafe, targetBlockA, 60)
	sys.L2ELB.Reached(eth.Unsafe, targetBlockB, 60)

	// Stop batcher B so we control block production on chain B.
	sys.L2BatcherB.Stop()

	// Build the exec tx without submitting to the mempool.
	// InteropMempoolFiltering would reject the expired message, so we
	// bypass the mempool by injecting the raw tx via the test sequencer.
	// This models a malicious sequencer force-including an invalid tx.
	rawExecTx, execTxHash := aliceB.PrepareExecTx(initMsg)

	// Inject the expired exec message into a block on chain B via test sequencer.
	parentB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	chainBID := sys.L2ChainB.ChainID()
	sys.TestSequencer.SequenceBlockWithTxs(t, chainBID, parentB.Hash, [][]byte{rawExecTx})

	// Also advance chain A by one empty block to keep timestamps aligned.
	parentA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	chainAID := sys.L2ChainA.ChainID()
	sys.TestSequencer.SequenceBlock(t, chainAID, parentA.Hash)

	// The injected block is the new unsafe head on chain B.
	newHeadB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	execBlockNum := newHeadB.Number

	// Verify the expired exec tx is actually in the injected block before consolidation.
	sys.L2ELB.AssertTxInBlock(execBlockNum, execTxHash)

	// Restart batcher B so batch data gets submitted to L1.
	sys.L2BatcherB.Start()

	endTimestamp := sys.L2ChainB.TimestampForBlockNum(execBlockNum)
	t.Require().Greaterf(endTimestamp, initTimestamp+msgExpiryWindow,
		"exec message timestamp %d should exceed init timestamp %d + expiry window %d",
		endTimestamp, initTimestamp, msgExpiryWindow)
	startTimestamp := endTimestamp - 1

	// Wait for cross-safe validation, which should replace the invalid block.
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	sys.L2CLB.Reached(types.CrossSafe, execBlockNum, 30)

	// Verify the expired exec tx was reorged out during consolidation.
	sys.L2ELB.AssertTxNotInBlock(execBlockNum, execTxHash)

	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))

	crossSafeSuperRootEnd := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)

	start := superRootAtTimestamp(t, chains, startTimestamp)
	paddingStep := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	preReplacementSuperRoot := eth.NewSuperV1(endTimestamp,
		eth.ChainIDAndOutput{ChainID: chains[0].ID, Output: firstOptimistic.OutputRoot},
		eth.ChainIDAndOutput{ChainID: chains[1].ID, Output: secondOptimistic.OutputRoot})

	tests := []*transitionTest{
		{
			Name:               "Consolidate-ExpectInvalidPendingBlock",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      preReplacementSuperRoot.Marshal(),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        false,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "Consolidate-ReplaceExpiredMessage",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      crossSafeSuperRootEnd.Marshal(),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()
	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})

		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

// RunDepositMessageTest verifies that the fault proof system correctly handles
// consolidation when a cross-chain message is initiated via an L1 deposit transaction.
func RunDepositMessageTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	rng := rand.New(rand.NewSource(5678))

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceL1 := aliceA.AsEL(sys.L1EL)
	sys.FunderL1.Fund(aliceL1, eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)

	eventLogger := aliceA.DeployEventLogger()
	depositEOA := aliceA.ViaDepositTx(aliceL1, sys.L2ELA, sys.L2ChainA)
	initMsg := depositEOA.SendRandomInitMessage(rng, eventLogger)
	execMsg := aliceB.SendExecMessage(initMsg)

	endTimestamp := sys.L2ChainB.TimestampForBlockNum(bigs.Uint64Strict(execMsg.BlockNumber()))
	startTimestamp := endTimestamp - 1

	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))

	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)
	paddingStep := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	tests := []*transitionTest{
		{
			Name:               "Consolidate",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      end.Marshal(),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "Consolidate-InvalidNoChange",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      paddingStep(consolidateStep),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        false,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()
	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})

		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

// RunDepositMessageInvalidExecutionTest verifies that the fault proof system correctly
// detects an invalid executing message when the initiating message was sent via an L1
// deposit transaction. The executing message uses an invalid identifier, so consolidation
// must replace the optimistic block with the cross-safe result.
func RunDepositMessageInvalidExecutionTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	rng := rand.New(rand.NewSource(9012))

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceL1 := aliceA.AsEL(sys.L1EL)
	sys.FunderL1.Fund(aliceL1, eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)

	eventLogger := aliceA.DeployEventLogger()
	depositEOA := aliceA.ViaDepositTx(aliceL1, sys.L2ELA, sys.L2ChainA)
	initMsg := depositEOA.SendRandomInitMessage(rng, eventLogger)
	execMsg := aliceB.SendInvalidExecMessage(initMsg)

	endTimestamp := sys.L2ChainB.TimestampForBlockNum(bigs.Uint64Strict(execMsg.BlockNumber()))
	startTimestamp := endTimestamp - 1

	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	sys.L2CLB.Reached(types.CrossSafe, bigs.Uint64Strict(execMsg.BlockNumber()), 10)
	sys.L2ELB.AssertExecMessageNotInBlock(execMsg)

	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))

	start := superRootAtTimestamp(t, chains, startTimestamp)
	crossSafeEnd := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)
	paddingStep := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	optimisticEnd := eth.NewSuperV1(endTimestamp,
		eth.ChainIDAndOutput{ChainID: chains[0].ID, Output: firstOptimistic.OutputRoot},
		eth.ChainIDAndOutput{ChainID: chains[1].ID, Output: secondOptimistic.OutputRoot})

	tests := []*transitionTest{
		{
			Name:               "Consolidate",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      crossSafeEnd.Marshal(),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        true,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "Consolidate-InvalidNoChange",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      paddingStep(consolidateStep),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        false,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
		{
			Name:               "Consolidate-ExpectInvalidPendingBlock",
			AgreedClaim:        paddingStep(consolidateStep),
			DisputedClaim:      optimisticEnd.Marshal(),
			DisputedTraceIndex: consolidateStep,
			ExpectValid:        false,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
		},
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()
	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})

		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}
