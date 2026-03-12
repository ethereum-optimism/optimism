package batcher

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBatcherFullChannelsAfterDowntime(gt *testing.T) {
	gt.Skip("Skipping test until we fix nonce too high error: tx: 177 state: 176")

	t := devtest.ParallelT(gt)
	opts := presets.Combine(
		// Keep verifier EL-sync behavior and no-discovery from the old package-level TestMain.
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.VerifierSyncMode = sync.ELSync
			cfg.NoDiscovery = true
		})),
		// The test advances L1/L2 time aggressively, so enable orchestrator time travel.
		presets.WithTimeTravelEnabled(),
		presets.WithBatcherOption(func(id sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			cfg.Stopped = true

			// Set the blob max size to 40_000 bytes for test purposes.
			cfg.MaxL1TxSize = 40_000
			cfg.TestUseMaxTxSizeForBlobs = true

			cfg.PollInterval = 1000 * time.Millisecond

			cfg.MaxChannelDuration = 50
			cfg.MaxPendingTransactions = 7
		}),
	)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t, opts...)
	sys.AdvanceTime(0) // assert time-travel support is available in this constructor path
	l := t.Logger()
	ts_L2 := sys.TestSequencer.Escape().ControlAPI(sys.L2EL.ChainID())

	alice := sys.FunderL2.NewFundedEOA(eth.OneWei)
	cathrine := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)

	sys.L1CL.Stop()

	latestUnsafe_A := sys.L2CL.StopSequencer()
	l.Info("Latest unsafe block after stopping the L2 sequencer", "latestUnsafe", latestUnsafe_A)

	parent := latestUnsafe_A
	nonce := uint64(0)
	for j := 0; j < 200; j++ {
		l1Origin := sys.L1EL.BlockRefByLabel(eth.Unsafe).Hash

		for i := 0; i < 5; i++ {
			l.Debug("Sequencing L2 block", "iteration", i, "parent", parent)
			sequenceBlockWithL1Origin(t, ts_L2, parent, l1Origin, cathrine, alice, nonce)
			nonce++

			parent = sys.L2CL.HeadBlockRef(types.LocalUnsafe).Hash

			sys.L2EL.WaitForPendingNonceMatch(cathrine.Address(), nonce, 10, 1*time.Second)

			sys.AdvanceTime(time.Second * 2)
		}

		l.Debug("Sequencing L1 block", "iteration_j", j)
		sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})
	}

	sys.L2CL.StartSequencer()

	l.Info("Current L1 unsafe block", "currentL1Unsafe", sys.L1EL.BlockRefByLabel(eth.Unsafe))
	sys.L1CL.Start()

	sys.L2Batcher.Start()

	channels, channelFrames, l2Txs := sys.L2Chain.DeriveData(4) // over the next 4 blocks, collect batches/channels/frames submitted by the batcher on the L1 network, and parse them
	{
		for _, c := range channels {
			l.Info("Channel details", "channelID", c.String(), "frameCount", len(channelFrames[c]), "dataLength_frame0", len(channelFrames[c][0].Data))
		}

		require.Equal(t, 2, len(channels)) // we expect a total of 2 channels

		// values are dependent on:
		// - MaxPendingTransactions
		// - number of blocks and transactions sent in the test - 1000 L2 blocks with 1 transaction from cathrine to alice
		// - MaxL1TxSize (this is set to 40_000 bytes for test purposes)
		sizeRanges := []struct {
			min  int
			max  int
			note string
		}{
			{min: 30_000, max: 40_000, note: "channel 0 - filled to the max capacity"},
			{min: 30_000, max: 40_000, note: "channel 1 - remaining data, filling channel close to max capacity"},
		}

		for i, entry := range sizeRanges {
			require.LessOrEqual(t, len(channelFrames[channels[i]][0].Data), entry.max, entry.note)
			require.GreaterOrEqual(t, len(channelFrames[channels[i]][0].Data), entry.min, entry.note)
		}

		require.Equal(t, len(l2Txs[cathrine.Address()]), 1000) // we expect 1000 transactions total sent from cathrine to alice
	}

	status := sys.L2CL.SyncStatus()
	spew.Dump(status)
}

func sequenceBlockWithL1Origin(t devtest.T, ts apis.TestSequencerControlAPI, parent common.Hash, l1Origin common.Hash, from *dsl.EOA, to *dsl.EOA, nonce uint64) {
	require.NoError(t, ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent, L1Origin: &l1Origin}))

	// include simple transfer tx in opened block
	{
		to := from.PlanTransfer(to.Address(), eth.OneWei)
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
