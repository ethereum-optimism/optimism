package actions

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

// TestCrossLayerUser tests that common actions of the CrossLayerUser actor work:
// - transact on L1
// - transact on L2
// - deposit on L1
// - withdraw from L2
// - prove tx on L1
// - wait 1 week + 1 second
// - finalize withdrawal on L1
func TestCrossLayerUser(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)

	miner, seqEngine, seq := setupSequencerTest(t, sd, log)
	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, seq.RollupClient(), miner.EthClient(), seqEngine.EthClient())
	proposer := NewL2Proposer(t, log, &ProposerCfg{
		OutputOracleAddr:  sd.DeploymentsL1.L2OutputOracleProxy,
		ProposerKey:       dp.Secrets.Proposer,
		AllowNonFinalized: true,
	}, miner.EthClient(), seq.RollupClient())

	// need to start derivation before we can make L2 blocks
	seq.ActL2PipelineFull(t)

	l1Cl := miner.EthClient()
	l2Cl := seqEngine.EthClient()
	l2ProofCl := seqEngine.GethClient()

	addresses := e2eutils.CollectAddresses(sd, dp)

	l1UserEnv := &BasicUserEnv[*L1Bindings]{
		EthCl:          l1Cl,
		Signer:         types.LatestSigner(sd.L1Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       NewL1Bindings(t, l1Cl, &sd.DeploymentsL1),
	}
	l2UserEnv := &BasicUserEnv[*L2Bindings]{
		EthCl:          l2Cl,
		Signer:         types.LatestSigner(sd.L2Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       NewL2Bindings(t, l2Cl, l2ProofCl),
	}

	alice := NewCrossLayerUser(log, dp.Secrets.Alice, rand.New(rand.NewSource(1234)))
	alice.L1.SetUserEnv(l1UserEnv)
	alice.L2.SetUserEnv(l2UserEnv)

	// regular L2 tx, in new L2 block
	alice.L2.ActResetTxOpts(t)
	alice.L2.ActSetTxToAddr(&dp.Addresses.Bob)(t)
	alice.L2.ActMakeTx(t)
	seq.ActL2StartBlock(t)
	seqEngine.ActL2IncludeTx(alice.Address())(t)
	seq.ActL2EndBlock(t)
	alice.L2.ActCheckReceiptStatusOfLastTx(true)(t)

	// regular L1 tx, in new L1 block
	alice.L1.ActResetTxOpts(t)
	alice.L1.ActSetTxToAddr(&dp.Addresses.Bob)(t)
	alice.L1.ActMakeTx(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(alice.Address())(t)
	miner.ActL1EndBlock(t)
	alice.L1.ActCheckReceiptStatusOfLastTx(true)(t)

	// regular Deposit, in new L1 block
	alice.ActDeposit(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(alice.Address())(t)
	miner.ActL1EndBlock(t)

	seq.ActL1HeadSignal(t)

	// sync sequencer build enough blocks to adopt latest L1 origin
	for seq.SyncStatus().UnsafeL2.L1Origin.Number < miner.l1Chain.CurrentBlock().NumberU64() {
		seq.ActL2StartBlock(t)
		seq.ActL2EndBlock(t)
	}
	// Now that the L2 chain adopted the latest L1 block, check that we processed the deposit
	alice.ActCheckDepositStatus(true, true)(t)

	// regular withdrawal, in new L2 block
	alice.ActStartWithdrawal(t)
	seq.ActL2StartBlock(t)
	seqEngine.ActL2IncludeTx(alice.Address())(t)
	seq.ActL2EndBlock(t)
	alice.ActCheckStartWithdrawal(true)(t)

	// build a L1 block and more L2 blocks,
	// to ensure the L2 withdrawal is old enough to be able to get into an output root proposal on L1
	miner.ActEmptyBlock(t)
	seq.ActL1HeadSignal(t)
	seq.ActBuildToL1Head(t)

	// submit everything to L1
	batcher.ActSubmitAll(t)
	// include batch on L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	// derive from L1, blocks will now become safe to propose
	seq.ActL2PipelineFull(t)

	// make proposals until there is nothing left to propose
	for proposer.CanPropose(t) {
		// propose it to L1
		proposer.ActMakeProposalTx(t)
		// include proposal on L1
		miner.ActL1StartBlock(12)(t)
		miner.ActL1IncludeTx(dp.Addresses.Proposer)(t)
		miner.ActL1EndBlock(t)
		// Check proposal was successful
		receipt, err := miner.EthClient().TransactionReceipt(t.Ctx(), proposer.LastProposalTx())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "proposal failed")
	}

	// prove our withdrawal on L1
	alice.ActProveWithdrawal(t)
	// include proved withdrawal in new L1 block
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(alice.Address())(t)
	miner.ActL1EndBlock(t)
	// check withdrawal succeeded
	alice.L1.ActCheckReceiptStatusOfLastTx(true)(t)

	// A bit hacky- Mines an empty block with the time delta
	// of the finalization period (12s) + 1 in order for the
	// withdrawal to be finalized successfully.
	miner.ActL1StartBlock(13)(t)
	miner.ActL1EndBlock(t)

	// make the L1 finalize withdrawal tx
	alice.ActCompleteWithdrawal(t)
	// include completed withdrawal in new L1 block
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(alice.Address())(t)
	miner.ActL1EndBlock(t)
	// check withdrawal succeeded
	alice.L1.ActCheckReceiptStatusOfLastTx(true)(t)
}
