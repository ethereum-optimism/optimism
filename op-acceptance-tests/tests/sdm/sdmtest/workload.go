package sdmtest

import (
	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// IncludedTx pairs a confirmed transaction receipt with the L2 block it landed in.
type IncludedTx struct {
	Receipt  *types.Receipt
	BlockNum uint64
}

// AddrPtr returns a pointer to addr, for the many txplan options that take *common.Address.
func AddrPtr(addr common.Address) *common.Address {
	return &addr
}

// SubmitTxWithoutWait sends a transaction to the mempool without waiting for inclusion.
// Returns the PlannedTx whose Included field can be evaluated later.
// The caller must provide a nonce to avoid the default PendingNonce lookup racing between txs.
func SubmitTxWithoutWait(
	t devtest.T,
	alice *dsl.EOA,
	nonce uint64,
	opts ...txplan.Option,
) *txplan.PlannedTx {
	combined := append([]txplan.Option{
		alice.Plan(),
		txplan.WithNonce(nonce),
	}, opts...)
	ptx := txplan.NewPlannedTx(combined...)
	_, err := ptx.Submitted.Eval(t.Ctx())
	t.Require().NoError(err, "failed to submit tx with nonce %d", nonce)
	return ptx
}

// DeployContract deploys the given hex bytecode from eoa and returns the deployed address.
func DeployContract(t devtest.T, eoa *dsl.EOA, hexBytecode string) common.Address {
	tx := txplan.NewPlannedTx(eoa.Plan(), txplan.WithData(common.FromHex(hexBytecode)))
	res, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "failed to deploy contract")
	return res.ContractAddress
}

// MustFindRepeatedSlotBlock drives a repeated-slot warming workload (a burst of txs hitting the
// same storage slots) until one block lands at least minUserTxs of them together, which is what
// produces cross-tx warming refunds and therefore a PostExec (0x7D) tx. It returns that block, the
// txs that landed in it, and its number. Fails the test after maxAttempts dense-block misses.
func MustFindRepeatedSlotBlock(
	t devtest.T,
	sys *RethSystem,
	minUserTxs int,
	maxAttempts int,
) (*sdmpkg.RPCBlock, []IncludedTx, uint64) {
	l := t.Logger()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
		stateBloatAddr := DeployContract(t, alice, sdmpkg.StateBloatBin)

		const batchSize = 50
		const slotCount = 20
		startNonce := alice.PendingNonce()
		plannedTxs := make([]*txplan.PlannedTx, 0, batchSize)

		l.Info("Submitting repeated-slot workload",
			"attempt", attempt,
			"alice", alice.Address(),
			"contract", stateBloatAddr,
			"startNonce", startNonce,
			"batchSize", batchSize,
			"slotCount", slotCount)

		for i := 0; i < batchSize; i++ {
			nonce := startNonce + uint64(i)
			plannedTxs = append(plannedTxs, SubmitTxWithoutWait(
				t,
				alice,
				nonce,
				txplan.WithTo(AddrPtr(stateBloatAddr)),
				txplan.WithData(sdmpkg.EncodeRun(slotCount)),
				txplan.WithGasLimit(1_000_000),
			))
		}

		blockTxs := make(map[uint64][]IncludedTx)
		for i, ptx := range plannedTxs {
			receipt, err := ptx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "attempt %d tx %d: failed to get receipt", attempt, i)
			t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status,
				"attempt %d tx %d: must succeed", attempt, i)

			itx := IncludedTx{Receipt: receipt, BlockNum: bigs.Uint64Strict(receipt.BlockNumber)}
			blockTxs[itx.BlockNum] = append(blockTxs[itx.BlockNum], itx)
		}

		var targetBlockNum uint64
		var targetIncluded []IncludedTx
		for blockNum, txs := range blockTxs {
			if len(txs) > len(targetIncluded) {
				targetBlockNum = blockNum
				targetIncluded = txs
			}
		}
		if len(targetIncluded) < minUserTxs {
			l.Warn("Repeated-slot workload did not produce a dense-enough block",
				"attempt", attempt,
				"requiredUserTxs", minUserTxs,
				"bestUserTxs", len(targetIncluded),
				"bestBlock", targetBlockNum)
			continue
		}

		block := GetBlockWithTxs(t, sys.L2EL, targetBlockNum)
		t.Require().Greater(len(block.Transactions), 0, "block must have at least one transaction")
		t.Require().Equal(uint64(types.DepositTxType), uint64(block.Transactions[0].Type),
			"position 0 must be a deposit tx (L1 info)")
		return block, targetIncluded, targetBlockNum
	}

	t.Require().FailNowf("repeated-slot workload failed",
		"no block with at least %d user txs found after %d attempts", minUserTxs, maxAttempts)
	return nil, nil, 0
}
