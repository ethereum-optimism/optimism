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
)

// sequencerSystem is an op-reth sequencer alone, with --txpool.minimum-priority-fee set, reth's
// local exemptions kept and conditional transactions enabled, so what it does with a transaction
// submitted over its RPC is what a production sequencer running with the flag does.
type sequencerSystem struct {
	FunderL2  *dsl.FunderEOA
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
		FunderL2:  frontends.FunderL2,
		Sequencer: frontends.Nodes[0].EL,
		Wallet:    frontends.Wallet,
	}
}

// fundAndSignTransfer funds a fresh account, waiting for the funding to land, and returns a
// transfer from it to another fresh account, signed offline with the given tip and not yet
// submitted anywhere.
func (sys *sequencerSystem) fundAndSignTransfer(tip *big.Int) *types.Transaction {
	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bob := sys.Wallet.NewEOA(sys.Sequencer).Address()
	return alice.Sign(
		alice.PlanOffline(),
		txplan.WithStaticNonce(0),
		txplan.WithTo(&bob),
		txplan.WithValue(eth.OneHundredthEther),
		txplan.WithGasLimit(params.TxGas),
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

// submissionCase is one way of handing a transaction to the sequencer.
type submissionCase struct {
	// localOrigin is whether the sequencer pools the transaction as a local one. reth 2.5.2
	// exempts local transactions from the minimum priority fee; a conditional transaction enters
	// the pool as a private one, which has never been exempt.
	localOrigin bool
	name        string
	submit      func(sys *sequencerSystem, tx *types.Transaction) *dsl.TransactionSubmission
}

func submissionMatrix() []submissionCase {
	return []submissionCase{
		{
			localOrigin: true,
			name:        "eth_sendRawTransaction",
			submit: func(sys *sequencerSystem, tx *types.Transaction) *dsl.TransactionSubmission {
				return sys.Sequencer.SendRawTransaction(tx)
			},
		},
		{
			name: "eth_sendRawTransactionConditional",
			submit: func(sys *sequencerSystem, tx *types.Transaction) *dsl.TransactionSubmission {
				return sys.Sequencer.SendRawTransactionConditional(tx, sys.inclusionWindow())
			},
		},
	}
}

func skipUnsupported(t devtest.T) {
	sysgo.SkipOnOpGeth(t, "the sequencer under test is this tree's op-reth")
}
