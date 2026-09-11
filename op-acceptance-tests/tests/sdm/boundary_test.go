package sdm

import (
	"testing"
	"time"

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
	sdmtest.VerifySDMFixture(t, sys.L2EL)
	sdmtest.VerifyOpReth(t, sys.L2ELVerifier)
	sys.L2CL.StartSequencer()

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

// TestSDMActivatesAtLagoonBoundary exercises the chain-spec-driven SDM gate across
// the Lagoon activation timestamp. Both layers read the same chain spec, so activating
// Lagoon mid-run must enable the fixture mechanism without a node-level fork override.
//
// Phase 1 (pre-Lagoon): the block must not contain a PostExec transaction.
// Phase 2 (post-Lagoon): the fixture must produce a PostExec transaction with refunds.
func TestSDMActivatesAtLagoonBoundary(gt *testing.T) {
	t := devtest.SerialT(gt)
	offset := boundaryLagoonOffset
	sys := newSDMRethSystemWithLagoonOffset(t, &offset)
	sdmtest.VerifySDMFixture(t, sys.L2EL)
	sdmtest.VerifyOpReth(t, sys.L2ELVerifier)

	t.Require().False(sys.L2Network.IsForkActive(forks.Lagoon),
		"Lagoon must not be active yet at the start of the boundary test")

	// Phase 1: queue funding transactions while sequencing is stopped. Waiting for both to enter
	// the mempool before starting the sequencer guarantees they land in block 1, independently of
	// how much wall-clock time system setup consumed.
	const preUserTxCount = 2
	fundedUsersCh := make(chan []*dsl.EOA, 1)
	go func() {
		defer close(fundedUsersCh)
		fundedUsersCh <- sys.FunderL2.NewFundedEOAs(preUserTxCount, eth.OneEther)
	}()
	sys.L2EL.WaitForPendingNonceMatch(sys.FunderL2.Address(), preUserTxCount, 100, 100*time.Millisecond)
	sys.L2CL.StartSequencer()
	t.Require().Len(<-fundedUsersCh, preUserTxCount, "pre-Lagoon users must be funded")

	const preBlockNum = 1
	preBlock := sdmtest.GetBlockWithTxs(t, sys.L2EL, preBlockNum)
	t.Require().GreaterOrEqual(len(preBlock.Transactions), preUserTxCount+1,
		"pre-Lagoon block must contain the L1-info deposit and funding transactions")
	preRef := sys.L2EL.BlockRefByNumber(preBlockNum)
	t.Require().False(sys.L2Network.IsForkActiveAt(forks.Lagoon, preRef.Time),
		"pre-Lagoon workload block %d (ts=%d) must land before Lagoon activation",
		preBlockNum, preRef.Time)

	prePostExecTx, _ := sdmpkg.FindPostExecTransaction(preBlock)
	t.Require().Nil(prePostExecTx,
		"pre-Lagoon block %d must not contain a PostExec tx; chain-spec gates SDM off", preBlockNum)

	// Phase 2: wait for Lagoon activation, then drive the workload again.
	activationBlock := sys.L2Network.AwaitActivation(t, forks.Lagoon)
	t.Logger().Info("Lagoon activated", "block", activationBlock)
	t.Require().True(sys.L2Network.IsForkActive(forks.Lagoon),
		"Lagoon must be active after AwaitActivation returns")

	postBlock, postIncluded, postBlockNum := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(postIncluded), 2, "post-Lagoon target block must contain user txs")
	postRef := sys.L2EL.BlockRefByNumber(postBlockNum)
	t.Require().True(sys.L2Network.IsForkActiveAt(forks.Lagoon, postRef.Time),
		"post-Lagoon workload block %d (ts=%d) must land after Lagoon activation",
		postBlockNum, postRef.Time)

	postPostExecTx, _ := sdmpkg.FindPostExecTransaction(postBlock)
	t.Require().NotNil(postPostExecTx,
		"post-Lagoon block %d must contain a PostExec tx; chain-spec gates SDM on", postBlockNum)
	t.Require().Equal(uint64(optypes.PostExecTxType), uint64(postPostExecTx.Type),
		"post-exec tx type must be 0x7D")

	payload, err := optypes.DecodePostExecPayload(postPostExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode")
	t.Require().Equal(optypes.PostExecPayloadVersion, payload.Version,
		"post-exec payload version must be 1")
	t.Require().NotEmpty(payload.GasRefundEntries,
		"post-exec payload must carry refund entries for the fixture workload")
	assertFixtureBlockOracle(t, sys, postBlock, postBlockNum)

	sys.L2Batcher.Start()
	dsl.CheckAll(t,
		sys.L2CLVerifier.ReachedRefFn(safety.CrossSafe, postRef.ID(), 120),
		sys.L2ELVerifier.ReachedFn(eth.Safe, postBlockNum, 120),
	)
	verifierRef := sys.L2ELVerifier.BlockRefByNumber(postBlockNum)
	t.Require().Equal(postRef.Hash, verifierRef.Hash,
		"stock verifier must safely derive the post-activation fixture block")
}
