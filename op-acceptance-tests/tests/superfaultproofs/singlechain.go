package superfaultproofs

import (
	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// singleChain bundles the DSL handles for the single L2 chain in a SingleChainInterop system.
func singleChainFrom(sys *presets.SingleChainInterop) *chain {
	return &chain{
		ID:      sys.L2ChainA.ChainID(),
		Cfg:     sys.L2ChainA.Escape().RollupConfig(),
		Rollup:  sys.L2CLA.Escape().RollupAPI(),
		EL:      sys.L2ELA,
		CLNode:  sys.L2CLA,
		Batcher: sys.L2BatcherA,
	}
}

// RunSingleChainSuperFaultProofSmokeTest is a minimal smoke test for single-chain super fault proofs.
// It verifies that the super-root transition works correctly when the dependency set has only one chain.
// The test stops the batcher, waits for the safe head to stall, then resumes batching and verifies
// a basic set of valid/invalid transitions through both the FPP and challenger trace provider.
func RunSingleChainSuperFaultProofSmokeTest(t devtest.T, sys *presets.SingleChainInterop) {
	runSingleChainSuperFaultProofSmokeTest(t, sys, prepareDefaultSingleChainTarget)
}

// RunSingleChainSuperFaultProofSDMSmokeTest verifies the same single-chain super-root transition
// while the disputed block contains an SDM PostExec tx. This proves kona-host super --native can
// derive interop batches whose L2 payload includes SDM's synthetic post-exec transaction.
func RunSingleChainSuperFaultProofSDMSmokeTest(t devtest.T, sys *presets.SingleChainInterop) {
	runSingleChainSuperFaultProofSmokeTest(t, sys, prepareSDMSingleChainTarget)
}

type singleChainTargetFn func(t devtest.T, sys *presets.SingleChainInterop, c *chain, chains []*chain) (startTimestamp uint64, endTimestamp uint64, targetBlock uint64)

func runSingleChainSuperFaultProofSmokeTest(t devtest.T, sys *presets.SingleChainInterop, prepareTarget singleChainTargetFn) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")

	c := singleChainFrom(sys)
	chains := []*chain{c}

	// Stop batch submission so safe head stalls, then we have a known boundary.
	c.Batcher.Stop()
	sys.L2CLA.WaitForStall(safety.CrossSafe)

	startTimestamp, endTimestamp, target := prepareTarget(t, sys, c, chains)

	// Batcher is stopped, so no batch data for endTimestamp is on L1.
	l1HeadBefore := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()

	// Resume batching and wait for the safe head to reach the target.
	c.Batcher.Start()
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	c.EL.Reached(eth.Safe, target, 60)
	l1HeadCurrent := sys.L1EL.BlockRefByLabel(eth.Unsafe).ID()

	// Build expected transition states for a single chain.
	start := superRootAtTimestamp(t, chains, startTimestamp)
	end := superRootAtTimestamp(t, chains, endTimestamp)

	optimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), c.ID, endTimestamp)

	// With one chain: step 0 = chain's optimistic block, steps 1..consolidateStep-1 = padding,
	// consolidateStep = consolidation to next super root.
	step1 := marshalTransition(start, 1, optimistic)
	padding := func(step uint64) []byte {
		return marshalTransition(start, step, optimistic)
	}

	tests := []*transitionTest{
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
			Name:               "ChainOptimisticBlock",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      step1,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "ChainOptimisticBlock-InvalidNoChange",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      start.Marshal(),
			DisputedTraceIndex: 0,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
		{
			Name:               "FirstPaddingStep",
			AgreedClaim:        step1,
			DisputedClaim:      padding(2),
			DisputedTraceIndex: 1,
			L1Head:             l1HeadCurrent,
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
		{
			Name:               "ChainReachesL1Head",
			AgreedClaim:        start.Marshal(),
			DisputedClaim:      super.InvalidTransition,
			DisputedTraceIndex: 0,
			L1Head:             l1HeadBefore,
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

func prepareDefaultSingleChainTarget(t devtest.T, _ *presets.SingleChainInterop, c *chain, chains []*chain) (uint64, uint64, uint64) {
	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	startTimestamp := endTimestamp - 1

	// Ensure the chain has produced the target block as unsafe.
	target, err := c.Cfg.TargetBlockNumber(endTimestamp)
	t.Require().NoError(err)
	c.EL.Reached(eth.Unsafe, target, 60)
	return startTimestamp, endTimestamp, target
}

func prepareSDMSingleChainTarget(t devtest.T, sys *presets.SingleChainInterop, c *chain, _ []*chain) (uint64, uint64, uint64) {
	targetBlock := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	validation, err := sdmpkg.ValidatePostExecBlock(
		t.Ctx(),
		sys.L2ELA.Escape().L2EthClient().RPC(),
		targetBlock,
		sdmpkg.ValidationOptions{CheckReceipts: true},
	)
	t.Require().NoError(err, "target block must contain a valid SDM PostExec tx")
	t.Require().Equal(targetBlock, validation.Payload.BlockNumber, "post-exec payload must be anchored to the target block")

	endTimestamp := c.Cfg.TimestampForBlock(targetBlock)
	return endTimestamp - 1, endTimestamp, targetBlock
}

func submitTxWithoutWait(
	t devtest.T,
	alice *dsl.EOA,
	nonce uint64,
	opts ...txplan.Option,
) *txplan.PlannedTx {
	combined := append([]txplan.Option{
		alice.Plan(),
		txplan.WithNonce(nonce),
	}, opts...)
	ptx := txplan.NewPlannedTx(combined...)
	_, err := ptx.Submitted.Eval(t.Ctx())
	t.Require().NoError(err, "failed to submit tx with nonce %d", nonce)
	return ptx
}

func mustFindRepeatedSlotBlock(
	t devtest.T,
	sys *presets.SingleChainInterop,
	minUserTxs int,
	maxAttempts int,
) uint64 {
	l := t.Logger()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		alice := sys.FunderA.NewFundedEOA(eth.OneEther)
		stateBloatAddr := deployContract(t, alice, sdmpkg.StateBloatBin)

		const batchSize = 50
		const slotCount = 20
		startNonce := alice.PendingNonce()
		plannedTxs := make([]*txplan.PlannedTx, 0, batchSize)

		l.Info("Submitting repeated-slot workload",
			"attempt", attempt,
			"alice", alice.Address(),
			"contract", stateBloatAddr,
			"startNonce", startNonce,
			"batchSize", batchSize,
			"slotCount", slotCount)

		for i := 0; i < batchSize; i++ {
			nonce := startNonce + uint64(i)
			plannedTxs = append(plannedTxs, submitTxWithoutWait(
				t,
				alice,
				nonce,
				txplan.WithTo(&stateBloatAddr),
				txplan.WithData(sdmpkg.EncodeRun(slotCount)),
				txplan.WithGasLimit(1_000_000),
			))
		}

		blockTxCounts := make(map[uint64]int)
		for i, ptx := range plannedTxs {
			receipt, err := ptx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "attempt %d tx %d: failed to get receipt", attempt, i)
			t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status,
				"attempt %d tx %d: must succeed", attempt, i)

			blockNum := bigs.Uint64Strict(receipt.BlockNumber)
			blockTxCounts[blockNum]++
		}

		var targetBlockNum uint64
		var targetUserTxCount int
		for blockNum, txCount := range blockTxCounts {
			if txCount > targetUserTxCount {
				targetBlockNum = blockNum
				targetUserTxCount = txCount
			}
		}
		if targetUserTxCount < minUserTxs {
			l.Warn("Repeated-slot workload did not produce a dense-enough block",
				"attempt", attempt,
				"requiredUserTxs", minUserTxs,
				"bestUserTxs", targetUserTxCount,
				"bestBlock", targetBlockNum)
			continue
		}

		return targetBlockNum
	}

	t.Require().FailNowf("repeated-slot workload failed",
		"no block with at least %d user txs found after %d attempts", minUserTxs, maxAttempts)
	return 0
}

func deployContract(t devtest.T, eoa *dsl.EOA, hexBytecode string) common.Address {
	tx := txplan.NewPlannedTx(eoa.Plan(), txplan.WithData(common.FromHex(hexBytecode)))
	res, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "failed to deploy contract")
	t.Require().Equal(types.ReceiptStatusSuccessful, res.Status, "contract deployment must succeed")
	return res.ContractAddress
}
