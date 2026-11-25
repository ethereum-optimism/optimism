package derivation_parity

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/testutils/parityreport"
)

// TestDerivationParityHonestBatchesAcrossHoloceneActivation is the control for the node layer:
// with the honest batcher's data, kona-node and op-node record the same safe head for every L1
// block, across Holocene activating after genesis.
func TestDerivationParityHonestBatchesAcrossHoloceneActivation(gt *testing.T) {
	recordWhenPassed(gt, &parityreport.Record{Case: "control", Forks: holoceneAfterGenesisForks, Agree: true})
	t := devtest.ParallelT(gt)
	sys := newParitySystem(t, withHoloceneAfterGenesis(holoceneOffset))
	rollupCfg := sys.L2Network.Escape().RollupConfig()
	holoceneBlock, err := rollupCfg.TargetBlockNumber(*rollupCfg.HoloceneTime)
	t.Require().NoError(err)

	sys.L2Batcher.Start()
	dsl.CheckAll(t,
		sys.OpNode.ReachedFn(safety.LocalSafe, holoceneBlock+2, 60),
		sys.Kona.ReachedFn(safety.LocalSafe, holoceneBlock+2, 60))
	sys.Kona.InSync(sys.OpNode, safety.LocalSafe, 30)

	sys.Kona.VerifySafeHeadDatabaseMatches(sys.OpNode)
	sys.OpNode.VerifySafeHeadDatabaseMatches(sys.Kona)
}
