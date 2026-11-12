package op_reth_txpool_minimum_priority_fee

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const (
	// conditionalWindowBlocks is how many blocks past the head a conditional transaction may be
	// included in: long enough that the sequencer still meets the condition when it validates it.
	conditionalWindowBlocks = 1_000
	// feeCapGwei is pinned on every transaction the tests sign offline. It leaves room for the
	// base fee, so only the tip decides what the pool does with a transaction.
	feeCapGwei = 100
	// l2BlockTimeSeconds keeps the runs short.
	l2BlockTimeSeconds = 1
	// minimumPriorityFeeWei is the sequencer's --txpool.minimum-priority-fee: 1 gwei, so a tip
	// just below it is an ordinary low tip rather than no tip at all.
	minimumPriorityFeeWei uint64 = params.GWei
)

var (
	// atMinimum is the lowest tip the pool admits; belowMinimum is the highest tip it refuses.
	atMinimum    = new(big.Int).SetUint64(minimumPriorityFeeWei)
	belowMinimum = new(big.Int).SetUint64(minimumPriorityFeeWei - 1)
	feeCap       = big.NewInt(feeCapGwei * params.GWei)
	// fundingTip is the tip on every funding transfer. It clears the minimum with room to spare,
	// whatever txplan's default tip, so funding a sender never rides on the boundary the tests probe.
	fundingTip = new(big.Int).SetUint64(2 * minimumPriorityFeeWei)
)

// sequencerSystem is an op-reth sequencer alone, with --txpool.minimum-priority-fee set, reth's
// local exemptions kept and conditional transactions enabled, so what it does with a transaction
// submitted over its RPC is what a production sequencer running with the flag does.
type sequencerSystem struct {
	Funder    *dsl.FunderEOA
	Sequencer *dsl.L2ELNode
	Wallet    *dsl.HDWallet
}

func newSequencerSystem(t devtest.T) *sequencerSystem {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds)},
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{{
			CLKey:       "sequencer",
			CLKind:      sysgo.ResolveMixedL2CLKind(),
			ELKey:       "sequencer",
			ELKind:      sysgo.MixedL2ELOpReth,
			IsSequencer: true,
			OpRethOpts: []sysgo.OpRethOption{
				sysgo.OpRethWithTxConditional(),
				sysgo.OpRethWithTxPoolLocalExemptions(),
				sysgo.OpRethWithTxPoolMinimumPriorityFee(minimumPriorityFeeWei),
			},
		}},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 1, "expected the sequencer alone")
	return &sequencerSystem{
		Funder:    frontends.FunderL2.WithFundingOptions(txplan.WithGasTipCap(fundingTip)),
		Sequencer: frontends.Nodes[0].EL,
		Wallet:    frontends.Wallet,
	}
}

// signTransfer returns a transfer from sender to a fresh account, signed offline with the given tip
// and not yet submitted anywhere.
func (sys *sequencerSystem) signTransfer(sender *dsl.EOA, tip *big.Int) *types.Transaction {
	recipient := sys.Wallet.NewEOA(sys.Sequencer).Address()
	return sender.Sign(
		sender.PlanOffline(),
		txplan.WithTo(&recipient),
		txplan.WithValue(eth.OneHundredthEther),
		txplan.WithGasFeeCap(feeCap),
		txplan.WithGasTipCap(tip),
	)
}

// inclusionWindow is a condition the sequencer meets for a while: inclusion within the next
// conditionalWindowBlocks blocks.
func (sys *sequencerSystem) inclusionWindow() dsl.TransactionConditional {
	lastBlock := hexutil.Uint64(sys.Sequencer.BlockRefByLabel(eth.Unsafe).Number + conditionalWindowBlocks)
	return dsl.TransactionConditional{BlockNumberMax: &lastBlock}
}

// submissionCase is one way of handing a transaction to the sequencer, with the transfer to hand
// it.
type submissionCase struct {
	submit   func(sys *sequencerSystem, tx *types.Transaction) *dsl.TransactionSubmission
	transfer *types.Transaction
}

// submissionMatrix returns every way of handing a transaction to the sequencer, each with its own
// transfer at the given tip from its own sender. The senders are funded together before any case
// runs, so the cases share nothing but the sequencer.
func (sys *sequencerSystem) submissionMatrix(tip *big.Int) []submissionCase {
	cases := []submissionCase{
		{
			submit: func(sys *sequencerSystem, tx *types.Transaction) *dsl.TransactionSubmission {
				return sys.Sequencer.SendRawTransaction(tx)
			},
		},
		{
			submit: func(sys *sequencerSystem, tx *types.Transaction) *dsl.TransactionSubmission {
				return sys.Sequencer.SendRawTransactionConditional(tx, sys.inclusionWindow())
			},
		},
	}
	for i, sender := range sys.Funder.NewFundedEOAs(len(cases), eth.OneEther) {
		cases[i].transfer = sys.signTransfer(sender, tip)
	}
	return cases
}
