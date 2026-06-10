package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

// postExecMode selects which trailing PostExec transaction (if any) the batcher
// injects into the test block when it is encoded into the span batch.
type postExecMode int

const (
	// noPostExec leaves the block untouched: user transactions only.
	noPostExec postExecMode = iota
	// validPostExec appends a well-formed, refund-bearing PostExec tx anchored to
	// the block it lives in.
	validPostExec
	// invalidPostExec appends a PostExec tx whose payload is anchored to the wrong
	// block number, which kona's executor rejects (InvalidPostExecPayload).
	invalidPostExec
)

// TestSDMPostExecDerivation exercises op-node derivation of span batches carrying
// SDM (Sequencer-Defined Metering) PostExec transactions across the Lagoon
// activation boundary. Lagoon gates SDM and Interop together
// (IsSDM == IsInterop == IsLagoon), so activating it turns both on at once.
//
// This is a derivation-only test: the fault-proof program is intentionally NOT
// run, for the same reason as the Lagoon branch of TestActivationBlockNUTBundle —
// the single-chain interop fault-proof host wiring has not landed yet
// (https://github.com/ethereum-optimism/optimism/issues/21114, item 4). Interop
// fault proofs are covered in op-acceptance-tests via kona-host super.
//
// IMPORTANT HARNESS LIMITATION: the action-test L2 engine is op-geth, which
// recognizes the PostExec tx *type* but has no SDM *execution* (no refund
// metering, no payload validation — see core/state_processor.go). A PostExec tx
// therefore fails execution in the engine queue, and Holocene falls back to a
// deposit-only block. Consequently BOTH validPostExec and invalidPostExec
// degrade to a deposit-only block here; the payload-validity distinction is an
// execution-layer concern verified against op-reth + kona in
// op-acceptance-tests/tests/sdm. What this test pins on the op-node side is:
//   - PostExec txs pass the span-batch SDM type-gate once Lagoon is active, and
//   - an unexecutable payload is replaced by a deposit-only block (and the safe
//     head still advances), rather than stalling derivation.
func TestSDMPostExecDerivation(gt *testing.T) {
	run := func(gt *testing.T, testCfg *helpers.TestCfg[postExecMode]) {
		t := actionsHelpers.NewDefaultTesting(gt)

		// Schedule Lagoon (= SDM + Interop) after genesis so the test crosses the
		// activation boundary. Genesis sits at the preceding fork (Karst).
		lagoonOffset := uint64(4)
		setup := func(dc *genesis.DeployConfig) {
			dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
			dc.SetForkTimeOffset(forks.Lagoon, &lagoonOffset)
		}
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), setup)

		// Build empty L2 blocks until SDM/Interop activates, then make them safe.
		env.Miner.ActEmptyBlock(t)
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActBuildL2ToFork(t, forks.Lagoon)
		require.True(t, env.Sd.RollupCfg.IsSDM(env.Sequencer.L2Unsafe().Time),
			"SDM must be active once Lagoon has activated")
		env.BatchMineAndSync(t)

		// Build the block under test: a single user transaction (Alice -> Bob).
		env.Alice.L2.ActResetTxOpts(t)
		env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
		env.Alice.L2.ActMakeTx(t)
		env.Sequencer.ActL2StartBlock(t)
		env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
		env.Sequencer.ActL2EndBlock(t)
		testBlock := env.Sequencer.L2Unsafe()

		// Submit the test block to L1 and derive it back. The honest block can ride
		// the BatchMineAndSync helper; the PostExec variants cannot, because the
		// synthetic PostExec tx must be injected with a batcher block modifier
		// (op-geth can't sequence it) and BatchMineAndSync / ActSubmitAll buffer
		// every block unmodified. Those drive the batcher directly, mirroring
		// holocene_invalid_batch_test.go.
		if testCfg.Custom == noPostExec {
			env.BatchMineAndSync(t)
		} else {
			var pe *types.Transaction
			if testCfg.Custom == validPostExec {
				// Valid payload anchored to this block, offering a small refund to the
				// user tx at index 1 (index 0 is the L1-info deposit).
				pe = postExecTx(t, sdmpkg.PostExecPayload{
					Version:          sdmpkg.PostExecPayloadVersion,
					BlockNumber:      testBlock.Number,
					GasRefundEntries: []sdmpkg.SDMGasEntry{{Index: 1, GasRefund: 1}},
				})
			} else {
				// Wrong block number: kona rejects this as "does not match block number".
				pe = postExecTx(t, sdmpkg.PostExecPayload{
					Version:     sdmpkg.PostExecPayloadVersion,
					BlockNumber: testBlock.Number + 99,
				})
			}

			env.Batcher.ActL2BatchBuffer(t, actionsHelpers.WithBlockModifier(appendTx(pe)))
			env.Batcher.ActL2ChannelClose(t)
			env.Batcher.ActL2BatchSubmit(t)
			env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
			env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
			env.Miner.ActL1EndBlock(t)
			env.Sequencer.ActL1HeadSignal(t)
			env.Sequencer.ActL2PipelineFull(t)
		}

		// In every case the safe head advances to the test block's height.
		safe := env.Sequencer.L2Safe()
		require.Equal(t, testBlock.Number, safe.Number, "safe head should advance to the test block height")
		safeBlock := env.Engine.L2Chain().GetBlockByHash(safe.Hash)
		require.NotNil(t, safeBlock, "safe block must be present in the engine")

		if testCfg.Custom == noPostExec {
			// The honest block derives verbatim: same hash, user tx retained.
			require.Equal(t, testBlock.Hash, safe.Hash, "no-PostExec block should derive verbatim")
			require.Len(t, safeBlock.Transactions(), 2, "block should hold the L1-info deposit + 1 user tx")
			require.False(t, safeBlock.Transactions()[1].IsDepositTx(), "second tx should be the user tx")
			return
		}

		// op-geth cannot execute a PostExec tx, so the engine queue rejects the
		// payload and Holocene installs a deposit-only block in its place.
		require.NotEqual(t, testBlock.Hash, safe.Hash, "PostExec block must be replaced on derivation")
		for i, tx := range safeBlock.Transactions() {
			require.Truef(t, tx.IsDepositTx(), "deposit-only fallback must contain only deposits, tx %d is not a deposit", i)
		}
		require.NotEmpty(t,
			env.Logs.FindLogs(testlog.NewMessageContainsFilter("deposits-only")),
			"derivation should fall back to deposits-only attributes")
	}

	matrix := helpers.NewMatrix[postExecMode]()
	base := helpers.NewForkMatrix(helpers.Karst)
	matrix.AddTestCase("NoPostExec", noPostExec, base, run, helpers.ExpectNoError())
	matrix.AddTestCase("ValidPostExec", validPostExec, base, run, helpers.ExpectNoError())
	matrix.AddTestCase("InvalidPostExec", invalidPostExec, base, run, helpers.ExpectNoError())
	matrix.Run(gt)
}

// postExecTx builds a synthetic SDM PostExec transaction (tx type 0x7D) wrapping
// the RLP-encoded payload.
func postExecTx(t actionsHelpers.Testing, payload sdmpkg.PostExecPayload) *types.Transaction {
	data, err := rlp.EncodeToBytes(&payload)
	require.NoError(t, err, "encode post-exec payload")
	return types.NewTx(&types.PostExecTx{Data: data})
}

// appendTx returns a BlockModifier that appends extra as the trailing transaction
// of the block, leaving the header untouched (re-derivation recomputes roots).
func appendTx(extra *types.Transaction) actionsHelpers.BlockModifier {
	return func(block *types.Block) *types.Block {
		body := block.Body()
		body.Transactions = append(body.Transactions, extra)
		return block.WithBody(*body)
	}
}
