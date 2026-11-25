// Package derivation_parity runs op-node and kona-node verifiers side by side on the same L1, feeds
// both the same batches, and compares the safe head each one records for every L1 block. It covers
// the derivation differences that only kona-node has, in its engine, which kona-client does not run.
// Each case records its outcome with parityreport, under the node layer.
package derivation_parity

import (
	"crypto/ecdsa"
	"math/big"
	"slices"
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils/parityreport"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

const (
	opNodeVerifierKey = "verifier-op-node"
	konaVerifierKey   = "verifier-kona"

	// holoceneOffset activates Holocene at the sixth L2 block, leaving blocks 1 to 5 from before
	// it. The system takes longer than this to start, so by the time a test posts its first
	// batch, Holocene is active on L1.
	holoceneOffset = 12

	holoceneAfterGenesisForks = "Holocene activates after L2 genesis"
)

// paritySystem runs one chain with an op-node verifier and a kona-node verifier deriving from the
// same L1. op-node is the reference. The batcher starts stopped, so each test decides what reaches
// L1: honest batches by starting it, or crafted ones with L2Network.PostBatches.
type paritySystem struct {
	*presets.MixedSingleChainFrontends
	Sequencer   *dsl.L2CLNode
	SequencerEL *dsl.L2ELNode
	OpNode      *dsl.L2CLNode
	OpNodeEL    *dsl.L2ELNode
	Kona        *dsl.L2CLNode
	KonaEL      *dsl.L2ELNode
	// Funder is prefunded in the L2 genesis, and sends no L2 transaction unless a test makes it.
	Funder *ecdsa.PrivateKey
}

func newParitySystem(t devtest.T, deployerOpts ...sysgo.DeployerOption) *paritySystem {
	if sysgo.ResolveMixedL2CLKind(t) == sysgo.MixedL2CLKona {
		t.Skip("skipping on kona-node: this suite starts op-node and kona-node itself whatever DEVSTACK_L2CL_KIND says")
	}
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth",
				CLKey:       "sequencer",
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
			},
			{
				ELKey:        opNodeVerifierKey + "-op-reth",
				CLKey:        opNodeVerifierKey,
				CLKind:       sysgo.MixedL2CLOpNode,
				EnableSafeDB: true,
			},
			{
				ELKey:        konaVerifierKey + "-op-reth",
				CLKey:        konaVerifierKey,
				CLKind:       sysgo.MixedL2CLKona,
				EnableSafeDB: true,
			},
		},
		DeployerOptions: deployerOpts,
		BatcherOptions: []sysgo.BatcherOption{func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			cfg.Stopped = true
		}},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	funder, err := sysgo.FunderKey(runtime.L2Network.Keys())
	t.Require().NoError(err, "must derive the L2 funder key")

	sys := &paritySystem{MixedSingleChainFrontends: frontends, Funder: funder}
	for _, node := range frontends.Nodes {
		switch {
		case node.Spec.IsSequencer:
			sys.Sequencer, sys.SequencerEL = node.CL, node.EL
		case node.Spec.CLKey == opNodeVerifierKey:
			sys.OpNode, sys.OpNodeEL = node.CL, node.EL
		case node.Spec.CLKey == konaVerifierKey:
			sys.Kona, sys.KonaEL = node.CL, node.EL
		}
	}
	require := t.Require()
	require.NotNil(sys.SequencerEL, "mixed runtime must expose the sequencer EL")
	require.NotNil(sys.OpNode, "mixed runtime must expose the op-node verifier")
	require.NotNil(sys.Kona, "mixed runtime must expose the kona-node verifier")
	return sys
}

// withHoloceneAfterGenesis activates Holocene the given number of seconds after L2 genesis, and
// leaves every later fork inactive.
func withHoloceneAfterGenesis(offset uint64) sysgo.DeployerOption {
	return func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithForkAtOffset(forks.Holocene, &offset)
		}
	}
}

// recordWhenPassed writes the case's parity record at the end of the test, if the test passed. Call
// it before the system starts, so that the record is written after the system has shut down; the
// test can fill in the record until then.
func recordWhenPassed(gt *testing.T, record *parityreport.Record) {
	gt.Cleanup(func() {
		if gt.Failed() || gt.Skipped() {
			return
		}
		record.Layer = parityreport.LayerNode
		record.Group = gt.Name()
		parityreport.Write(gt, *record)
	})
}

// sequencerBlock returns a copy of the sequencer's block at number, safe to modify.
func sequencerBlock(sys *paritySystem, number uint64) *eth.ExecutionPayload {
	block := *sys.SequencerEL.PayloadByNumber(number).ExecutionPayload
	block.Transactions = slices.Clone(block.Transactions)
	return &block
}

// signedTransfer returns a transfer of 1 wei from key, with fees the block whose base fee is
// baseFee accepts. Whether the block can include it depends only on the sender's nonce and balance.
func signedTransfer(t devtest.T, sys *paritySystem, key *ecdsa.PrivateKey, nonce uint64, baseFee *big.Int) *types.Transaction {
	to := crypto.PubkeyToAddress(key.PublicKey)
	tx, err := types.SignNewTx(key, types.LatestSignerForChainID(sys.L2Network.ChainID().ToBig()), &types.DynamicFeeTx{
		ChainID:   sys.L2Network.ChainID().ToBig(),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), big.NewInt(1)),
		Gas:       21_000,
		To:        &to,
		Value:     big.NewInt(1),
	})
	t.Require().NoError(err)
	return tx
}

// withTransaction returns block with tx appended to its transactions.
func withTransaction(t devtest.T, block *eth.ExecutionPayload, tx *types.Transaction) *eth.ExecutionPayload {
	raw, err := tx.MarshalBinary()
	t.Require().NoError(err)
	block.Transactions = append(block.Transactions, raw)
	return block
}

func baseFee(block *eth.ExecutionPayload) *big.Int {
	return (*uint256.Int)(&block.BaseFeePerGas).ToBig()
}

// verifyDepositsOnly asserts that the block at number on el holds deposits and nothing else.
func verifyDepositsOnly(t devtest.T, el *dsl.L2ELNode, number uint64) {
	txs := el.PayloadByNumber(number).ExecutionPayload.Transactions
	t.Require().NotEmpty(txs, "block %d must hold its L1 info deposit", number)
	for i, tx := range txs {
		t.Require().Truef(optypes.IsDepositTxBytes(tx), "block %d must hold only deposits, but transaction %d is not one", number, i)
	}
}
