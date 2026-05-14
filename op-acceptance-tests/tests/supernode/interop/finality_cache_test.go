package interop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSupernodeInterop_FinalityCacheSurvivesEngineRPCOutage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	ctx := t.Ctx()

	sys := newSupernodeInteropWithTimeTravel(t, 0)

	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.CrossSafe, 10, 60),
		sys.L2BCL.ReachedFn(types.CrossSafe, 10, 60),
	)

	safeA := sys.L2ELA.BlockRefByLabel(eth.Safe)
	safeB := sys.L2ELB.BlockRefByLabel(eth.Safe)

	sys.AdvanceTime(90 * time.Second)
	sys.L1Network.WaitForFinalization()

	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.Finalized, safeA.Number, 60),
		sys.L2BCL.ReachedFn(types.Finalized, safeB.Number, 60),
	)

	finalizedA := sys.L2ACL.HeadBlockRef(types.Finalized)
	require.Greater(t, finalizedA.Number, uint64(0), "test needs a non-genesis finalized head cached")

	genesis := sys.L2ELA.BlockRefByNumber(0)

	sys.L2ELA.DisconnectEngineRPC()
	disconnected := true
	defer func() {
		if disconnected {
			sys.L2ELA.ReconnectEngineRPC()
		}
	}()

	sys.Supernode.PromoteFinalizedForTest(sys.L2A.ChainID(), genesis)

	sys.L2ELA.ReconnectEngineRPC()
	disconnected = false

	require.Eventually(t, func() bool {
		_, err := sys.L2ACL.Escape().RollupAPI().SyncStatus(ctx)
		return err == nil
	}, 15*time.Second, 500*time.Millisecond, "op-node should remain online after engine RPC outage")

	sys.L2ACL.Advanced(types.LocalUnsafe, 1, 30)
}
