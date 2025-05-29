package interop

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	actionhelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"
)

func TestInteropUpgrade(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)

	// Run Isthmus until the interop upgrade
	interopActivationOffset := 3
	system := dsl.NewInteropDSL(t, dsl.SetInteropOffsetForAllL2s(uint64(interopActivationOffset)))

	actors := system.Actors
	chainA := actors.ChainA
	chainB := actors.ChainB
	svr := actors.Supervisor
	chains := []*dsl.Chain{chainA, chainB}
	superchainSyncAsserter := dsl.NewSuperchainSyncStatusAsserter(t, system, chains, true)
	var genesisTime uint64

	////////////////////////////
	// Before Interop Upgrade
	////////////////////////////

	superchainSyncAsserter.RequireAllInitialSeqSyncStatuses(dsl.WithUnsafeEquals(0), dsl.WithCrossUnsafeEquals(0), dsl.WithLocalSafeEquals(0), dsl.WithSafeEquals(0), dsl.WithFinalizedEquals(0))

	for _, c := range chains {
		// Sanity check: chain interop activation time was correctly configured
		genesisTime = c.RollupCfg.Genesis.L2Time
		require.Equal(t, uint64(interopActivationOffset), *c.RollupCfg.InteropTime-genesisTime, "interop activation offset does not match configured expectation")

		// Sanity check: chain is at genesis block and interop is not activated according to config and unsafe head timestamp
		dsl.RequireUnsafeTimeOffset(t, c, 0) // Interop not be enabled

		syncAsserter := superchainSyncAsserter.ChainAsserters[c.ChainID]
		syncAsserter.RequireSeqSyncStatus(func() {
			// Advance unsafe head by one
			system.AddL2Block(c)
		}, dsl.WithUnsafeAdvancesTo(1), dsl.WithCrossUnsafeAdvancesTo(1))

		// Sanity check system state
		dsl.RequireUnsafeTimeOffset(t, c, 2) // Interop still not enabled
	}

	// Settle both chains to DA and assert nothing changed
	superchainSyncAsserter.RequireAllSeqSyncStatuses(func() {
		system.Actors.ActBatchAndMine(t, dsl.WithMarkFinal())
		dsl.RequireL1Heads(t, system, 2, 1)
	})

	// Still pre interop
	for _, c := range chains {
		syncAsserter := superchainSyncAsserter.ChainAsserters[c.ChainID]

		// Verify promotion of crossUnsafe, localSafe and safe
		syncAsserter.RequireSeqSyncStatus(func() {
			system.ActSyncSupernode(t, dsl.WithChains(c), dsl.WithLatestSignal())
		}, dsl.WithLocalSafeAdvancesTo(1), dsl.WithSafeAdvancesTo(1)) // Assert unsafe head, crossUnsafe head does not move, but update localSafe and safe heads

		syncAsserter.RequireSeqSyncStatus(func() {
			c.Sequencer.ActL1FinalizedSignal(t)
			c.Sequencer.ActL2PipelineFull(t)
		}, dsl.WithFinalizedAdvancesTo(1)) // assert unsafe head does not move, but update finalized head

		// Interop
		syncAsserter.RequireSeqSyncStatus(func() {
			// Build another L2 block so that Interop activates
			system.AddL2Block(c, dsl.WithL2BlocksUntilTimestamp(*c.Sequencer.RollupCfg.InteropTime))
		}, dsl.WithUnsafeAdvancesTo(2), dsl.WithCrossUnsafeAdvancesTo(2)) // assert unsafe and crossUnsafe advance by one

		////////////////////////////
		// After Interop Upgrade (for the current chain in iteration)
		////////////////////////////

		dsl.RequireUnsafeTimeOffset(t, c, 4) // interop should be enabled
		// Assert for the first time on supervisor heads
		dsl.RequireSupervisorChainHeads(t, svr, c, syncAsserter.PrevStatus.UnsafeL2.ID(), syncAsserter.PrevStatus.CrossUnsafeL2.ID(), syncAsserter.PrevStatus.LocalSafeL2.ID(), syncAsserter.PrevStatus.SafeL2.ID(), eth.BlockID{Hash: common.BytesToHash([]byte{}), Number: 0})

		// TODO(15863): The initial state of the finalized head differs between supervisor
		// and sequencer. In an action test setup, the sequencer considers
		// genesis head as finalized from the start. The supervisor however sets
		// the finalized head to a nil value until it it set. Seems worth fixing.
		// The following will fail.
		// system.ActSyncSupernode(t, dsl.WithChains(c), dsl.WithFinalizedSignal(), dsl.WithLatestSignal(), dsl.WithRequireFinalizedAdvances())
		// syncAsserter.RequireSupChainHeadsBySyncStatus()
	}

	// Settle both chains to DA again and have supervisor ingest
	system.Actors.ActBatchAndMine(t)

	// Verify that L1 heads actually updated
	dsl.RequireL1Heads(t, system, 3, 1)

	// // TODO: Why doesn't the cross head promotion happen here?
	superchainSyncAsserter.RequireAllSeqSyncStatuses(func() {
		system.ActSyncSupernode(t, dsl.WithLatestSignal())
	}, dsl.WithMapChainAssertions(dsl.WithLocalSafeAdvancesTo(2)))

	superchainSyncAsserter.RequireAllSeqSyncStatuses(func() {
		system.ActSyncSupernode(t)
	}, dsl.WithMapChainAssertions(dsl.WithSafeAdvancesTo(2)))

	// Note that finalized remains nil
	for _, c := range chains {
		syncAsserter := superchainSyncAsserter.ChainAsserters[c.ChainID]
		dsl.RequireSupervisorChainHeads(t, svr, c, syncAsserter.PrevStatus.UnsafeL2.ID(), syncAsserter.PrevStatus.CrossUnsafeL2.ID(), syncAsserter.PrevStatus.LocalSafeL2.ID(), syncAsserter.PrevStatus.SafeL2.ID(), eth.BlockID{Hash: common.BytesToHash([]byte{}), Number: 0})
	}

	// Verify proofs agree
	assertProgramOutputMatchesDerivationForBlockTimestamp(gt, system, system.Actors.ChainA.Sequencer.L2Unsafe().Time)

	superchainSyncAsserter.RequireAllSeqSyncStatuses(func() {
		// Advance L1 safe head and finalized head
		system.Actors.L1Miner.ActL1SafeNext(t)
		system.Actors.L1Miner.ActL1FinalizeNext(t)
		system.Actors.L1Miner.ActL1SafeNext(t)
		system.Actors.L1Miner.ActL1FinalizeNext(t)

		dsl.RequireL1Heads(t, system, 3, 3)

		system.ActSyncSupernode(t, dsl.WithLatestSignal(), dsl.WithFinalizedSignal())
	}, dsl.WithMapChainAssertions(dsl.WithFinalizedAdvancesTo(2)))

	for _, syncAsserter := range superchainSyncAsserter.ChainAsserters {
		syncAsserter.RequireSupChainHeadsBySyncStatus()
	}

	// Verify proofs agree on prev blocks
	assertProgramOutputMatchesDerivationForBlockTimestamp(gt, system, system.Actors.ChainA.Sequencer.L2Safe().Time)

	for _, c := range chains {
		// Advance unsafe head again
		syncAsserter := superchainSyncAsserter.ChainAsserters[c.ChainID]
		syncAsserter.RequireSeqSyncStatus(func() {
			system.AddL2Block(c)
		}, dsl.WithUnsafeAdvancesTo(3))
	}

	// Settle both chains to DA
	system.Actors.ActBatchAndMine(t)
	dsl.RequireL1Heads(t, system, 4, 3)

	for _, c := range chains {
		syncAsserter := superchainSyncAsserter.ChainAsserters[c.ChainID]
		syncAsserter.RequireSeqSyncStatus(func() {
			// Promote localSafe
			system.ActSyncSupernode(t, dsl.WithChains(c), dsl.WithLatestSignal())
		}, dsl.WithLocalSafeAdvancesTo(3))

		// Promote safe
		syncAsserter.RequireSeqSyncStatus(func() {
			system.ActSyncSupernode(t, dsl.WithChains(c))
		}, dsl.WithSafeAdvancesTo(3))
	}

	// Verify proofs agree on prev blocks
	assertProgramOutputMatchesDerivationForBlockTimestamp(gt, system, system.Actors.ChainA.Sequencer.L2Safe().Time)

	////////////////////////////
	// Send cross-chain message
	////////////////////////////

	// Prepare to send a message
	alice := system.CreateUser()
	emitter := system.DeployEmitterContracts()

	// Verify chain time
	dsl.RequireUnsafeTimeOffset(t, system.Actors.ChainA, 8)
	dsl.RequireUnsafeTimeOffset(t, system.Actors.ChainB, 8)

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})
	dsl.RequireL1Heads(t, system, 5, 3)

	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainB)
	})
	dsl.RequireL1Heads(t, system, 6, 3)

	// Verify proofs agree on prev blocks
	assertProgramOutputMatchesDerivationForBlockTimestamp(gt, system, system.Actors.ChainA.Sequencer.L2Safe().Time)

	// Send a message
	system.AddL2Block(system.Actors.ChainA, dsl.WithL2BlockTransactions(emitter.EmitMessage(alice, "hello")))
	initMsg := emitter.LastEmittedMessage()
	system.AddL2Block(system.Actors.ChainB, dsl.WithL2BlockTransactions(system.InboxContract.Execute(alice, initMsg)))

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})
	dsl.RequireL1Heads(t, system, 7, 3)
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainB)
	})
	dsl.RequireL1Heads(t, system, 8, 3)

	// Verify chain time
	dsl.RequireUnsafeTimeOffset(t, system.Actors.ChainA, 10)
	dsl.RequireUnsafeTimeOffset(t, system.Actors.ChainB, 10)

	// Verify proofs agree on prev blocks
	assertProgramOutputMatchesDerivationForBlockTimestamp(gt, system, system.Actors.ChainA.Sequencer.L2Safe().Time)
}

func VerifyContractsDeployedCorrectly(t helpers.Testing, chain *dsl.Chain, activationBlockTxs []*types.Transaction, activationBlockID eth.BlockID) {
	require.Len(t, activationBlockTxs, 5) // 4 upgrade txs + 1 system deposit tx
	upgradeTransactions := activationBlockTxs[1:]
	upgradeTransactionBytes := make([]hexutil.Bytes, len(upgradeTransactions))
	for i, tx := range upgradeTransactions {
		txBytes, err := tx.MarshalBinary()
		require.NoError(t, err)
		upgradeTransactionBytes[i] = txBytes
	}

	expectedUpgradeTransactions, err := derive.InteropNetworkUpgradeTransactions()
	require.NoError(t, err)

	require.Equal(t, upgradeTransactionBytes, expectedUpgradeTransactions)

	RequireContractDeployedAndProxyUpdated(t, chain, derive.CrossL2InboxAddress, predeploys.CrossL2InboxAddr, new(big.Int).SetUint64(activationBlockID.Number))
	RequireContractDeployedAndProxyUpdated(t, chain, derive.L2ToL2MessengerAddress, predeploys.L2toL2CrossDomainMessengerAddr, new(big.Int).SetUint64(activationBlockID.Number))
}

// Runs superchain fault proofs across entire superchain pseudo-blocks
func assertProgramOutputMatchesDerivationForBlockTimestamp(gt *testing.T, system *dsl.InteropDSL, endTimestamp uint64) {
	// Verify the proofs backend agrees
	actors := system.Actors

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

	// Fail immediately if the test does not pass
	require.True(gt, runFppAndChallengerTests(gt, system, []*transitionTest{
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
			name:               "Consolidate",
			agreedClaim:        paddingStep(consolidateStep),
			disputedClaim:      end.Marshal(),
			disputedTraceIndex: consolidateStep,
			expectValid:        true,
		},
	}), "fault proof program verification failed")
}

var ProxyImplGetterFunc = w3.MustNewFunc(`implementation()`, `address`)

func RequireContractDeployedAndProxyUpdated(t actionhelpers.Testing, chain *dsl.Chain, implAddr common.Address, proxyAddress common.Address, activationBlockNumber *big.Int) {
	code, err := chain.SequencerEngine.EthClient().CodeAt(t.Ctx(), implAddr, activationBlockNumber)
	require.NoError(t, err)
	require.NotEmpty(t, code, "contract should be deployed")
	selector, err := ProxyImplGetterFunc.EncodeArgs()
	require.NoError(t, err)
	implAddrBytes, err := chain.SequencerEngine.EthClient().CallContract(t.Ctx(), ethereum.CallMsg{
		To:   &proxyAddress,
		Data: selector,
	}, activationBlockNumber)
	require.NoError(t, err)
	var implAddrActual common.Address
	err = ProxyImplGetterFunc.DecodeReturns(implAddrBytes, &implAddrActual)
	require.NoError(t, err)
	require.Equal(t, implAddr, implAddrActual)
}
