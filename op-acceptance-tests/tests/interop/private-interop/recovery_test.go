package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
)

func TestPrivateChainRecoversFromInvalidExecutingMessage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck(), sysgo.WithPrivateInteropCadence(12)))
	require := t.Require()
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	source := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2BatcherB.Stop()
	parent := sys.L2BCL.StopSequencer()

	// The test sequencer bypasses mempool checks. Include a valid private EVM
	// call whose claimed initiating event does not exist on the public chain.
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](alice.Plan())
	tx.Content.Set(&txintent.ExecTrigger{Executor: predeploys.CrossL2InboxAddr, Msg: messages.Message{
		Identifier: messages.Identifier{Origin: alice.Address(), BlockNumber: source.Number, LogIndex: 1024,
			Timestamp: source.Time, ChainID: sys.L2A.ChainID()}, PayloadHash: common.Hash{1},
	}})
	signed, err := tx.PlannedTx.Signed.Eval(t.Ctx())
	require.NoError(err)
	raw, err := signed.MarshalBinary()
	require.NoError(err)
	// Keep an ordinary prefix before the invalid execution.
	sys.TestSequencer.SequenceBlock(t, sys.L2B.ChainID(), parent)
	prefix := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	sys.TestSequencer.SequenceBlockWithTxs(t, sys.L2B.ChainID(), prefix.Hash, [][]byte{raw})
	invalid := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	receipt := sys.L2ELB.WaitForReceipt(signed.Hash())
	require.Equal(uint64(1), receipt.Status)
	require.NotEmpty(receipt.Logs)
	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.UnsafeL2.Hash == invalid.Hash
	},
		30*time.Second, 100*time.Millisecond, "private LightCL must observe the test-sequenced block")
	sys.L2BCL.StartSequencer()
	sys.L2BatcherB.Start()

	// Cross-safety replaces the projection block. The private LightCL must
	// independently rebuild the corresponding block against its private parent.
	require.Eventually(func() bool {
		ref, err := sys.L2ELB.Escape().L2EthClient().L2BlockRefByNumber(t.Ctx(), invalid.Number)
		return err == nil && ref.Hash != invalid.Hash && ref.ParentHash == invalid.ParentHash
	}, 3*time.Minute, time.Second, "invalid private execution must be replaced")
	// The rejected span's remaining positions may become canonical only when
	// the sequencing window expires. Wait for publication beyond that range.
	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.LocalSafeL2.Number >= invalid.Number+12 &&
			status.UnsafeL2.Number >= status.LocalSafeL2.Number && status.UnsafeL2.Number-status.LocalSafeL2.Number <= 8
	}, 3*time.Minute, time.Second, "private execution and publication must catch up after replacement")
	transfer := alice.Transfer(common.Address{0xee}, eth.OneGWei)
	included, err := transfer.Included.Eval(t.Ctx())
	require.NoError(err)
	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.LocalSafeL2.Number >= bigs.Uint64Strict(included.BlockNumber)
	}, 3*time.Minute, time.Second, "private publication must resume after replacement")
}
