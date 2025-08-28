package interop

import (
	"bytes"
	"fmt"
	"log/slog"
	"math/big"
	"reflect"
	"testing"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	fpHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	sync2 "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum-optimism/optimism/op-program/client/interop"
	"github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const (
	stepsPerTimestamp = 128
	consolidateStep   = stepsPerTimestamp - 1
)

func TestInteropFaultProofs_TraceExtensionActivation(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t)

	system.AddL2Block(system.Actors.ChainA)
	system.AddL2Block(system.Actors.ChainB)

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData()

	endTimestamp := system.Actors.ChainA.Sequencer.L2Safe().Time

	agreedClaim := system.Outputs.SuperRoot(endTimestamp).Marshal()
	disputedClaim := system.Outputs.TransitionState(endTimestamp, 1,
		system.Outputs.OptimisticBlockAtTimestamp(system.Actors.ChainA, endTimestamp+1)).Marshal()
	disputedTraceIndex := int64(stepsPerTimestamp)
	tests := []*transitionTest{
		{
			name:               "CorrectlyDidNotActivate",
			agreedClaim:        agreedClaim,
			disputedClaim:      disputedClaim,
			disputedTraceIndex: disputedTraceIndex,
			// Trace extension does not activate because we have not reached the proposal timestamp yet
			proposalTimestamp: endTimestamp + 1,
			expectValid:       true,
		},
		{
			name:               "IncorrectlyDidNotActivate",
			agreedClaim:        agreedClaim,
			disputedClaim:      disputedClaim,
			disputedTraceIndex: disputedTraceIndex,
			// Trace extension should have activated because we have gone past the proposal timestamp yet, but did not
			proposalTimestamp: endTimestamp,
			expectValid:       false,
		},
		{
			name:               "CorrectlyActivated",
			agreedClaim:        agreedClaim,
			disputedClaim:      agreedClaim,
			disputedTraceIndex: disputedTraceIndex,
			// Trace extension does not activate because we have not reached the proposal timestamp yet
			proposalTimestamp: endTimestamp,
			expectValid:       true,
		},
		{
			name:               "IncorrectlyActivated",
			agreedClaim:        agreedClaim,
			disputedClaim:      agreedClaim,
			disputedTraceIndex: disputedTraceIndex,
			// Trace extension does not activate because we have not reached the proposal timestamp yet
			proposalTimestamp: endTimestamp + 1,
			expectValid:       false,
		},
	}
	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs_UnsafeProposal(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t)

	system.AddL2Block(system.Actors.ChainA)
	system.AddL2Block(system.Actors.ChainB)

	proposalTimestamp := system.Actors.ChainA.Sequencer.L2Unsafe().Time
	agreedTimestamp := proposalTimestamp - 1
	agreedClaim := system.Outputs.SuperRoot(agreedTimestamp).Marshal()
	disputedClaim := system.Outputs.TransitionState(agreedTimestamp, 1).Marshal()
	disputedTraceIndex := int64(0)
	tests := []*transitionTest{
		{
			name:               "ProposedUnsafeBlock-NotValid",
			agreedClaim:        agreedClaim,
			disputedClaim:      disputedClaim,
			disputedTraceIndex: disputedTraceIndex,
			proposalTimestamp:  proposalTimestamp,
			expectValid:        false,
		},
		{
			name:               "ProposedUnsafeBlock-ShouldBeInvalid",
			agreedClaim:        agreedClaim,
			disputedClaim:      interop.InvalidTransition,
			disputedTraceIndex: disputedTraceIndex,
			proposalTimestamp:  proposalTimestamp,
			expectValid:        true,
		},
	}
	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs_ConsolidateValidCrossChainMessage(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t)
	actors := system.Actors

	alice := system.CreateUser()
	emitter := system.DeployEmitterContracts()

	system.AddL2Block(system.Actors.ChainA, dsl.WithL2BlockTransactions(emitter.EmitMessage(alice, "hello")))
	initMsg := emitter.LastEmittedMessage()
	system.AddL2Block(system.Actors.ChainB, dsl.WithL2BlockTransactions(system.InboxContract.Execute(alice, initMsg)))

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainB)
	})

	endTimestamp := system.Actors.ChainA.Sequencer.L2Safe().Time
	startTimestamp := endTimestamp - 1
	end := system.Outputs.SuperRoot(endTimestamp)

	paddingStep := func(step uint64) []byte {
		return system.Outputs.TransitionState(startTimestamp, step,
			system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
			system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
		).Marshal()
	}

	tests := []*transitionTest{
		{
			name:               "Consolidate-AllValid",
			agreedClaim:        paddingStep(consolidateStep),
			disputedClaim:      end.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
		},
		{
			name:               "Consolidate-AllValid-InvalidNoChange",
			agreedClaim:        paddingStep(consolidateStep),
			disputedClaim:      paddingStep(consolidateStep),
			disputedTraceIndex: consolidateStep,
			expectValid:        false,
		},
	}
	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs_PreForkActivation(gt *testing.T) {
	// TODO(#16166): Fix non-genesis Interop activation proofs
	gt.Skip()
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t, dsl.SetInteropForkScheduledButInactive())

	actors := system.Actors
	endTimestamp := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime
	startTimestamp := endTimestamp - 1
	require.False(t, actors.ChainA.RollupCfg.IsInterop(endTimestamp), "Interop should not be active")

	alice := system.CreateUser()
	emitter := system.DeployEmitterContracts()

	system.AddL2Block(system.Actors.ChainA, dsl.WithL2BlockTransactions(emitter.EmitMessage(alice, "hello")))
	initMsg := emitter.LastEmittedMessage()
	system.AddL2Block(system.Actors.ChainB,
		dsl.WithL2BlockTransactions(system.InboxContract.Execute(alice, initMsg,
			dsl.WithPayload([]byte("wrong")),
			// CrossL2Inbox contract isn't deployed so the tx will revert. Need to avoid using eth_estimateGas
			dsl.WithFixedGasLimit())))
	system.InboxContract.LastTransaction().CheckIncluded(dsl.WithRevertExpected())

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainB)
	})
	// Check that the supervisor didn't re-org out this transaction.
	// Interop isn't active yet so the extra derivation rules to validate executing messages must not be active.
	system.InboxContract.LastTransaction().CheckIncluded(dsl.WithRevertExpected())

	start := system.Outputs.SuperRoot(startTimestamp)
	end := system.Outputs.SuperRoot(endTimestamp)

	step1Expected := system.Outputs.TransitionState(startTimestamp, 1,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
	).Marshal()

	step2Expected := system.Outputs.TransitionState(startTimestamp, 2,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	firstPaddingStep := system.Outputs.TransitionState(startTimestamp, 3,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	lastPaddingStep := system.Outputs.TransitionState(startTimestamp, consolidateStep,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	tests := []*transitionTest{
		{
			name:               "FirstChainOptimisticBlock",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			disputedTraceIndex: 0,
			expectValid:        true,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
		},
		{
			name:               "SecondChainOptimisticBlock",
			agreedClaim:        step1Expected,
			disputedClaim:      step2Expected,
			disputedTraceIndex: 1,
			expectValid:        true,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
		},
		{
			name:               "FirstPaddingStep",
			agreedClaim:        step2Expected,
			disputedClaim:      firstPaddingStep,
			disputedTraceIndex: 2,
			expectValid:        true,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
		},
		{
			name:               "Consolidate",
			agreedClaim:        lastPaddingStep,
			disputedClaim:      end.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
		},
	}

	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t)

	system.AddL2Block(system.Actors.ChainA)
	system.AddL2Block(system.Actors.ChainB)

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainB)
	})

	actors := system.Actors

	endTimestamp := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime
	startTimestamp := endTimestamp - 1

	start := system.Outputs.SuperRoot(startTimestamp)
	end := system.Outputs.SuperRoot(endTimestamp)

	step1Expected := system.Outputs.TransitionState(startTimestamp, 1,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
	).Marshal()

	step2Expected := system.Outputs.TransitionState(startTimestamp, 2,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	paddingStep := func(step uint64) []byte {
		return system.Outputs.TransitionState(startTimestamp, step,
			system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
			system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
		).Marshal()
	}

	tests := []*transitionTest{
		{
			name:               "ClaimDirectToNextTimestamp",
			agreedClaim:        start.Marshal(),
			disputedClaim:      end.Marshal(),
			disputedTraceIndex: 0,
			expectValid:        false,
		},
		{
			name:               "FirstChainOptimisticBlock",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			disputedTraceIndex: 0,
			expectValid:        true,
		},
		{
			name:               "FirstChainOptimisticBlock-InvalidNoChange",
			agreedClaim:        start.Marshal(),
			disputedClaim:      start.Marshal(),
			disputedTraceIndex: 0,
			expectValid:        false,
		},
		{
			name:               "SecondChainOptimisticBlock",
			agreedClaim:        step1Expected,
			disputedClaim:      step2Expected,
			disputedTraceIndex: 1,
			expectValid:        true,
		},
		{
			name:               "SecondChainOptimisticBlock-InvalidNoChange",
			agreedClaim:        step1Expected,
			disputedClaim:      step1Expected,
			disputedTraceIndex: 1,
			expectValid:        false,
		},
		{
			name:               "FirstPaddingStep",
			agreedClaim:        step2Expected,
			disputedClaim:      paddingStep(3),
			disputedTraceIndex: 2,
			expectValid:        true,
		},
		{
			name:               "FirstPaddingStep-InvalidNoChange",
			agreedClaim:        step2Expected,
			disputedClaim:      step2Expected,
			disputedTraceIndex: 2,
			expectValid:        false,
		},
		{
			name:               "SecondPaddingStep",
			agreedClaim:        paddingStep(3),
			disputedClaim:      paddingStep(4),
			disputedTraceIndex: 3,
			expectValid:        true,
		},
		{
			name:               "SecondPaddingStep-InvalidNoChange",
			agreedClaim:        paddingStep(3),
			disputedClaim:      paddingStep(3),
			disputedTraceIndex: 3,
			expectValid:        false,
		},
		{
			name:               "LastPaddingStep",
			agreedClaim:        paddingStep(consolidateStep - 1),
			disputedClaim:      paddingStep(consolidateStep),
			disputedTraceIndex: consolidateStep - 1,
			expectValid:        true,
		},
		{
			// The proposed block timestamp is after the unsafe head block timestamp.
			// Expect to transition to invalid because the unsafe head is reached but challenger needs to handle
			// not having any data at the next timestamp because the chain doesn't extend that far.
			name:        "DisputeTimestampAfterChainHeadChainA",
			agreedClaim: end.Marshal(),
			// With 2 second block times, we haven't yet reached the next block on the first chain so it's still valid
			disputedClaim: system.Outputs.TransitionState(endTimestamp, 1,
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp+1),
			).Marshal(),
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: consolidateStep + 1,
			expectValid:        true,
		},
		{
			// The proposed block timestamp is after the unsafe head block timestamp.
			// Expect to transition to invalid because the unsafe head is reached but challenger needs to handle
			// not having any data at the next timestamp because the chain doesn't extend that far.
			name: "DisputeTimestampAfterChainHeadChainB",
			agreedClaim: system.Outputs.TransitionState(endTimestamp, 1,
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp+1),
			).Marshal(),
			// With 2 second block times, we haven't yet reached the next block on the second chain so it's still valid
			disputedClaim: system.Outputs.TransitionState(endTimestamp, 2,
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp+1),
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp+1),
			).Marshal(),
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: consolidateStep + 2,
			expectValid:        true,
		},
		{
			// The proposed block timestamp is after the unsafe head block timestamp.
			// Expect to transition to invalid because the unsafe head is reached but challenger needs to handle
			// not having any data at the next timestamp because the chain doesn't extend that far.
			name: "DisputeTimestampAfterChainHeadConsolidate",
			agreedClaim: system.Outputs.TransitionState(endTimestamp, consolidateStep,
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp+1),
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp+1),
			).Marshal(),
			// With 2 second block times, we haven't yet reached the next block on either chain so it's still valid
			// It will have an incremented timestamp but the same chain output roots
			disputedClaim:      system.Outputs.SuperRoot(endTimestamp + 1).Marshal(),
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 2*stepsPerTimestamp - 1,
			expectValid:        true,
		},
		{
			// The proposed block timestamp is after the unsafe head block timestamp.
			// Expect to transition to invalid because the unsafe head is reached but challenger needs to handle
			// not having any data at the next timestamp because the chain doesn't extend that far.
			name:        "DisputeBlockAfterChainHead-FirstChain",
			agreedClaim: system.Outputs.SuperRoot(endTimestamp + 1).Marshal(),
			// Timestamp has advanced enough to expect the next block now, but it doesn't exit so transition to invalid
			disputedClaim:      interop.InvalidTransition,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 2 * stepsPerTimestamp,
			expectValid:        true,
		},
		{
			// The agreed and disputed claim are both after the current chain head
			name:               "AgreedBlockAfterChainHead-Consolidate",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 4*stepsPerTimestamp - 1,
			expectValid:        true,
		},
		{
			// The agreed and disputed claim are both after the current chain head and disputing an optimistic block
			name:               "AgreedBlockAfterChainHead-Optimistic",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 4*stepsPerTimestamp + 1,
			expectValid:        true,
		},

		{
			name:               "FirstChainReachesL1Head",
			agreedClaim:        start.Marshal(),
			disputedClaim:      interop.InvalidTransition,
			disputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: true,
		},
		{
			name:               "SecondChainReachesL1Head",
			agreedClaim:        step1Expected,
			disputedClaim:      interop.InvalidTransition,
			disputedTraceIndex: 1,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().GetCanonicalHash(1),
			expectValid: true,
		},
		{
			name:               "SuperRootInvalidIfUnsupportedByL1Data",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			disputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: false,
		},
		{
			name:               "FromInvalidTransitionHash",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			disputedTraceIndex: 2,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: true,
		},
	}

	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs_IntraBlock(gt *testing.T) {
	cases := []intraBlockTestCase{
		new(cascadeInvalidBlockCase),
		new(swapCascadeInvalidBlockCase),
		new(cyclicDependencyInvalidCase),
		new(cyclicDependencyValidCase),
		new(longDependencyChainValidCase),
		new(sameChainMessageValidCase),
		new(sameChainMessageInvalidCase),
	}
	for _, c := range cases {
		c := c
		name := reflect.TypeOf(c).Elem().Name()
		gt.Run(name, func(gt *testing.T) {
			t := helpers.NewDefaultTesting(gt)
			system := dsl.NewInteropDSL(t)

			actors := system.Actors
			emitterContract := system.DeployEmitterContracts()

			chainAHook, chainBHook := c.Setup(t, system, emitterContract, actors)
			actors.ChainA.Sequencer.ActL2BuildBlock(t, chainAHook)
			actors.ChainB.Sequencer.ActL2BuildBlock(t, chainBHook)

			system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
				opts.SkipCrossSafeUpdate = true
			})

			endTimestamp := actors.ChainB.Sequencer.L2Unsafe().Time
			startTimestamp := endTimestamp - 1
			optimisticEnd := system.Outputs.SuperRoot(endTimestamp)

			preConsolidation := system.Outputs.TransitionState(startTimestamp, consolidateStep,
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
			).Marshal()

			// Induce block replacement
			system.ProcessCrossSafe()
			c.RunCrossSafeChecks(t, system, actors)
			crossSafeEnd := system.Outputs.SuperRoot(endTimestamp)
			optimisticIsCrossSafe := bytes.Equal(optimisticEnd.Marshal(), crossSafeEnd.Marshal())

			tests := []*transitionTest{
				{
					name:               "Consolidate",
					agreedClaim:        preConsolidation,
					disputedClaim:      crossSafeEnd.Marshal(),
					disputedTraceIndex: consolidateStep,
					expectValid:        true,
				},
				{
					name:               "Consolidate-InvalidNoChange",
					agreedClaim:        preConsolidation,
					disputedClaim:      preConsolidation,
					disputedTraceIndex: consolidateStep,
					expectValid:        false,
				},
			}
			if !optimisticIsCrossSafe {
				tests = append(tests, &transitionTest{
					name:               "Consolidate-ExpectInvalidPendingBlock",
					agreedClaim:        preConsolidation,
					disputedClaim:      optimisticEnd.Marshal(),
					disputedTraceIndex: consolidateStep,
					expectValid:        false,
				})
			}
			runFppAndChallengerTests(gt, system, tests)
		})
	}
}

func TestInteropFaultProofs_MessageExpiry(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t)

	actors := system.Actors
	alice := system.CreateUser()
	emitterContract := system.DeployEmitterContracts()
	system.AddL2Block(actors.ChainA, dsl.WithL2BlockTransactions(
		emitterContract.EmitMessage(alice, "test message"),
	))
	emitTx := emitterContract.LastEmittedMessage()

	// Bring ChainB to the same height and timestamp
	system.AddL2Block(actors.ChainB, dsl.WithL2BlocksUntilTimestamp(actors.ChainA.Sequencer.L2Unsafe().Time))
	system.SubmitBatchData()

	// Advance the chain until the init msg expires
	msgExpiryTime := system.DepSet().MessageExpiryWindow()
	end := emitTx.Identifier().Timestamp.Uint64() + msgExpiryTime
	system.AddL2Block(actors.ChainA, dsl.WithL2BlocksUntilTimestamp(end))
	system.AddL2Block(actors.ChainB, dsl.WithL2BlocksUntilTimestamp(end))
	system.SubmitBatchData()

	system.AddL2Block(actors.ChainB, func(opts *dsl.AddL2BlockOpts) {
		opts.TransactionCreators = []dsl.TransactionCreator{system.InboxContract.Execute(alice, emitTx)}
		opts.BlockIsNotCrossUnsafe = true
	})
	system.AddL2Block(actors.ChainA)

	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SkipCrossSafeUpdate = true
	})
	execTx := system.InboxContract.LastTransaction()
	execTx.CheckIncluded()

	endTimestamp := actors.ChainB.Sequencer.L2Unsafe().Time
	startTimestamp := endTimestamp - 1
	optimisticEnd := system.Outputs.SuperRoot(endTimestamp)

	preConsolidation := system.Outputs.TransitionState(startTimestamp, consolidateStep,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	// Induce block replacement
	system.ProcessCrossSafe()
	// assert that the invalid message txs were reorged out
	execTx.CheckNotIncluded()
	crossSafeEnd := system.Outputs.SuperRoot(endTimestamp)

	tests := []*transitionTest{
		{
			name:               "Consolidate-ExpectInvalidPendingBlock",
			agreedClaim:        preConsolidation,
			disputedClaim:      optimisticEnd.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        false,
		},
		{
			name:               "Consolidate-ReplaceInvalidBlocks",
			agreedClaim:        preConsolidation,
			disputedClaim:      crossSafeEnd.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
		},
	}
	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofsInvalidBlock(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)

	system := dsl.NewInteropDSL(t)

	actors := system.Actors
	alice := system.CreateUser()
	emitterContract := system.DeployEmitterContracts()

	system.AddL2Block(actors.ChainA, dsl.WithL2BlockTransactions(
		emitterContract.EmitMessage(alice, "test message"),
	))
	emitTx := emitterContract.LastEmittedMessage()

	// Bring ChainB to the same height and timestamp
	system.AddL2Block(actors.ChainB)
	system.SubmitBatchData()

	// Create a message with a conflicting payload
	fakeMessage := []byte("this message was never emitted")
	system.AddL2Block(actors.ChainB, func(opts *dsl.AddL2BlockOpts) {
		opts.TransactionCreators = []dsl.TransactionCreator{system.InboxContract.Execute(alice, emitTx, dsl.WithPayload(fakeMessage))}
		opts.BlockIsNotCrossUnsafe = true
	})
	system.AddL2Block(actors.ChainA)

	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SkipCrossSafeUpdate = true
	})

	execTx := system.InboxContract.LastTransaction()
	execTx.CheckIncluded()

	// safe head is still behind until we verify cross-safe
	assertHeads(t, actors.ChainA, 3, 3, 3, 2) // Chain A's block is cross unsafe
	assertHeads(t, actors.ChainB, 3, 3, 2, 2) // Chain B's block is not
	endTimestamp := actors.ChainB.Sequencer.L2Unsafe().Time

	startTimestamp := endTimestamp - 1
	start := system.Outputs.SuperRoot(startTimestamp)
	end := system.Outputs.SuperRoot(endTimestamp)

	step1Expected := system.Outputs.TransitionState(startTimestamp, 1,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
	).Marshal()

	// Capture optimistic blocks now before the invalid block is reorg'd out
	// Otherwise later calls to paddingStep would incorrectly use the deposit-only block
	allOptimisticBlocks := []types.OptimisticBlock{
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	}
	step2Expected := system.Outputs.TransitionState(startTimestamp, 2,
		allOptimisticBlocks...,
	).Marshal()

	paddingStep := func(step uint64) []byte {
		return system.Outputs.TransitionState(startTimestamp, step, allOptimisticBlocks...).Marshal()
	}

	// Induce block replacement
	system.ProcessCrossSafe()
	// assert that the invalid message tx was reorged out
	execTx.CheckNotIncluded()
	assertHeads(t, actors.ChainA, 3, 3, 3, 3)
	assertHeads(t, actors.ChainB, 3, 3, 3, 3)

	crossSafeSuperRootEnd := system.Outputs.SuperRoot(endTimestamp)

	tests := []*transitionTest{
		{
			name:               "FirstChainOptimisticBlock",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			disputedTraceIndex: 0,
			expectValid:        true,
		},
		{
			name:               "SecondChainOptimisticBlock",
			agreedClaim:        step1Expected,
			disputedClaim:      step2Expected,
			disputedTraceIndex: 1,
			expectValid:        true,
		},
		{
			name:               "FirstPaddingStep",
			agreedClaim:        step2Expected,
			disputedClaim:      paddingStep(3),
			disputedTraceIndex: 2,
			expectValid:        true,
		},
		{
			name:               "SecondPaddingStep",
			agreedClaim:        paddingStep(3),
			disputedClaim:      paddingStep(4),
			disputedTraceIndex: 3,
			expectValid:        true,
		},
		{
			name:               "LastPaddingStep",
			agreedClaim:        paddingStep(consolidateStep - 1),
			disputedClaim:      paddingStep(consolidateStep),
			disputedTraceIndex: consolidateStep - 1,
			expectValid:        true,
		},
		{
			name:               "Consolidate-ExpectInvalidPendingBlock",
			agreedClaim:        paddingStep(consolidateStep),
			disputedClaim:      end.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        false,
		},
		{
			name:               "Consolidate-ReplaceInvalidBlock",
			agreedClaim:        paddingStep(consolidateStep),
			disputedClaim:      crossSafeSuperRootEnd.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
		},
		{
			name:               "AlreadyAtClaimedTimestamp",
			agreedClaim:        crossSafeSuperRootEnd.Marshal(),
			disputedClaim:      crossSafeSuperRootEnd.Marshal(),
			disputedTraceIndex: 5000,
			expectValid:        true,
		},

		{
			name:               "FirstChainReachesL1Head",
			agreedClaim:        start.Marshal(),
			disputedClaim:      interop.InvalidTransition,
			disputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: true,
		},
		{
			name:               "SuperRootInvalidIfUnsupportedByL1Data",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			disputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: false,
		},
		{
			name:               "FromInvalidTransitionHash",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			disputedTraceIndex: 2,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: true,
		},
	}

	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs_VariedBlockTimes(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t, dsl.SetBlockTimeForChainA(1), dsl.SetBlockTimeForChainB(2))
	actors := system.Actors

	system.AdvanceSafeHeads()
	assertTime(t, actors.ChainA, 1, 1, 1, 1)
	assertTime(t, actors.ChainB, 2, 2, 2, 2)

	endTimestamp := actors.ChainA.Sequencer.L2Safe().Time
	startTimestamp := endTimestamp - 1

	start := system.Outputs.SuperRoot(startTimestamp)
	end := system.Outputs.SuperRoot(endTimestamp)
	l1Head := actors.L1Miner.L1Chain().CurrentBlock().Hash()

	step1Expected := system.Outputs.TransitionState(startTimestamp, 1,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
	).Marshal()

	step2Expected := system.Outputs.TransitionState(startTimestamp, 2,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	paddingStep := func(step uint64) []byte {
		return system.Outputs.TransitionState(startTimestamp, step,
			system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
			system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
		).Marshal()
	}

	// Add one more block on each chain to setup challenger test cases that fetch a super root that's past the end timestamp
	// This is necessary because on a 1-second block time, a new super root is created immediately after the end timestamp.
	system.AdvanceSafeHeads()

	tests := []*transitionTest{
		{
			name:               "ClaimDirectToNextTimestamp",
			agreedClaim:        start.Marshal(),
			disputedClaim:      end.Marshal(),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 0,
			expectValid:        false,
		},
		{
			name:               "FirstChainOptimisticBlock",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 0,
			expectValid:        true,
		},
		{
			name:               "FirstChainOptimisticBlock-InvalidNoChange",
			agreedClaim:        start.Marshal(),
			disputedClaim:      start.Marshal(),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 0,
			expectValid:        false,
		},
		{
			name:               "SecondChainOptimisticBlock",
			agreedClaim:        step1Expected,
			disputedClaim:      step2Expected,
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 1,
			expectValid:        true,
		},
		{
			name:               "SecondChainOptimisticBlock-InvalidNoChange",
			agreedClaim:        step1Expected,
			disputedClaim:      step1Expected,
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 1,
			expectValid:        false,
		},
		{
			name:               "FirstPaddingStep",
			agreedClaim:        step2Expected,
			disputedClaim:      paddingStep(3),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 2,
			expectValid:        true,
		},
		{
			name:               "FirstPaddingStep-InvalidNoChange",
			agreedClaim:        step2Expected,
			disputedClaim:      step2Expected,
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 2,
			expectValid:        false,
		},
		{
			name:               "SecondPaddingStep",
			agreedClaim:        paddingStep(3),
			disputedClaim:      paddingStep(4),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 3,
			expectValid:        true,
		},
		{
			name:               "SecondPaddingStep-InvalidNoChange",
			agreedClaim:        paddingStep(3),
			disputedClaim:      paddingStep(3),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 3,
			expectValid:        false,
		},
		{
			name:               "LastPaddingStep",
			agreedClaim:        paddingStep(consolidateStep - 1),
			disputedClaim:      paddingStep(consolidateStep),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: consolidateStep - 1,
			expectValid:        true,
		},
		{
			name:               "Consolidate",
			agreedClaim:        paddingStep(consolidateStep),
			disputedClaim:      end.Marshal(),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
		},
		{
			// The proposed block timestamp is after the unsafe head block timestamp.
			// With 1 second block time, we have reached the next block on chain A.
			// But the next pending block is past the chain A's safe head, so we expect the transition to be invalid
			name:               "DisputeTimestampAfterChainHeadChainA",
			agreedClaim:        end.Marshal(),
			l1Head:             l1Head,
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: consolidateStep + 1,
			expectValid:        true,
		},
		{
			name: "DisputeTimestampAfterChainHeadConsolidate",
			agreedClaim: system.Outputs.TransitionState(endTimestamp, consolidateStep,
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp+1),
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp+1),
			).Marshal(),
			disputedClaim:      system.Outputs.SuperRoot(endTimestamp + 1).Marshal(),
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 2*stepsPerTimestamp - 1,
			expectValid:        true,
		},
		{
			// With a 1 second block time on chain A, the implied agreed trace index references data past the l1 head.
			// So the prestate transition is invalid.
			name:        "DisputeBlockAfterChainHead-FirstChain",
			agreedClaim: interop.InvalidTransition,
			l1Head:      l1Head,
			// Timestamp has advanced enough to expect the next block now, but it doesn't exit so transition to invalid
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 2 * stepsPerTimestamp,
			expectValid:        true,
		},
		{
			// The agreed and disputed claim are both after the current chain head
			name:               "AgreedBlockAfterChainHead-Consolidate",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			l1Head:             l1Head,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 4*stepsPerTimestamp - 1,
			expectValid:        true,
		},
		{
			// The agreed and disputed claim are both after the current chain head and disputing an optimistic block
			name:               "AgreedBlockAfterChainHead-Optimistic",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 4*stepsPerTimestamp + 1,
			expectValid:        true,
		},

		{
			name:               "FirstChainReachesL1Head",
			agreedClaim:        start.Marshal(),
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
			disputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: true,
		},
		{
			// The transition from start to end timestamp only changes chain A, since it has a 1-second block time.
			// So although the L1 head doesn't contain any chain B data, the next state is still valid because the proposed timestamp is still covered by chain B's head
			name:               "SecondChainReachesL1Head",
			agreedClaim:        step1Expected,
			disputedClaim:      step2Expected,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
			disputedTraceIndex: 1,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().GetCanonicalHash(1),
			expectValid: true,
		},
		{
			name:               "SuperRootInvalidIfUnsupportedByL1Data",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
			disputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: false,
		},
		{
			name:               "FromInvalidTransitionHash",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
			disputedTraceIndex: 2,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: true,
		},
	}

	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs_VariedBlockTimes_FasterChainB(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t, dsl.SetBlockTimeForChainA(2), dsl.SetBlockTimeForChainB(1))
	actors := system.Actors

	system.AdvanceSafeHeads()
	assertTime(t, actors.ChainA, 2, 2, 2, 2)
	assertTime(t, actors.ChainB, 1, 1, 1, 1)

	endTimestamp := actors.ChainB.Sequencer.L2Safe().Time
	startTimestamp := endTimestamp - 1

	start := system.Outputs.SuperRoot(startTimestamp)
	end := system.Outputs.SuperRoot(endTimestamp)
	l1Head := actors.L1Miner.L1Chain().CurrentBlock().Hash()

	step1Expected := system.Outputs.TransitionState(startTimestamp, 1,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
	).Marshal()

	step2Expected := system.Outputs.TransitionState(startTimestamp, 2,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	paddingStep := func(step uint64) []byte {
		return system.Outputs.TransitionState(startTimestamp, step,
			system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
			system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
		).Marshal()
	}

	// Add one more block on each chain to setup challenger test cases that fetch a super root that's past the end timestamp
	// This is necessary because on a 1-second block time, a new super root is created immediately after the end timestamp.
	system.AdvanceSafeHeads()

	tests := []*transitionTest{
		{
			name:               "ClaimDirectToNextTimestamp",
			agreedClaim:        start.Marshal(),
			disputedClaim:      end.Marshal(),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 0,
			expectValid:        false,
		},
		{
			name:               "FirstChainOptimisticBlock",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 0,
			expectValid:        true,
		},
		{
			name:               "FirstChainOptimisticBlock-InvalidNoChange",
			agreedClaim:        start.Marshal(),
			disputedClaim:      start.Marshal(),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 0,
			expectValid:        false,
		},
		{
			name:               "SecondChainOptimisticBlock",
			agreedClaim:        step1Expected,
			disputedClaim:      step2Expected,
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 1,
			expectValid:        true,
		},
		{
			name:               "SecondChainOptimisticBlock-InvalidNoChange",
			agreedClaim:        step1Expected,
			disputedClaim:      step1Expected,
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 1,
			expectValid:        false,
		},
		{
			name:               "FirstPaddingStep",
			agreedClaim:        step2Expected,
			disputedClaim:      paddingStep(3),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 2,
			expectValid:        true,
		},
		{
			name:               "FirstPaddingStep-InvalidNoChange",
			agreedClaim:        step2Expected,
			disputedClaim:      step2Expected,
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 2,
			expectValid:        false,
		},
		{
			name:               "SecondPaddingStep",
			agreedClaim:        paddingStep(3),
			disputedClaim:      paddingStep(4),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 3,
			expectValid:        true,
		},
		{
			name:               "SecondPaddingStep-InvalidNoChange",
			agreedClaim:        paddingStep(3),
			disputedClaim:      paddingStep(3),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: 3,
			expectValid:        false,
		},
		{
			name:               "LastPaddingStep",
			agreedClaim:        paddingStep(consolidateStep - 1),
			disputedClaim:      paddingStep(consolidateStep),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: consolidateStep - 1,
			expectValid:        true,
		},
		{
			name:               "Consolidate",
			agreedClaim:        paddingStep(consolidateStep),
			disputedClaim:      end.Marshal(),
			startTimestamp:     startTimestamp,
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
		},
		{
			// The proposed block timestamp is after the unsafe head block timestamp.
			name:        "DisputeTimestampAfterChainHeadChainA",
			agreedClaim: end.Marshal(),
			l1Head:      l1Head,
			// With 2 second block times, we haven't yet reached the next block on the first chain so it's still valid
			disputedClaim: system.Outputs.TransitionState(endTimestamp, 1,
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp+1),
			).Marshal(),
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: consolidateStep + 1,
			expectValid:        true,
		},
		{
			name: "DisputeTimestampAfterChainHeadConsolidate",
			agreedClaim: system.Outputs.TransitionState(endTimestamp, consolidateStep,
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp+1),
				system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp+1),
			).Marshal(),
			disputedClaim:      system.Outputs.SuperRoot(endTimestamp + 1).Marshal(),
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 2*stepsPerTimestamp - 1,
			expectValid:        true,
		},
		{
			// With a 1 second block time on chain A, the implied agreed trace index references data past the l1 head.
			// So the prestate transition is invalid.
			name:        "DisputeBlockAfterChainHead-FirstChain",
			agreedClaim: interop.InvalidTransition,
			l1Head:      l1Head,
			// Timestamp has advanced enough to expect the next block now, but it doesn't exit so transition to invalid
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 2 * stepsPerTimestamp,
			expectValid:        true,
		},
		{
			// The agreed and disputed claim are both after the current chain head
			name:               "AgreedBlockAfterChainHead-Consolidate",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			l1Head:             l1Head,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 4*stepsPerTimestamp - 1,
			expectValid:        true,
		},
		{
			// The agreed and disputed claim are both after the current chain head and disputing an optimistic block
			name:               "AgreedBlockAfterChainHead-Optimistic",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			proposalTimestamp:  endTimestamp + 100,
			disputedTraceIndex: 4*stepsPerTimestamp + 1,
			expectValid:        true,
		},

		{
			// The transition from start to end timestamp only changes chain A, since it has a 1-second block time.
			// So although the L1 head doesn't contain any chain B data, the next state is still valid because the proposed timestamp is still covered by chain B's head
			name:               "FirstChainReachesL1Head",
			agreedClaim:        start.Marshal(),
			disputedClaim:      step1Expected,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
			disputedTraceIndex: 0,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: true,
		},
		{
			name:               "SecondChainReachesL1Head",
			agreedClaim:        step1Expected,
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
			disputedTraceIndex: 1,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().GetCanonicalHash(1),
			expectValid: true,
		},
		{
			name:               "FromInvalidTransitionHash",
			agreedClaim:        interop.InvalidTransition,
			disputedClaim:      interop.InvalidTransition,
			startTimestamp:     startTimestamp,
			proposalTimestamp:  endTimestamp,
			disputedTraceIndex: 2,
			// The derivation reaches the L1 head before the next block can be created
			l1Head:      actors.L1Miner.L1Chain().Genesis().Hash(),
			expectValid: true,
		},
	}

	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs_DepositMessage(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)

	system := dsl.NewInteropDSL(t)
	actors := system.Actors
	emitter := system.DeployEmitterContracts()

	// Advance L1 a couple times to avoid deposit gas metering issues near genesis
	system.AdvanceL1()
	system.AdvanceL1()

	l1User := system.CreateUser()
	depositMessage := dsl.NewMessage(system, actors.ChainA, emitter, "hello")
	system.AdvanceL1(
		dsl.WithActIncludeTx(
			depositMessage.ActEmitDeposit(l1User)))

	// As such, the next block timestamp across both chains will contain a user-deposit message and an executing message
	system.AdvanceL2ToLastBlockOfOrigin(actors.ChainA, 2)
	system.AdvanceL2ToLastBlockOfOrigin(actors.ChainB, 2)

	actors.ChainA.Sequencer.ActL2EmptyBlock(t)
	actors.ChainB.Sequencer.ActL2BuildBlock(t, func() {
		// The pending block on chain A will contain the user deposit
		depositMessage.ExecutePendingOn(actors.ChainB, actors.ChainA.Sequencer.L2Unsafe().Number)
	})
	system.SubmitBatchData(dsl.WithSkipCrossSafeUpdate())

	endTimestamp := actors.ChainB.Sequencer.L2Unsafe().Time
	startTimestamp := endTimestamp - 1
	preConsolidation := system.Outputs.TransitionState(startTimestamp, consolidateStep,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	system.ProcessCrossSafe()
	depositMessage.CheckExecuted()
	assertUserDepositEmitted(t, system.Actors.ChainA, nil, emitter)
	crossSafeEnd := system.Outputs.SuperRoot(endTimestamp)

	tests := []*transitionTest{
		{
			name:               "Consolidate",
			agreedClaim:        preConsolidation,
			disputedClaim:      crossSafeEnd.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
		},
		{
			name:               "Consolidate-InvalidNoChange",
			agreedClaim:        preConsolidation,
			disputedClaim:      preConsolidation,
			disputedTraceIndex: consolidateStep,
			expectValid:        false,
		},
	}
	runFppAndChallengerTests(gt, system, tests)
}

func TestInteropFaultProofs_DepositMessage_InvalidExecution(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)

	system := dsl.NewInteropDSL(t)
	actors := system.Actors
	emitter := system.DeployEmitterContracts()

	// Advance L1 a couple times to avoid deposit gas metering issues near genesis
	system.AdvanceL1()
	system.AdvanceL1()

	l1User := system.CreateUser()
	depositMessage := dsl.NewMessage(system, actors.ChainA, emitter, "hello")
	system.AdvanceL1(
		dsl.WithActIncludeTx(
			depositMessage.ActEmitDeposit(l1User)))

	// As such, the next block timestamp across both chains will contain a user-deposit message and an executing message
	system.AdvanceL2ToLastBlockOfOrigin(actors.ChainA, 2)
	system.AdvanceL2ToLastBlockOfOrigin(actors.ChainB, 2)

	actors.ChainA.Sequencer.ActL2EmptyBlock(t)
	actors.ChainB.Sequencer.ActL2BuildBlock(t, func() {
		// The pending block on chain A will contain the user deposit
		depositMessage.ExecutePendingOn(actors.ChainB,
			actors.ChainA.Sequencer.L2Unsafe().Number+1,
			dsl.WithPayload([]byte("this message was never emitted")),
		)
	})
	system.SubmitBatchData(dsl.WithSkipCrossSafeUpdate())

	endTimestamp := actors.ChainB.Sequencer.L2Unsafe().Time
	startTimestamp := endTimestamp - 1
	optimisticEnd := system.Outputs.SuperRoot(endTimestamp)

	preConsolidation := system.Outputs.TransitionState(startTimestamp, consolidateStep,
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainA, endTimestamp),
		system.Outputs.OptimisticBlockAtTimestamp(actors.ChainB, endTimestamp),
	).Marshal()

	system.ProcessCrossSafe()
	depositMessage.CheckNotExecuted()
	assertUserDepositEmitted(t, system.Actors.ChainA, nil, emitter)
	crossSafeEnd := system.Outputs.SuperRoot(endTimestamp)

	tests := []*transitionTest{
		{
			name:               "Consolidate",
			agreedClaim:        preConsolidation,
			disputedClaim:      crossSafeEnd.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
		},
		{
			name:               "Consolidate-InvalidNoChange",
			agreedClaim:        preConsolidation,
			disputedClaim:      preConsolidation,
			disputedTraceIndex: consolidateStep,
			expectValid:        false,
		},
	}
	tests = append(tests, &transitionTest{
		name:               "Consolidate-ExpectInvalidPendingBlock",
		agreedClaim:        preConsolidation,
		disputedClaim:      optimisticEnd.Marshal(),
		disputedTraceIndex: consolidateStep,
		expectValid:        false,
	})
	runFppAndChallengerTests(gt, system, tests)
}

// Returns true if all tests passed, otherwise returns false
func runFppAndChallengerTests(gt *testing.T, system *dsl.InteropDSL, tests []*transitionTest) bool {
	passed := true
	for _, test := range tests {
		test := test
		passed = gt.Run(fmt.Sprintf("%s-fpp", test.name), func(gt *testing.T) {
			runFppTest(gt, test, system.Actors, system.DepSet())
		}) && passed

		passed = gt.Run(fmt.Sprintf("%s-challenger", test.name), func(gt *testing.T) {
			runChallengerTest(gt, test, system.Actors)
		}) && passed
	}
	return passed
}

func runFppTest(gt *testing.T, test *transitionTest, actors *dsl.InteropActors, depSet *depset.StaticConfigDependencySet) {
	t := helpers.SubTest(gt)
	if test.skipProgram {
		t.Skip("Not yet implemented")
		return
	}
	logger := testlog.Logger(t, slog.LevelInfo)
	checkResult := fpHelpers.ExpectNoError()
	if !test.expectValid {
		checkResult = fpHelpers.ExpectError(claim.ErrClaimNotValid)
	}
	l1Head := test.l1Head
	if l1Head == (common.Hash{}) {
		l1Head = actors.L1Miner.L1Chain().CurrentBlock().Hash()
	}
	proposalTimestamp := test.proposalTimestamp
	if proposalTimestamp == 0 {
		proposalTimestamp = actors.ChainA.Sequencer.L2Unsafe().Time
	}
	fpHelpers.RunFaultProofProgram(
		t,
		logger,
		actors.L1Miner,
		checkResult,
		WithInteropEnabled(t, actors, depSet, test.agreedClaim, crypto.Keccak256Hash(test.disputedClaim), proposalTimestamp),
		fpHelpers.WithL1Head(l1Head),
	)
}

func runChallengerTest(gt *testing.T, test *transitionTest, actors *dsl.InteropActors) {
	t := helpers.SubTest(gt)
	if test.skipChallenger {
		t.Skip("Not yet implemented")
		return
	}
	logger := testlog.Logger(t, slog.LevelInfo)
	endTimestamp := test.proposalTimestamp
	if endTimestamp == 0 {
		endTimestamp = actors.ChainA.Sequencer.L2Unsafe().Time
	}
	startTimestamp := test.startTimestamp
	if startTimestamp == 0 {
		startTimestamp = actors.ChainA.Sequencer.L2Unsafe().Time - 1
	}
	prestateProvider := super.NewSuperRootPrestateProvider(actors.Supervisor, startTimestamp)
	var l1Head eth.BlockID
	if test.l1Head == (common.Hash{}) {
		l1Head = eth.ToBlockID(eth.HeaderBlockInfo(actors.L1Miner.L1Chain().CurrentBlock()))
	} else {
		l1Head = eth.ToBlockID(actors.L1Miner.L1Chain().GetBlockByHash(test.l1Head))
	}
	gameDepth := challengerTypes.Depth(30)
	rollupCfgs, err := super.NewRollupConfigsFromParsed(actors.ChainA.RollupCfg, actors.ChainB.RollupCfg)
	require.NoError(t, err)
	provider := super.NewSuperTraceProvider(logger, rollupCfgs, prestateProvider, actors.Supervisor, l1Head, gameDepth, startTimestamp, endTimestamp)
	var agreedPrestate []byte
	if test.disputedTraceIndex > 0 {
		agreedPrestate, err = provider.GetPreimageBytes(t.Ctx(), challengerTypes.NewPosition(gameDepth, big.NewInt(test.disputedTraceIndex-1)))
		require.NoError(t, err)
	} else {
		superRoot, err := provider.AbsolutePreState(t.Ctx())
		require.NoError(t, err)
		agreedPrestate = superRoot.Marshal()
	}
	require.Equal(t, test.agreedClaim, agreedPrestate, "agreed prestate mismatch")

	disputedClaim, err := provider.GetPreimageBytes(t.Ctx(), challengerTypes.NewPosition(gameDepth, big.NewInt(test.disputedTraceIndex)))
	require.NoError(t, err)
	if test.expectValid {
		require.Equal(t, test.disputedClaim, disputedClaim, "Claim is correct so should match challenger's opinion")
	} else {
		require.NotEqual(t, test.disputedClaim, disputedClaim, "Claim is incorrect so should not match challenger's opinion")
	}
}

func WithInteropEnabled(t helpers.StatefulTesting, actors *dsl.InteropActors, depSet *depset.StaticConfigDependencySet, agreedPrestate []byte, disputedClaim common.Hash, claimTimestamp uint64) fpHelpers.FixtureInputParam {
	return func(f *fpHelpers.FixtureInputs) {
		f.InteropEnabled = true
		f.AgreedPrestate = agreedPrestate
		f.L2OutputRoot = crypto.Keccak256Hash(agreedPrestate)
		f.L2Claim = disputedClaim
		f.L2BlockNumber = claimTimestamp
		f.DependencySet = depSet

		for _, chain := range []*dsl.Chain{actors.ChainA, actors.ChainB} {
			verifier, canonicalOnlyEngine := createVerifierWithOnlyCanonicalBlocks(t, actors.L1Miner, chain)
			f.L2Sources = append(f.L2Sources, &fpHelpers.FaultProofProgramL2Source{
				Node:        verifier,
				Engine:      canonicalOnlyEngine,
				ChainConfig: chain.L2Genesis.Config,
			})
		}
	}
}

// createVerifierWithOnlyCanonicalBlocks creates a new L2Verifier and associated L2Engine that only has the canonical
// blocks from chain in its database. Non-canonical blocks, their world state, receipts and other data are not available
func createVerifierWithOnlyCanonicalBlocks(t helpers.StatefulTesting, l1Miner *helpers.L1Miner, chain *dsl.Chain) (*helpers.L2Verifier, *helpers.L2Engine) {
	jwtPath := e2eutils.WriteDefaultJWT(t)
	canonicalOnlyEngine := helpers.NewL2Engine(t, testlog.Logger(t, log.LvlInfo).New("role", "canonicalOnlyEngine"), chain.L2Genesis, jwtPath)
	head := chain.Sequencer.L2Unsafe()
	for i := uint64(1); i <= head.Number; i++ {
		block, err := chain.SequencerEngine.EthClient().BlockByNumber(t.Ctx(), new(big.Int).SetUint64(i))
		require.NoErrorf(t, err, "failed to get block by number %v", i)
		envelope, err := eth.BlockAsPayloadEnv(block, chain.L2Genesis.Config)
		require.NoErrorf(t, err, "could not convert block %v to payload envelope")
		result, err := canonicalOnlyEngine.EngineApi.NewPayloadV4(t.Ctx(), envelope.ExecutionPayload, []common.Hash{}, envelope.ParentBeaconBlockRoot, []hexutil.Bytes{})
		require.NoErrorf(t, err, "could not import payload for block %v", i)
		require.Equal(t, eth.ExecutionValid, result.Status)
	}
	fcuResult, err := canonicalOnlyEngine.EngineApi.ForkchoiceUpdatedV3(t.Ctx(), &eth.ForkchoiceState{
		HeadBlockHash:      head.Hash,
		SafeBlockHash:      head.Hash,
		FinalizedBlockHash: head.Hash,
	}, nil)
	require.NoErrorf(t, err, "could not update fork choice for block %v", head.Hash)
	require.Equal(t, eth.ExecutionValid, fcuResult.PayloadStatus.Status)

	// Verify chain matches exactly
	for i := uint64(0); i <= head.Number; i++ {
		blockNum := new(big.Int).SetUint64(i)
		expected, err := chain.SequencerEngine.EthClient().BlockByNumber(t.Ctx(), blockNum)
		require.NoErrorf(t, err, "failed to get original block by number %v", i)
		actual, err := canonicalOnlyEngine.EthClient().BlockByNumber(t.Ctx(), blockNum)
		require.NoErrorf(t, err, "failed to get canonical-only block by number %v", i)
		require.Equalf(t, expected.Hash(), actual.Hash(), "block %v does not match", i)
	}

	verifier := helpers.NewL2Verifier(t,
		testlog.Logger(t, log.LvlInfo).New("role", "canonicalOnlyVerifier"),
		l1Miner.L1Client(t, chain.RollupCfg),
		l1Miner.BlobSource(),
		altda.Disabled,
		canonicalOnlyEngine.EngineClient(t, chain.RollupCfg),
		chain.RollupCfg,
		chain.DependencySet,
		&sync2.Config{},
		safedb.Disabled)
	return verifier, canonicalOnlyEngine
}

func assertTime(t helpers.Testing, chain *dsl.Chain, unsafe, crossUnsafe, localSafe, safe uint64) {
	start := chain.L2Genesis.Timestamp
	status := chain.Sequencer.SyncStatus()
	require.Equal(t, start+unsafe, status.UnsafeL2.Time, "Unsafe")
	require.Equal(t, start+crossUnsafe, status.CrossUnsafeL2.Time, "Cross Unsafe")
	require.Equal(t, start+localSafe, status.LocalSafeL2.Time, "Local safe")
	require.Equal(t, start+safe, status.SafeL2.Time, "Safe")
}

func assertUserDepositEmitted(t helpers.Testing, chain *dsl.Chain, number *big.Int, emitter *dsl.EmitterContract) {
	block, err := chain.SequencerEngine.EthClient().BlockByNumber(t.Ctx(), number)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(block.Transactions()), 2) // l1-attrs + user-deposit + [txs]
	userDepositTx := block.Transactions()[1]
	require.NotNil(t, userDepositTx.To())
	require.Equal(t, emitter.Address(chain), *userDepositTx.To())
}

type transitionTest struct {
	name               string
	agreedClaim        []byte
	disputedClaim      []byte
	disputedTraceIndex int64
	l1Head             common.Hash // Defaults to current L1 head if not set
	startTimestamp     uint64      // Defaults to latest L2 block timestamp - 1 if 0
	proposalTimestamp  uint64      // Defaults to latest L2 block timestamp if 0
	expectValid        bool
	skipProgram        bool
	skipChallenger     bool
}

type intraBlockTestCase interface {
	// Setup is called to create a single-block test scenario
	Setup(t helpers.StatefulTesting, system *dsl.InteropDSL, emitterContract *dsl.EmitterContract, actors *dsl.InteropActors) (chainAHook func(), chainBHook func())
	// RunCrossSafeChecks is called after cross-safe updates are applied to the system
	RunCrossSafeChecks(t helpers.StatefulTesting, system *dsl.InteropDSL, actors *dsl.InteropActors)
}

type cascadeInvalidBlockCase struct {
	msgA *dsl.Message
	msgB *dsl.Message
}

var _ intraBlockTestCase = (*cascadeInvalidBlockCase)(nil)

func (c *cascadeInvalidBlockCase) Setup(t helpers.StatefulTesting, system *dsl.InteropDSL, emitter *dsl.EmitterContract, actors *dsl.InteropActors) (chainAHook func(), chainBHook func()) {
	return func() {
			// Chain A build: emit A and schedule exec of B on A referencing B's pending message
			c.msgA = dsl.NewMessage(system, actors.ChainA, emitter, "chainA message").
				Emit()
			c.msgB = dsl.NewMessage(system, actors.ChainB, emitter, "chainB message")
			pendingBBlock := actors.ChainB.Sequencer.L2Unsafe().Number + 1
			c.msgB.ExecutePendingOn(actors.ChainA, pendingBBlock)
		}, func() {
			// Chain B build: emit B and include the invalid exec of A on B
			c.msgB.Emit()
			c.msgA.ExecuteOn(actors.ChainB, dsl.WithPayload([]byte("this message was never emitted")))
		}
}

func (c *cascadeInvalidBlockCase) RunCrossSafeChecks(t helpers.StatefulTesting, system *dsl.InteropDSL, actors *dsl.InteropActors) {
	c.msgA.CheckNotEmitted()
	c.msgA.CheckNotExecuted()
	c.msgB.CheckNotEmitted()
	c.msgB.CheckNotExecuted()
}

type swapCascadeInvalidBlockCase struct {
	cascadeInvalidBlockCase
}

var _ intraBlockTestCase = (*swapCascadeInvalidBlockCase)(nil)

func (c *swapCascadeInvalidBlockCase) Setup(t helpers.StatefulTesting, system *dsl.InteropDSL, emitter *dsl.EmitterContract, actors *dsl.InteropActors) (chainAHook func(), chainBHook func()) {
	c.msgA = dsl.NewMessage(system, actors.ChainA, emitter, "chainA message")
	c.msgB = dsl.NewMessage(system, actors.ChainB, emitter, "chainB message")
	return func() {
			// Chain A build: emit A then include invalid exec of B on A referencing B's pending message
			c.msgA.Emit()
			c.msgB.ExecutePendingOn(actors.ChainA, actors.ChainB.Sequencer.L2Unsafe().Number+1, dsl.WithPayload([]byte("this message was never emitted")))
		}, func() {
			// Chain B build: emit B to materialize the pending message
			c.msgB.Emit()
		}
}

func (c *swapCascadeInvalidBlockCase) RunCrossSafeChecks(t helpers.StatefulTesting, system *dsl.InteropDSL, actors *dsl.InteropActors) {
	// Chain A block with invalid exec is replaced, so msgA emission is reorged out
	c.msgA.CheckNotEmitted()
	// On Chain B, the emission of msgB remains, but the invalid exec on A is reorged out
	c.msgB.CheckEmitted()
	c.msgB.CheckNotExecuted()
}

type cyclicDependencyValidCase struct {
	msgA *dsl.Message
	msgB *dsl.Message
}

var _ intraBlockTestCase = (*cyclicDependencyValidCase)(nil)

func (c *cyclicDependencyValidCase) Setup(t helpers.StatefulTesting, system *dsl.InteropDSL, emitter *dsl.EmitterContract, actors *dsl.InteropActors) (chainAHook func(), chainBHook func()) {
	c.msgA = dsl.NewMessage(system, actors.ChainA, emitter, "hello")
	c.msgB = dsl.NewMessage(system, actors.ChainB, emitter, "world")
	return func() {
			// Emit A, and schedule exec of B on A referencing B's pending message
			c.msgA.Emit()
			pendingBBlock := actors.ChainB.Sequencer.L2Unsafe().Number + 1
			c.msgB.ExecutePendingOn(actors.ChainA, pendingBBlock)
		}, func() {
			// Emit B, and execute A on B (A already emitted)
			c.msgB.Emit()
			c.msgA.ExecuteOn(actors.ChainB)
		}
}

func (c *cyclicDependencyValidCase) RunCrossSafeChecks(t helpers.StatefulTesting, system *dsl.InteropDSL, actors *dsl.InteropActors) {
	assertHeads(t, actors.ChainA, 2, 2, 2, 2)
	assertHeads(t, actors.ChainB, 2, 2, 2, 2)
	c.msgA.CheckEmitted()
	c.msgB.CheckEmitted()
	c.msgA.CheckExecuted()
	c.msgB.CheckExecuted()
}

type cyclicDependencyInvalidCase struct {
	execATx *dsl.GeneratedTransaction
	execBTx *dsl.GeneratedTransaction
}

var _ intraBlockTestCase = (*cyclicDependencyInvalidCase)(nil)

func (c *cyclicDependencyInvalidCase) Setup(t helpers.StatefulTesting, system *dsl.InteropDSL, emitter *dsl.EmitterContract, actors *dsl.InteropActors) (func(), func()) {
	alice := system.CreateUser()

	// Plan cyclic dependency using pending identifiers on both chains (no included init tx passed)
	pendingB := actors.ChainB.Sequencer.L2Unsafe().Number + 1
	pendingA := actors.ChainA.Sequencer.L2Unsafe().Number + 1
	execAOpts := dsl.WithPendingMessage(emitter, actors.ChainB, pendingB, 0, "message from B")
	execBOpts := dsl.WithPendingMessage(emitter, actors.ChainA, pendingA, 0, "message from A")
	actExecA := system.InboxContract.Execute(alice, nil, execAOpts)
	actExecB := system.InboxContract.Execute(alice, nil, execBOpts)
	c.execATx = actExecA(actors.ChainA)
	c.execBTx = actExecB(actors.ChainB)

	return func() {
			c.execATx.IncludeOK()
		}, func() {
			c.execBTx.IncludeOK()
		}
}

func (c *cyclicDependencyInvalidCase) RunCrossSafeChecks(t helpers.StatefulTesting, system *dsl.InteropDSL, actors *dsl.InteropActors) {
	c.execATx.CheckNotIncluded()
	c.execBTx.CheckNotIncluded()
}

type longDependencyChainValidCase struct {
	initTxA *dsl.GeneratedTransaction
	execs   []*dsl.GeneratedTransaction
}

var _ intraBlockTestCase = (*longDependencyChainValidCase)(nil)

func (c *longDependencyChainValidCase) Setup(t helpers.StatefulTesting, system *dsl.InteropDSL, emitter *dsl.EmitterContract, actors *dsl.InteropActors) (func(), func()) {
	alice := system.CreateUser()
	const depth = 10

	// Build locally in each hook to avoid cross-chain inclusion during a single build.
	return func() {
			// Chain A build: emit message and include half of the execs on A referencing the init
			initTxA := emitter.EmitMessage(alice, "chain A")(actors.ChainA)
			initTxA.IncludeOK()
			c.initTxA = initTxA
			var execs []*dsl.GeneratedTransaction
			for i := 0; i < depth/2; i++ {
				tx := system.InboxContract.Execute(alice, initTxA)(actors.ChainA)
				tx.IncludeOK()
				execs = append(execs, tx)
			}
			c.execs = append(c.execs, execs...)
		}, func() {
			// Chain B build: include the remaining execs on B referencing the init on A
			// The init was included in the Chain A hook already.
			initRef := c.initTxA
			var execs []*dsl.GeneratedTransaction
			for i := 0; i < depth/2; i++ {
				tx := system.InboxContract.Execute(alice, initRef)(actors.ChainB)
				tx.IncludeOK()
				execs = append(execs, tx)
			}
			c.execs = append(c.execs, execs...)
		}
}

func (c *longDependencyChainValidCase) RunCrossSafeChecks(t helpers.StatefulTesting, system *dsl.InteropDSL, actors *dsl.InteropActors) {
	for _, exec := range c.execs {
		exec.CheckIncluded()
	}
	c.initTxA.CheckIncluded()
}

type sameChainMessageValidCase struct {
	msg *dsl.Message
}

var _ intraBlockTestCase = (*sameChainMessageValidCase)(nil)

func (c *sameChainMessageValidCase) Setup(t helpers.StatefulTesting, system *dsl.InteropDSL, emitter *dsl.EmitterContract, actors *dsl.InteropActors) (func(), func()) {
	msg := dsl.NewMessage(system, actors.ChainA, emitter, "hello")
	c.msg = msg
	return func() {
		msg.Emit()
		msg.ExecuteOn(actors.ChainA)
	}, func() {}
}

func (c *sameChainMessageValidCase) RunCrossSafeChecks(t helpers.StatefulTesting, system *dsl.InteropDSL, actors *dsl.InteropActors) {
	c.msg.CheckEmitted()
	c.msg.CheckExecuted()
}

var _ intraBlockTestCase = (*sameChainMessageInvalidCase)(nil)

type sameChainMessageInvalidCase struct {
	msg *dsl.Message
}

func (c *sameChainMessageInvalidCase) Setup(t helpers.StatefulTesting, system *dsl.InteropDSL, emitter *dsl.EmitterContract, actors *dsl.InteropActors) (func(), func()) {
	msg := dsl.NewMessage(system, actors.ChainA, emitter, "hello")
	c.msg = msg
	return func() {
		msg.Emit()
		msg.ExecuteOn(actors.ChainA, dsl.WithPayload([]byte("this message was never emitted")))
	}, func() {}
}

func (c *sameChainMessageInvalidCase) RunCrossSafeChecks(t helpers.StatefulTesting, system *dsl.InteropDSL, actors *dsl.InteropActors) {
	c.msg.CheckNotEmitted()
	c.msg.CheckNotExecuted()
}
