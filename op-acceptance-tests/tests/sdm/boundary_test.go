package sdm

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sdm/sdmtest"
	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// boundaryLagoonOffset schedules Lagoon a few blocks after L2 genesis so the test
// can observe both pre- and post-activation block production within a short window.
// At 2s block time, 30s ≈ 15 blocks of pre-activation runway.
const boundaryLagoonOffset uint64 = 30

// TestSDMPostExecSpanCrossesInteropBoundary verifies that a span can start before Lagoon,
// cross its activation block, and carry a genuinely produced PostExec payload afterwards.
func TestSDMPostExecSpanCrossesInteropBoundary(gt *testing.T) {
	t := devtest.SerialT(gt)
	offset := boundaryLagoonOffset
	sys := newSDMRethSystemWithLagoonOffset(t, &offset, withCrossActivationSpanBatcher)
	sdmtest.VerifyOpReth(t, sys.L2EL)
	sdmtest.VerifyOpReth(t, sys.L2ELVerifier)

	activationBlock := sys.L2Network.AwaitActivation(t, forks.Lagoon)
	activationRef := sys.L2EL.BlockRefByNumber(activationBlock.Number)

	block, included, postExecBlockNumber := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(included), 2,
		"post-Lagoon target block must contain repeated-slot transactions")
	postExecTx, _ := sdmpkg.FindPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "SDM producer must append a PostExec transaction")
	payload, err := optypes.DecodePostExecPayload(postExecTx.Input)
	t.Require().NoError(err, "produced PostExec payload must decode")
	t.Require().NotEmpty(payload.GasRefundEntries,
		"produced PostExec payload must contain repeated-slot gas refunds")
	postExecRef := sys.L2EL.BlockRefByNumber(postExecBlockNumber)

	// The batcher has been stopped since genesis. Starting it now submits the accumulated pre- and
	// post-Lagoon blocks in one span, including the genuinely produced PostExec payload above.
	sys.L2Batcher.Start()
	sdmtest.VerifyPostExecSpanCrossesActivation(
		t,
		sys.L2Network,
		4,
		activationRef.Time,
		postExecBlockNumber,
	)
	dsl.CheckAll(t,
		sys.L2CL.ReachedRefFn(safety.CrossSafe, postExecRef.ID(), 120),
		sys.L2CLVerifier.ReachedRefFn(safety.CrossSafe, postExecRef.ID(), 120),
		sys.L2EL.ReachedFn(eth.Safe, postExecBlockNumber, 120),
		sys.L2ELVerifier.ReachedFn(eth.Safe, postExecBlockNumber, 120),
	)
	verifierRef := sys.L2ELVerifier.BlockRefByNumber(postExecBlockNumber)
	t.Require().Equal(postExecRef.Hash, verifierRef.Hash,
		"verifier must derive the producer's PostExec block from the cross-Lagoon span")
}

// TestSDMActivatesAtInteropBoundary exercises the chain-spec-driven SDM gate across
// the Interop activation timestamp. Both layers (op-node derivation and op-reth
// execution) read IsInterop(timestamp) from the same chain spec, so flipping Interop
// active mid-run must flip SDM on without any node-level override.
//
// Phase 1 (pre-Interop): a repeated-slot workload lands in a block whose timestamp
// is before Interop activation; the block must not contain a PostExec (0x7D) tx.
//
// Phase 2 (post-Interop): the same workload after activation must produce a block
// containing a PostExec tx with refund entries.
func TestSDMActivatesAtInteropBoundary(gt *testing.T) {
	t := devtest.SerialT(gt)
	offset := boundaryLagoonOffset
	sys := newSDMRethSystemWithLagoonOffset(t, &offset)
	sdmtest.VerifyOpReth(t, sys.L2EL)

	t.Require().False(sys.L2Network.IsForkActive(forks.Lagoon),
		"Interop must not be active yet at the start of the boundary test")

	// Phase 1: pre-Interop workload. We may need a few attempts to land the densest
	// block before the activation timestamp; sdmtest.MustFindRepeatedSlotBlock retries
	// internally and sdmpkg.FindPostExecTransaction tolerates absence.
	preBlock, preIncluded, preBlockNum := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(preIncluded), 2, "pre-Interop target block must contain user txs")
	preRef := sys.L2EL.BlockRefByNumber(preBlockNum)
	t.Require().False(sys.L2Network.IsForkActiveAt(forks.Lagoon, preRef.Time),
		"pre-Interop workload block %d (ts=%d) must land before Interop activation",
		preBlockNum, preRef.Time)

	prePostExecTx, _ := sdmpkg.FindPostExecTransaction(preBlock)
	t.Require().Nil(prePostExecTx,
		"pre-Interop block %d must not contain a PostExec tx; chain-spec gates SDM off", preBlockNum)

	// Phase 2: wait for Interop activation, then drive the workload again.
	activationBlock := sys.L2Network.AwaitActivation(t, forks.Lagoon)
	t.Logger().Info("Interop activated", "block", activationBlock)
	t.Require().True(sys.L2Network.IsForkActive(forks.Lagoon),
		"Interop must be active after AwaitActivation returns")

	postBlock, postIncluded, postBlockNum := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(postIncluded), 2, "post-Interop target block must contain user txs")
	postRef := sys.L2EL.BlockRefByNumber(postBlockNum)
	t.Require().True(sys.L2Network.IsForkActiveAt(forks.Lagoon, postRef.Time),
		"post-Interop workload block %d (ts=%d) must land after Interop activation",
		postBlockNum, postRef.Time)

	postPostExecTx, _ := sdmpkg.FindPostExecTransaction(postBlock)
	t.Require().NotNil(postPostExecTx,
		"post-Interop block %d must contain a PostExec tx; chain-spec gates SDM on", postBlockNum)
	t.Require().Equal(uint64(optypes.PostExecTxType), uint64(postPostExecTx.Type),
		"post-exec tx type must be 0x7D")

	payload, err := optypes.DecodePostExecPayload(postPostExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode")
	t.Require().Equal(optypes.PostExecPayloadVersion, payload.Version,
		"post-exec payload version must be 1")
	t.Require().NotEmpty(payload.GasRefundEntries,
		"post-exec payload must carry refund entries for the repeated-slot workload")
}
