package sdm

import (
	"testing"

	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// boundaryInteropOffset schedules Interop a few blocks after L2 genesis so the test
// can observe both pre- and post-activation block production within a short window.
// At 2s block time, 30s ≈ 15 blocks of pre-activation runway.
const boundaryInteropOffset uint64 = 30

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
	offset := boundaryInteropOffset
	sys := newSDMRethSystemWithInteropOffset(t, &offset)
	verifyOpReth(t, sys.L2EL)

	t.Require().False(sys.L2Network.IsForkActive(forks.Interop),
		"Interop must not be active yet at the start of the boundary test")

	// Phase 1: pre-Interop workload. We may need a few attempts to land the densest
	// block before the activation timestamp; mustFindRepeatedSlotBlock retries
	// internally and findPostExecTransaction tolerates absence.
	preBlock, preIncluded, preBlockNum := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(preIncluded), 2, "pre-Interop target block must contain user txs")
	preRef := sys.L2EL.BlockRefByNumber(preBlockNum)
	t.Require().False(sys.L2Network.IsForkActiveAt(forks.Interop, preRef.Time),
		"pre-Interop workload block %d (ts=%d) must land before Interop activation",
		preBlockNum, preRef.Time)

	prePostExecTx, _ := findPostExecTransaction(preBlock)
	t.Require().Nil(prePostExecTx,
		"pre-Interop block %d must not contain a PostExec tx; chain-spec gates SDM off", preBlockNum)

	// Phase 2: wait for Interop activation, then drive the workload again.
	activationBlock := sys.L2Network.AwaitActivation(t, forks.Interop)
	t.Logger().Info("Interop activated", "block", activationBlock)
	t.Require().True(sys.L2Network.IsForkActive(forks.Interop),
		"Interop must be active after AwaitActivation returns")

	postBlock, postIncluded, postBlockNum := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(postIncluded), 2, "post-Interop target block must contain user txs")
	postRef := sys.L2EL.BlockRefByNumber(postBlockNum)
	t.Require().True(sys.L2Network.IsForkActiveAt(forks.Interop, postRef.Time),
		"post-Interop workload block %d (ts=%d) must land after Interop activation",
		postBlockNum, postRef.Time)

	postPostExecTx, _ := findPostExecTransaction(postBlock)
	t.Require().NotNil(postPostExecTx,
		"post-Interop block %d must contain a PostExec tx; chain-spec gates SDM on", postBlockNum)
	t.Require().Equal(uint64(sdmpkg.SDMTxType), uint64(postPostExecTx.Type),
		"post-exec tx type must be 0x7D")

	payload, err := sdmpkg.DecodePayload(postPostExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode")
	t.Require().Equal(sdmpkg.PostExecPayloadVersion, payload.Version,
		"post-exec payload version must be 1")
	t.Require().NotEmpty(payload.GasRefundEntries,
		"post-exec payload must carry refund entries for the repeated-slot workload")
}
