package sdm

import (
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sdm/sdmtest"
	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// SDM-H1: a batched block whose 0x7D fails verification must be replaced by a deposits-only
// block, as the fault-proof program does. op-reth used to fail such a build only at getPayload,
// which stalled the safe head.
func TestSDMInvalidPostExecIsReplacedWithDepositsOnly(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newFixedPolicySDMRethSystem(t)
	sdmtest.VerifyOpReth(t, sys.L2EL)
	sdmtest.VerifyOpReth(t, sys.L2ELVerifier)

	block, receipt := submitFixtureProbe(t, sys)
	target := uint64(block.Number)
	honest := sys.L2EL.BlockRefByNumber(target)
	postExecTx, _ := sdmpkg.FindPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "the probe block must carry a 0x7D")
	// The verifier holds the honest block as unsafe, so derivation must reorg it out.
	sys.L2ELVerifier.Reached(eth.Unsafe, target, 60)

	from := min(sys.L2CL.SafeL2BlockRef().Number, sys.L2CLVerifier.SafeL2BlockRef().Number) + 1
	sdmtest.PostTamperedSingularBatches(t, sys, from, target, func(txs []eth.Data) []eth.Data {
		return overRefundPostExec(t, receipt, txs)
	})

	dsl.CheckAll(t,
		sys.L2CL.ReachedFn(safety.CrossSafe, target, 120),
		sys.L2CLVerifier.ReachedFn(safety.CrossSafe, target, 120),
		sys.L2EL.ReachedFn(eth.Safe, target, 120),
		sys.L2ELVerifier.ReachedFn(eth.Safe, target, 120),
	)
	replacement := sdmtest.GetBlockWithTxs(t, sys.L2ELVerifier, target)
	t.Require().NotEqual(block.Hash, replacement.Hash, "the invalid block must be replaced")
	t.Require().Equal(honest.ParentHash, sys.L2ELVerifier.BlockRefByNumber(target).ParentHash,
		"only the tampered block may be replaced")
	for i, tx := range replacement.Transactions {
		t.Require().Equal(uint64(gethtypes.DepositTxType), uint64(tx.Type),
			"replacement tx %d must be a deposit", i)
	}
	t.Require().Equal(replacement.Hash, sdmtest.GetBlockWithTxs(t, sys.L2EL, target).Hash,
		"the sequencer must derive the same replacement")

	sys.L2Batcher.Start()
	dsl.CheckAll(t,
		sys.L2CL.ReachedFn(safety.CrossSafe, target+3, 120),
		sys.L2CLVerifier.ReachedFn(safety.CrossSafe, target+3, 120),
	)
}

// overRefundPostExec raises the probe tx's refund one gas above the gas it used before the refund.
func overRefundPostExec(t devtest.T, probe *gethtypes.Receipt, txs []eth.Data) []eth.Data {
	last := len(txs) - 1
	postExecTx, err := optypes.UnmarshalPostExecTx(txs[last])
	t.Require().NoError(err, "the probe block must end with a 0x7D")
	payload, err := optypes.DecodePostExecPayload(postExecTx.Data)
	t.Require().NoError(err)
	entry := slices.IndexFunc(payload.GasRefundEntries, func(e optypes.SDMGasEntry) bool {
		return e.Index == uint64(probe.TransactionIndex)
	})
	t.Require().GreaterOrEqual(entry, 0, "the 0x7D must refund the probe tx")
	// The receipt reports gas after the refund.
	evmGasUsed := probe.GasUsed + payload.GasRefundEntries[entry].GasRefund
	payload.GasRefundEntries[entry].GasRefund = evmGasUsed + 1
	postExecTx.Data, err = rlp.EncodeToBytes(payload)
	t.Require().NoError(err)
	txs[last], err = postExecTx.MarshalBinary()
	t.Require().NoError(err)
	return txs
}
