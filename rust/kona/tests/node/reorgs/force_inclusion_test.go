package reorgs

import (
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestSequencerRecoversAfterForcedInclusion exercises derivation replacing a live
// sequencer's unsafe chain when unpublished batches expire. The isolated verifier
// must independently derive the same forced deposit and the recovered chain.
func TestSequencerRecoversAfterForcedInclusion(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()
	ctx := t.Ctx()
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey: "sequencer-el", CLKey: "sequencer-cl",
				ELKind: sysgo.MixedL2ELOpReth, CLKind: sysgo.MixedL2CLKona,
				IsSequencer: true,
			},
			{
				ELKey: "verifier-el", CLKey: "verifier-cl",
				ELKind: sysgo.MixedL2ELOpReth, CLKind: sysgo.MixedL2CLKona,
				IsolateFromL2P2P: true,
			},
		},
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithSequencingWindow(10)},
		BatcherOptions: []sysgo.BatcherOption{func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			// Recovery mode requires singular batches.
			cfg.BatchType = derive.SingularBatchType
		}},
	})
	sys := presets.NewMixedSingleChainFrontends(t, runtime)
	sequencer := sys.Nodes[0]
	verifier := sys.Nodes[1]
	alice := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	depositor := sys.FunderL1.NewFundedEOA(eth.OneEther)

	initial := alice.Transfer(common.HexToAddress("0x7777"), eth.GWei(100))
	initialReceipt, err := initial.Included.Eval(ctx)
	require.NoError(err)
	initialNumber := bigs.Uint64Strict(initialReceipt.BlockNumber)
	// The isolated verifier must bootstrap and derive from L1 without any
	// unsafe payload injection or L2 peers, including before the batching outage.
	dsl.CheckAll(t,
		sequencer.EL.ReachedFn(eth.Safe, initialNumber, 60),
		verifier.EL.ReachedFn(eth.Safe, initialNumber, 60),
	)
	for _, node := range sys.Nodes {
		require.Equal(initialReceipt.BlockHash, node.EL.BlockRefByNumber(initialNumber).Hash,
			"initial transaction must be safe on %s", node.EL)
	}

	// Stop drains pending submissions, so L1 must keep running during this call.
	sys.L2Batcher.Stop()
	stoppedAt := sys.L1Network.WaitForBlock()
	sequencer.CL.AwaitMinL1Processed(stoppedAt.Number)

	marker := alice.Transfer(common.HexToAddress("0xdead"), eth.GWei(42))
	markerReceipt, err := marker.Included.Eval(ctx)
	require.NoError(err)
	markerBlock := eth.L2BlockRef{Hash: markerReceipt.BlockHash, Number: bigs.Uint64Strict(markerReceipt.BlockNumber)}
	require.Less(sequencer.EL.BlockRefByLabel(eth.Safe).Number, markerBlock.Number,
		"marker must be in unpublished unsafe history")
	// Keep sequencing during the forced reorg, but prevent mempool reinclusion of
	// the marker from making the replacement chain incompatible again.
	require.NoError(sequencer.CL.Escape().RollupAPI().SetRecoverMode(ctx, true))

	amount := eth.OneHundredthEther
	portal := sys.L2Network.DepositContractAddr()
	deposit := depositor.Transfer(portal, amount)
	l1Receipt, err := deposit.Included.Eval(ctx)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, l1Receipt.Status)
	depositReceipt := sequencer.EL.WaitForDeposit(portal, l1Receipt)
	require.Equal(types.ReceiptStatusSuccessful, depositReceipt.Status)

	sequencer.EL.ReorgExact(markerBlock, 120)
	depositOrigin := bigs.Uint64Strict(l1Receipt.BlockNumber)
	dsl.CheckAll(t,
		sequencer.EL.L1OriginReachedFn(eth.Safe, depositOrigin, 180),
		verifier.EL.L1OriginReachedFn(eth.Safe, depositOrigin, 180),
	)
	var forcedDeposit eth.BlockID
	for _, node := range sys.Nodes {
		// A receipt index can briefly retain the pre-reorg block. Require the
		// deposit receipt and the canonical chain to agree in the same retry.
		var included eth.BlockID
		require.Eventuallyf(func() bool {
			receipt, err := node.EL.Escape().EthClient().TransactionReceipt(ctx, depositReceipt.TxHash)
			if err != nil {
				return false
			}
			block, err := node.EL.Escape().L2EthClient().L2BlockRefByNumber(ctx, bigs.Uint64Strict(receipt.BlockNumber))
			if err != nil || block.Hash != receipt.BlockHash || receipt.Status != types.ReceiptStatusSuccessful {
				return false
			}
			included = block.ID()
			return true
		}, 30*time.Second, time.Second, "forced deposit must be canonical on %s", node.EL)
		if forcedDeposit == (eth.BlockID{}) {
			forcedDeposit = included
		}
		require.Equal(forcedDeposit, included, "nodes must derive the same deposit block")
		depositor.AsEL(node.EL).VerifyBalanceExact(amount)
		require.NotEqual(markerBlock.Hash, node.EL.BlockRefByNumber(markerBlock.Number).Hash,
			"unpublished marker block must be orphaned on %s", node.EL)
	}
	require.Eventually(func() bool {
		nonce, err := sequencer.EL.Escape().EthClient().NonceAt(ctx, alice.Address(), nil)
		return err == nil && nonce == marker.Nonce.Value()
	}, 30*time.Second, time.Second, "orphaned marker must not be reexecuted in recovery mode")

	sys.L2Batcher.Start()
	// Catch up with force-derived history before enabling normal transactions.
	dsl.CheckAll(t,
		sequencer.EL.AdvancedFn(eth.Unsafe, 5),
		sequencer.EL.AdvancedFn(eth.Safe, 5),
		verifier.EL.AdvancedFn(eth.Safe, 5),
	)
	require.NoError(sequencer.CL.Escape().RollupAPI().SetRecoverMode(ctx, false))
	recovered := alice.Transfer(common.HexToAddress("0x7777"), eth.GWei(100))
	recoveredReceipt, err := recovered.Included.Eval(ctx)
	require.NoError(err)
	recoveredNumber := bigs.Uint64Strict(recoveredReceipt.BlockNumber)
	var convergence []dsl.CheckFunc
	for _, node := range sys.Nodes {
		convergence = append(convergence,
			node.CL.ReachedFn(safety.CrossSafe, recoveredNumber, 90),
			node.EL.ReachedFn(eth.Safe, recoveredNumber, 90),
			node.CL.ReachedFn(safety.Finalized, recoveredNumber, 360),
			node.EL.ReachedFn(eth.Finalized, recoveredNumber, 360),
		)
	}
	dsl.CheckAll(t, convergence...)
	for _, node := range sys.Nodes {
		require.Equal(recoveredReceipt.BlockHash, node.EL.BlockRefByNumber(recoveredNumber).Hash,
			"recovered ordinary transaction must be canonical and finalized on %s", node.EL)
		status := node.CL.SyncStatus()
		for _, head := range []eth.L2BlockRef{status.SafeL2, status.FinalizedL2} {
			require.Equal(head.Hash, node.EL.BlockRefByNumber(head.Number).Hash,
				"CL safety heads must match the EL canonical chain on %s", node.EL)
		}
	}
}
