package superfaultproofs

import (
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	interopTypes "github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
		{ID: sys.L2ChainA.ChainID(), Cfg: sys.L2ChainA.Escape().RollupConfig(), Rollup: sys.L2CLA.Escape().RollupAPI(), EL: sys.L2ELA, Batcher: sys.L2BatcherA},
		{ID: sys.L2ChainB.ChainID(), Cfg: sys.L2ChainB.Escape().RollupConfig(), Rollup: sys.L2CLB.Escape().RollupAPI(), EL: sys.L2ELB, Batcher: sys.L2BatcherB},
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

// awaitSafeHeadsStalled waits until every node's safe head has stopped advancing
// for at least 10 seconds.
func awaitSafeHeadsStalled(t devtest.T, nodes ...*dsl.L2CLNode) {
	var last []eth.BlockID
	var stableSince time.Time
	t.Require().Eventually(func() bool {
		cur := make([]eth.BlockID, len(nodes))
		for i, n := range nodes {
			cur[i] = n.SyncStatus().SafeL2.ID()
		}
		if slices.Equal(cur, last) {
			if stableSince.IsZero() {
				stableSince = time.Now()
			}
			return time.Since(stableSince) >= 10*time.Second
		}
		last = cur
		stableSince = time.Time{}
		return false
	}, 2*time.Minute, 2*time.Second, "safe heads did not stall in time")
}

// awaitOptimisticPattern polls the supernode until every chain in mustHave has
// optimistic data and every chain in mustMiss does not.
func awaitOptimisticPattern(t devtest.T, sn *dsl.Supernode, timestamp uint64, mustHave, mustMiss []eth.ChainID) eth.SuperRootAtTimestampResponse {
	var resp eth.SuperRootAtTimestampResponse
	t.Require().Eventually(func() bool {
		resp = sn.SuperRootAtTimestamp(timestamp)
		for _, id := range mustHave {
			if _, has := resp.OptimisticAtTimestamp[id]; !has {
				return false
			}
		}
		for _, id := range mustMiss {
			if _, has := resp.OptimisticAtTimestamp[id]; has {
				return false
			}
		}
		return true
	}, 2*time.Minute, 2*time.Second, "timed out waiting for optimistic pattern")
	return resp
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
	cmd := exec.Command(exePath, argv[1:]...)
	cmd.Dir = tmpDir
	cmd.Env = append(append(cmd.Env, os.Environ()...), "NO_COLOR=1")

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
	}
}

// RunSuperFaultProofTest encapsulates the basic super fault proof test flow.
func RunSuperFaultProofTest(t devtest.T, sys *presets.SimpleInterop) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// -- Stage 1: Freeze batch submission ----------------------------------
	chains[0].Batcher.Stop()
	chains[1].Batcher.Stop()
	defer func() {
		chains[0].Batcher.Start()
		chains[1].Batcher.Start()
	}()
	awaitSafeHeadsStalled(t, sys.L2CLA, sys.L2CLB)

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
	respBefore := awaitOptimisticPattern(t, sys.SuperRoots, endTimestamp,
		nil, []eth.ChainID{chains[0].ID, chains[1].ID})
	l1HeadBefore := respBefore.CurrentL1

	// L1 head where only the first chain has batch data.
	chains[0].Batcher.Start()
	respAfterFirst := awaitOptimisticPattern(t, sys.SuperRoots, endTimestamp,
		[]eth.ChainID{chains[0].ID}, []eth.ChainID{chains[1].ID})
	l1HeadAfterFirst := respAfterFirst.CurrentL1
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
