package superfaultproofs

import (
	"context"
	"math"
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
		next := c.Cfg.TimestampForBlock(status.SafeL2.Number + 1)
		if next > ts {
			ts = next
		}
	}
	t.Require().NotZero(ts, "end timestamp must be non-zero")
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
func optimisticBlockAtTimestamp(t devtest.T, c *chain, timestamp uint64) interopTypes.OptimisticBlock {
	blockNum, err := c.Cfg.TargetBlockNumber(timestamp)
	t.Require().NoError(err)
	out, err := c.Rollup.OutputAtBlock(t.Ctx(), blockNum)
	t.Require().NoError(err)
	return interopTypes.OptimisticBlock{BlockHash: out.BlockRef.Hash, OutputRoot: out.OutputRoot}
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

// l1BlockWithLocalSafeBlocks finds an L1 block where the specified chains either do or do not have safe blocks.
func l1BlockWithLocalSafeBlocks(t devtest.T, l1El *dsl.L1ELNode, sn *dsl.Supernode, timestamp uint64, hasSafe, notSafe []eth.ChainID) eth.BlockID {
	t.Logf("Finding L1 block where %v have safe blocks and %v do not", hasSafe, notSafe)
	var l1Block eth.BlockID
	t.Require().Eventually(func() bool {
		resp := sn.SuperRootAtTimestamp(timestamp)

		candidate := uint64(math.MaxUint64)
		for _, id := range notSafe {
			if optimistic, has := resp.OptimisticAtTimestamp[id]; has && optimistic.RequiredL1.Number <= candidate {
				candidate = optimistic.RequiredL1.Number - 1 // We need this chain to not have a safe block, so L1 head must be the block before it.
			}
		}
		// If we didn't have any notSafe chains, we can use the current L1 block.
		if candidate == math.MaxUint64 {
			candidate = resp.CurrentL1.Number
		}

		// Now verify that all the required chains have a safe block at the candidate L1 block.
		for _, id := range hasSafe {
			if optimistic, has := resp.OptimisticAtTimestamp[id]; !has {
				return false
			} else if optimistic.RequiredL1.Number > candidate {
				return false
			}
		}

		l1Block = l1El.BlockRefByNumber(candidate).ID()
		return true
	}, 2*time.Minute, 2*time.Second, "timed out waiting for l1 block")
	return l1Block
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
	firstOptimistic := optimisticBlockAtTimestamp(t, chains[0], endTimestamp+1)
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
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// -- Stage 1: Freeze batch submission ----------------------------------
	chains[1].Batcher.Stop() // Stop chain 1 first and wait for chains[0] to have at least that local safe head.
	t.Cleanup(chains[1].Batcher.Start)
	// Wait for safe heads to stall (local safe will continue on chains[0] but interop validation can't progress because chains[1] local safe has stalled)
	chains[1].CLNode.WaitForStall(types.CrossSafe)
	chains[0].Batcher.Stop()
	t.Cleanup(chains[0].Batcher.Start)
	chains[0].CLNode.WaitForStall(types.LocalSafe) // Wait for chains[0] local safe head to stall

	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	startTimestamp := endTimestamp - 1

	// Ensure both chains have produced the target blocks as unsafe.
	for _, c := range chains {
		target, err := c.Cfg.TargetBlockNumber(endTimestamp)
		t.Require().NoError(err)
		c.EL.Reached(eth.Unsafe, target, 60)
	}

	// -- Stage 2: Capture L1 heads at different batch-availability points --

	// L1 head where neither chain has batch data at endTimestamp.
	l1HeadBefore := l1BlockWithLocalSafeBlocks(t, sys.L1EL, sys.SuperRoots, endTimestamp, nil, []eth.ChainID{chains[0].ID, chains[1].ID})

	// L1 head where only the first chain has batch data.
	chains[0].Batcher.Start()
	l1HeadAfterFirst := l1BlockWithLocalSafeBlocks(t, sys.L1EL, sys.SuperRoots, endTimestamp, []eth.ChainID{chains[0].ID}, []eth.ChainID{chains[1].ID})
	chains[0].Batcher.Stop()

	// L1 head where both chains have batch data (fully validated).
	chains[1].Batcher.Start()
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))
	chains[1].Batcher.Stop()

	// --- Stage 3: Build expected transition states --------------------------
	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, chains[0], endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, chains[1], endTimestamp)

	step1 := marshalTransition(start, 1, firstOptimistic)
	step2 := marshalTransition(start, 2, firstOptimistic, secondOptimistic)
	padding := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	// --- Stage 4: Transition test cases ------------------------------------
	tests := buildTransitionTests(start, end, step1, step2, padding,
		l1HeadCurrent, l1HeadBefore, l1HeadAfterFirst, endTimestamp)

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

// RunVariedBlockTimesTest verifies that the super fault proof system works
// correctly when chains have different block times (e.g. 1s and 2s), exercising
// edge cases where not every chain produces a new block at every timestamp.
//
// The system must be configured with varied block times before calling this
// function (e.g. via presets.WithL2BlockTimes).
func RunVariedBlockTimesTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// Verify chains have different block times.
	t.Require().NotEqual(chains[0].Cfg.BlockTime, chains[1].Cfg.BlockTime,
		"this test requires chains with different block times")

	// -- Stage 1: Freeze batch submission ----------------------------------
	chains[1].Batcher.Stop()
	t.Cleanup(chains[1].Batcher.Start)
	chains[1].CLNode.WaitForStall(types.CrossSafe)
	chains[0].Batcher.Stop()
	t.Cleanup(chains[0].Batcher.Start)
	chains[0].CLNode.WaitForStall(types.LocalSafe)

	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	startTimestamp := endTimestamp - 1

	// Ensure both chains have produced the target blocks as unsafe.
	for _, c := range chains {
		target, err := c.Cfg.TargetBlockNumber(endTimestamp)
		t.Require().NoError(err)
		c.EL.Reached(eth.Unsafe, target, 60)
	}

	// -- Stage 2: Capture L1 heads at different batch-availability points --

	l1HeadBefore := l1BlockWithLocalSafeBlocks(t, sys.L1EL, sys.SuperRoots, endTimestamp, nil, []eth.ChainID{chains[0].ID, chains[1].ID})

	chains[0].Batcher.Start()
	l1HeadAfterFirst := l1BlockWithLocalSafeBlocks(t, sys.L1EL, sys.SuperRoots, endTimestamp, []eth.ChainID{chains[0].ID}, []eth.ChainID{chains[1].ID})
	chains[0].Batcher.Stop()

	chains[1].Batcher.Start()
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))
	chains[1].Batcher.Stop()

	// -- Stage 3: Build expected transition states --------------------------
	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)

	firstOptimistic := optimisticBlockAtTimestamp(t, chains[0], endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, chains[1], endTimestamp)

	step1 := marshalTransition(start, 1, firstOptimistic)
	step2 := marshalTransition(start, 2, firstOptimistic, secondOptimistic)
	padding := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}

	// -- Stage 4: Transition test cases ------------------------------------
	tests := buildTransitionTests(start, end, step1, step2, padding,
		l1HeadCurrent, l1HeadBefore, l1HeadAfterFirst, endTimestamp)

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

	firstOptimistic := optimisticBlockAtTimestamp(t, chains[0], endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, chains[1], endTimestamp)
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

	firstOptimistic := optimisticBlockAtTimestamp(t, chains[0], endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, chains[1], endTimestamp)
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
