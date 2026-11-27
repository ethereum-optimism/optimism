package sequencer

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

// TestTestSequencerReorgsUnsafeHead drives the sequencer's rollup node through its opstack
// block-building API with op-test-sequencer. It replaces the unsafe head with a block that holds
// a transfer and extends it, and checks that the rollup node and its own sequencer continue on
// them, and that derivation keeps them on every node once they are batched.
func TestTestSequencerReorgsUnsafeHead(gt *testing.T) {
	t := devtest.SerialT(gt)
	kind := sysgo.ResolveMixedL2CLKind(t)
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{ELKey: "sequencer-el", CLKey: "sequencer-cl", CLKind: kind, IsSequencer: true},
			{ELKey: "verifier-el", CLKey: "verifier-cl", CLKind: kind},
		},
		WithTestSequencer: true,
	})
	sys := presets.NewMixedSingleChainFrontends(t, runtime)
	sequencer, verifier := sys.Nodes[0], sys.Nodes[1]
	chainID := sys.L2Network.ChainID()

	alice := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.Wallet.NewEOA(sequencer.EL)

	// Keep the blocks the test sequencer replaces unsafe.
	sys.L2Batcher.Stop()
	sequencer.CL.AdvancedUnsafe(2, 30)
	sequencer.CL.StopSequencer()
	original := sequencer.EL.BlockRefByLabel(eth.Unsafe)

	transfer, err := txplan.NewPlannedTx(alice.PlanTransfer(bob.Address(), eth.OneGWei)).Signed.Eval(t.Ctx())
	t.Require().NoError(err)
	rawTransfer, err := transfer.MarshalBinary()
	t.Require().NoError(err)
	ts := sys.TestSequencer.Escape().ControlAPI(chainID)
	t.Require().NoError(ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: original.ParentHash}))
	t.Require().NoError(ts.IncludeTx(t.Ctx(), rawTransfer))
	t.Require().NoError(ts.Next(t.Ctx()))
	sequencer.EL.ReorgExact(original, 30)
	replacement := sequencer.EL.BlockRefByNumber(original.Number)
	sys.TestSequencer.SequenceBlock(t, chainID, replacement.Hash)
	sequencer.EL.Reached(eth.Unsafe, replacement.Number+1, 30)
	tip := sequencer.EL.BlockRefByNumber(replacement.Number + 1)
	t.Require().Equal(replacement.Hash, tip.ParentHash, "the second block extends the replacement")
	bob.VerifyBalanceExact(eth.OneGWei)

	// The rollup node tracks the committed head, and its sequencer resumes on it.
	sequencer.CL.MatchedUnsafe(sequencer.EL, 30)
	sys.L2Batcher.Start()
	sequencer.CL.StartSequencer()
	sequencer.CL.AdvancedUnsafe(2, 30)
	t.Require().Equal(tip.Hash, sequencer.EL.BlockRefByNumber(tip.Number).Hash,
		"the node's sequencer must build on the committed blocks")
	verifier.CL.ReachedRef(safety.CrossSafe, tip.ID(), 60)
	sequencer.CL.ReachedRef(safety.CrossSafe, replacement.ID(), 60)
}
