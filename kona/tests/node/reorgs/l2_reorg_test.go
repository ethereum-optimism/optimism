package reorgs

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	node_utils "github.com/op-rs/kona/node/utils"
	"github.com/stretchr/testify/require"
)

func TestL2Reorg(gt *testing.T) {
	gt.Skip("Skipping l2 reorg test because the L2 test sequencer is flaky")
	const NUM_BLOCKS_TO_REORG = 5
	t := devtest.SerialT(gt)

	out := node_utils.NewMixedOpKonaWithTestSequencer(t)
	sequencerCL := out.L2CLSequencerNodes()[0]
	sequencerEL := out.L2ELSequencerNodes()[0]

	funder := dsl.NewFunder(out.Wallet, out.Faucet, sequencerEL)
	// three EOAs for triggering transfers
	alice := funder.NewFundedEOA(eth.OneHundredthEther)
	bob := funder.NewFundedEOA(eth.OneHundredthEther)

	advancedFnsPreReorg := make([]dsl.CheckFunc, 0, len(out.L2CLNodes()))

	// Wait for the nodes to advance a little bit
	for _, node := range out.L2CLNodes() {
		advancedFnsPreReorg = append(advancedFnsPreReorg, node.AdvancedFn(types.LocalUnsafe, 20, 40))
	}

	dsl.CheckAll(t, advancedFnsPreReorg...)

	unsafeHead := sequencerEL.BlockRefByLabel(eth.Unsafe)

	advancedFnsReorgedBlocks := make([]dsl.CheckFunc, 0, len(out.L2CLNodes()))
	// Wait for the nodes to advance a little bit more ahead the unsafe head
	for _, node := range out.L2CLNodes() {
		advancedFnsReorgedBlocks = append(advancedFnsReorgedBlocks, node.AdvancedFn(types.LocalUnsafe, NUM_BLOCKS_TO_REORG, 2*NUM_BLOCKS_TO_REORG))
	}
	dsl.CheckAll(t, advancedFnsReorgedBlocks...)

	checksPostReorg := []dsl.CheckFunc{}
	// Ensure all the nodes reorg as expected...
	for _, node := range out.L2ELSequencerNodes() {
		reorgedHead := node.BlockRefByLabel(eth.Unsafe)
		require.Greater(t, reorgedHead.Number, unsafeHead.Number)
		checksPostReorg = append(checksPostReorg, node.ReorgTriggeredFn(unsafeHead, 40))
	}

	// Ensure that all the nodes still advance even after the reorg
	for _, node := range out.L2CLNodes() {
		checksPostReorg = append(checksPostReorg, node.AdvancedFn(types.LocalUnsafe, 20, 40))
	}

	reorgFun := func() error {

		// Stop the batcher
		out.L2Batcher.Stop()

		// Stop the main sequencer
		sequencerCL.StopSequencer()

		t.Logger().Info("Rewinding to unsafe head", unsafeHead.Hash)

		parentOfHeadToReorgA := unsafeHead.ParentID()
		parentsL1Origin, err := sequencerEL.Escape().L2EthClient().L2BlockRefByHash(t.Ctx(), parentOfHeadToReorgA.Hash)
		require.NoError(t, err, "Expected to be able to call L2BlockRefByHash API, but got error")

		nextL1Origin := parentsL1Origin.L1Origin.Number + 1
		l1Origin, err := out.L1EL.EthClient().InfoByNumber(t.Ctx(), nextL1Origin)
		require.NoError(t, err, "Expected to get block number %v from L1 execution client", nextL1Origin)
		l1OriginHash := l1Origin.Hash()

		// Reorg the L2 Chain to the unsafe head
		controlAPI := out.TestSequencer.Escape().ControlAPI(out.L2CLNodes()[0].ChainID())
		t.Require().NoError(controlAPI.New(t.Ctx(), seqtypes.BuildOpts{
			Parent:   unsafeHead.ParentHash,
			L1Origin: &l1OriginHash,
		}))
		t.Require().NoError(controlAPI.Open(t.Ctx()))

		// include simple transfer tx in opened block
		{
			t.Logger().Info("Sequencing with op-test-sequencer simple transfer tx")
			to := alice.PlanTransfer(bob.Address(), eth.OneGWei)
			opt := txplan.Combine(to)
			ptx := txplan.NewPlannedTx(opt)
			signed_tx, err := ptx.Signed.Eval(t.Ctx())
			require.NoError(t, err, "Expected to be able to evaluate a planned transaction on op-test-sequencer, but got error")
			txdata, err := signed_tx.MarshalBinary()
			require.NoError(t, err, "Expected to be able to marshal a signed transaction on op-test-sequencer, but got error")

			err = controlAPI.IncludeTx(t.Ctx(), txdata)
			require.NoError(t, err, "Expected to be able to include a signed transaction on op-test-sequencer, but got error")
		}

		controlAPI.Next(t.Ctx())

		// Resume the main sequencer
		sequencerCL.StartSequencer()

		// Resume the batcher
		out.L2Batcher.Start()

		// Ensure all the nodes are connected to the sequencer
		sequencerPeerID := sequencerCL.PeerInfo().PeerID
		for _, node := range out.L2CLValidatorNodes() {
			found := false
			for _, peer := range node.Peers().Peers {
				if peer.PeerID == sequencerPeerID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("expected node %s to be connected to the sequencer", node.Escape().ID().Key())
			}
		}

		return nil
	}

	checksPostReorg = append(checksPostReorg, reorgFun)

	dsl.CheckAll(t, checksPostReorg...)

	// Ensure the current unsafe head is ahead of the reorg head
	for _, node := range out.L2CLNodes() {
		require.Greater(t, node.HeadBlockRef(types.LocalUnsafe).Number, unsafeHead.Number)
	}

	// Ensure that bob has the funds
	for _, node := range out.L2ELSequencerNodes() {
		// Ensure that the recipient's balance has been updated in the eyes of the EL node.
		bob.AsEL(node).VerifyBalanceExact(eth.OneHundredthEther.Add(eth.OneGWei))
		alice.AsEL(node).VerifyBalanceLessThan(eth.OneHundredthEther.Sub(eth.OneGWei))
	}
}
