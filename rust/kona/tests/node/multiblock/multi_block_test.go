package multiblock

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

const (
	// One-second blocks keep the slot short enough that a burst has to be spread over siblings.
	blockTime = 1
	// The largest block group the chain allows.
	maxMultiBlocks = 8
	// Seconds after L2 genesis at which blocks may start sharing a timestamp.
	activationOffset = 8
	// Transactions submitted in the burst. Each block only needs one to be worth sealing, so this
	// is sized to keep the mempool non-empty for the whole observation window.
	burstSize = 200
)

// newMultiBlockPreset brings up a kona-node sequencer and a kona-node verifier, both on op-reth,
// on a chain that allows block groups. The sequencer's execution layer is told to consider a
// payload worth sealing as soon as it holds a user transaction, which is what lets the sequencer
// fit several blocks into one block time.
func newMultiBlockPreset(t devtest.T) *node_utils.MixedOpKonaPreset {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "el-reth-kona-sequencer-0",
				CLKey:       "cl-reth-kona-sequencer-0",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLKona,
				IsSequencer: true,
			},
			{
				ELKey:  "el-reth-kona-validator-0",
				CLKey:  "cl-reth-kona-validator-0",
				ELKind: sysgo.MixedL2ELOpReth,
				CLKind: sysgo.MixedL2CLKona,
			},
		},
		DeployerOptions: []sysgo.DeployerOption{
			sysgo.WithUniformL2BlockTimes(blockTime),
			sysgo.WithMultiBlockAtOffset(activationOffset),
			sysgo.WithMaxMultiBlocks(maxMultiBlocks),
		},
		OpRethOptions: []sysgo.OpRethOption{
			sysgo.OpRethWithExtraArgs(
				"--rollup.multi-block.min-txs", "1",
				"--rollup.multi-block.min-build-time", "100",
			),
			sysgo.OpRethWithBuilderInterval(50 * time.Millisecond),
		},
	})
	return node_utils.NewMixedOpKonaFromRuntime(t, runtime)
}

// submitBurst fills the mempool with transfers, without waiting for any of them to be included.
// Nonces are explicit so the submissions do not race each other through a pending-nonce lookup.
func submitBurst(t devtest.T, user *dsl.EOA, to *dsl.EOA, count uint64) {
	nonce := user.PendingNonce()
	for i := uint64(0); i < count; i++ {
		tx := txplan.NewPlannedTx(
			user.PlanTransfer(to.Address(), eth.GWei(1)),
			txplan.WithNonce(nonce+i),
		)
		_, err := tx.Submitted.Eval(t.Ctx())
		t.Require().NoError(err, fmt.Sprintf("failed to submit burst tx %d", i))
	}
}

// TestSequencerBuildsSiblingBlocksUnderLoad asserts that a kona-node sequencer on op-reth puts
// more than one block on a single timestamp once the chain allows it, and that the resulting
// chain obeys the multi-blocks rules.
func TestSequencerBuildsSiblingBlocksUnderLoad(gt *testing.T) {
	t := devtest.SerialT(gt)
	out := newMultiBlockPreset(t)

	out.L2Chain.AwaitMultiBlockActivation(t)

	sequencerEL := out.L2ELKonaSequencerNodes[0]
	funder := out.Funder.AsFunder(sequencerEL)
	user := funder.NewFundedEOA(eth.OneEther)
	recipient := out.Wallet.NewEOA(sequencerEL)

	submitBurst(t, user, recipient, burstSize)

	first, last := sequencerEL.WaitForSiblingBlocks(2, 60*time.Second)
	t.Require().Equal(first.Time, last.Time, "the run shares one timestamp")

	sequencerEL.VerifyTimestampGroups(first.Number, last.Number, out.L2Chain.Escape().RollupConfig())
}
