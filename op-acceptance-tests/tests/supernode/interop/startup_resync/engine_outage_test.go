package startup_resync

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestSupernodeChainContainerEngineOutageIsolation proves that losing one
// chain's execution engine does not stop the sibling chain container. Chain B
// keeps building and deriving local-safe blocks while chain A's engine is down,
// and neither chain rewrites the canonical history produced before recovery.
func TestSupernodeChainContainerEngineOutageIsolation(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()
	sys := presets.NewTwoL2SupernodeInterop(t, 0,
		presets.WithUniformL2BlockTimes(l2BlockTime),
	)

	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.LocalUnsafe, 4, 60),
		sys.L2BCL.ReachedFn(safety.LocalUnsafe, 4, 60),
		sys.L2ACL.ReachedFn(safety.LocalSafe, 3, 60),
		sys.L2BCL.ReachedFn(safety.LocalSafe, 3, 60),
	)
	beforeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	beforeB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	beforeSafeB := sys.L2BCL.HeadBlockRef(safety.LocalSafe)

	sys.L2ELA.Stop()
	t.Cleanup(sys.L2ELA.Start)

	dsl.CheckAll(t,
		sys.L2BCL.ReachedFn(safety.LocalUnsafe, beforeB.Number+4, 90),
		sys.L2BCL.ReachedFn(safety.LocalSafe, beforeSafeB.Number+2, 90),
	)
	duringB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	canonicalB := make(map[uint64]common.Hash, duringB.Number-beforeB.Number+1)
	for number := beforeB.Number; number <= duringB.Number; number++ {
		canonicalB[number] = sys.L2ELB.BlockRefByNumber(number).Hash
	}

	sys.L2ELA.Start()
	sys.L2ELA.WaitForOnline()
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.LocalUnsafe, beforeA.Number+2, 120),
		sys.L2BCL.ReachedFn(safety.LocalUnsafe, duringB.Number+2, 60),
		sys.L2ACL.ReachedFn(safety.CrossSafe, beforeA.Number+1, 120),
		sys.L2BCL.ReachedFn(safety.CrossSafe, beforeB.Number+1, 120),
	)

	require.Equal(beforeA.Hash, sys.L2ELA.BlockRefByNumber(beforeA.Number).Hash,
		"chain A rewrote its pre-outage canonical head during recovery")
	require.Equal(beforeA.Hash, sys.L2ELA.BlockRefByNumber(beforeA.Number+1).ParentHash,
		"chain A did not resume contiguously after its engine restarted")
	for number, expected := range canonicalB {
		require.Equal(expected, sys.L2ELB.BlockRefByNumber(number).Hash,
			"chain B canonical hash changed at outage-produced height %d", number)
	}
}
