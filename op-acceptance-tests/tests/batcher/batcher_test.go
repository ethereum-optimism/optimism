package batcher

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBatcherFullChannelsAfterDowntime(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	l := t.Logger()
	ts_L2 := sys.TestSequencer.Escape().ControlAPI(sys.L2EL.ChainID())
	ts_L1 := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())

	alice := sys.FunderL2.NewFundedEOA(eth.OneWei)
	cathrine := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)

	cl := sys.L1Network.Escape().L1CLNode(match.FirstL1CL)

	sys.ControlPlane.FakePoSState(cl.ID(), stack.Stop)

	latestUnsafe_A := sys.L2CL.StopSequencer()
	l.Info("Latest unsafe block after stopping the L2 sequencer", "latestUnsafe", latestUnsafe_A)

	parent := latestUnsafe_A
	nonce := uint64(0)
	for j := 0; j < 200; j++ {
		l1Origin := sys.L1EL.BlockRefByLabel(eth.Unsafe).Hash

		for i := 0; i < 5; i++ {
			l.Debug("Sequencing L2 block", "iteration", i, "parent", parent)
			sequenceBlockWithL1Origin(t, ts_L2, parent, l1Origin, alice, cathrine, nonce)
			nonce++

			parent = sys.L2CL.HeadBlockRef(types.LocalUnsafe).Hash

			sys.AdvanceTime(time.Second * 2)
		}

		l.Debug("Sequencing L1 block", "iteration_j", j)
		sequenceBlock(t, ts_L1, common.Hash{})
	}

	sys.L2CL.StartSequencer()

	l.Info("Current L1 unsafe block", "currentL1Unsafe", sys.L1EL.BlockRefByLabel(eth.Unsafe))
	sys.ControlPlane.FakePoSState(cl.ID(), stack.Start)

	sys.L2Batcher.Start()

	channels, channelFrames, l2Txs := sys.L2Chain.DeriveData(4) // over the next 4 blocks, collect batches/channels/frames submitted by the batcher on the L1 network, and parse them
	{
		for _, c := range channels {
			l.Info("Channel details", "channelID", c.String(), "frameCount", len(channelFrames[c]), "dataLength_frame0", len(channelFrames[c][0].Data))
		}

		require.Equal(t, 10, len(channels)) // we expect a total of 10 channels, due to existing MaxChannelDuration bug filed at: https://github.com/ethereum-optimism/optimism/issues/18092

		// values are dependent on:
		// - MaxPendingTransactions
		// - number of blocks and transactions sent in the test - 1000 L2 blocks with 1 transaction from cathrine to alice
		// - MaxL1TxSize (this is set to 40_000 bytes for test purposes)
		sizeRanges := []struct {
			min  int
			max  int
			note string
		}{
			{min: 7_000, max: 10_000, note: "channel 0 - the first 100 blocks"},
			{min: 100, max: 1000, note: "channel 1 - only a few blocks due to racy behavior, nowhere near the channel capacity"},
			{min: 100, max: 1000, note: "channel 2 - only a few blocks due to racy behavior, nowhere near the channel capacity"},
			{min: 100, max: 1000, note: "channel 3 - only a few blocks due to racy behavior, nowhere near the channel capacity"},
			{min: 100, max: 1000, note: "channel 4 - only a few blocks due to racy behavior, nowhere near the channel capacity"},
			{min: 100, max: 1000, note: "channel 5 - only a few blocks due to racy behavior, nowhere near the channel capacity"},
			{min: 100, max: 1000, note: "channel 6 - only a few blocks due to racy behavior, nowhere near the channel capacity"},
			{min: 100, max: 1000, note: "channel 7 - only a few blocks due to racy behavior, nowhere near the channel capacity"},
			{min: 20_000, max: 40_000, note: "channel 8 - filled to the max capacity, due to blocking behavior because of MaxPendingTransactions limit reached"},
			{min: 20_000, max: 40_000, note: "channel 9 - filled to the max capacity, due to blocking behavior because of MaxPendingTransactions limit reached"},
		}

		for i, entry := range sizeRanges {
			require.LessOrEqual(t, len(channelFrames[channels[i]][0].Data), entry.max, entry.note)
			require.GreaterOrEqual(t, len(channelFrames[channels[i]][0].Data), entry.min, entry.note)
		}

		require.Equal(t, len(l2Txs[cathrine.Address()]), 1000) // we expect 1000 transactions total sent from cathrine to alice
	}

	//sys.L2Chain.PrintChain()
	//sys.L1Network.PrintChain()

	status := sys.L2CL.SyncStatus()
	spew.Dump(status)
}

func sequenceBlock(t devtest.T, ts apis.TestSequencerControlAPI, parent common.Hash) {
	require.NoError(t, ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent}))
	require.NoError(t, ts.Next(t.Ctx()))
}

func sequenceBlockWithL1Origin(t devtest.T, ts apis.TestSequencerControlAPI, parent common.Hash, l1Origin common.Hash, alice *dsl.EOA, cathrine *dsl.EOA, nonce uint64) {
	require.NoError(t, ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent, L1Origin: &l1Origin}))

	// include simple transfer tx in opened block
	{
		to := cathrine.PlanTransfer(alice.Address(), eth.OneWei)
		opt := txplan.Combine(to, txplan.WithStaticNonce(nonce))
		ptx := txplan.NewPlannedTx(opt)
		signed_tx, err := ptx.Signed.Eval(t.Ctx())
		require.NoError(t, err, "Expected to be able to evaluate a planned transaction on op-test-sequencer, but got error")
		txdata, err := signed_tx.MarshalBinary()
		require.NoError(t, err, "Expected to be able to marshal a signed transaction on op-test-sequencer, but got error")

		err = ts.IncludeTx(t.Ctx(), txdata)
		require.NoError(t, err, "Expected to be able to include a signed transaction on op-test-sequencer, but got error")
	}

	require.NoError(t, ts.Next(t.Ctx()))
}
