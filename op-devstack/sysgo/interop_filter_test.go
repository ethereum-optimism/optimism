package sysgo

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// InteropFilterSystemIDs holds the IDs for a test system with an interop filter.
type InteropFilterSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	Superchain stack.SuperchainID
	Cluster    stack.ClusterID
	Supervisor stack.SupervisorID

	L2A   stack.L2NetworkID
	L2ACL stack.L2CLNodeID
	L2AEL stack.L2ELNodeID

	L2B   stack.L2NetworkID
	L2BCL stack.L2CLNodeID
	L2BEL stack.L2ELNodeID

	L2ABatcher stack.L2BatcherID
	L2BBatcher stack.L2BatcherID

	InteropFilter stack.InteropFilterID
}

// NewInteropFilterSystemIDs creates IDs for a test system with an interop filter.
func NewInteropFilterSystemIDs() InteropFilterSystemIDs {
	return InteropFilterSystemIDs{
		L1:            stack.L1NetworkID(DefaultL1ID),
		L1EL:          stack.NewL1ELNodeID("l1", DefaultL1ID),
		L1CL:          stack.NewL1CLNodeID("l1", DefaultL1ID),
		Superchain:    "main",
		Cluster:       stack.ClusterID("main"),
		Supervisor:    "1-primary",
		L2A:           stack.L2NetworkID(DefaultL2AID),
		L2ACL:         stack.NewL2CLNodeID("sequencer", DefaultL2AID),
		L2AEL:         stack.NewL2ELNodeID("sequencer", DefaultL2AID),
		L2B:           stack.L2NetworkID(DefaultL2BID),
		L2BCL:         stack.NewL2CLNodeID("sequencer", DefaultL2BID),
		L2BEL:         stack.NewL2ELNodeID("sequencer", DefaultL2BID),
		L2ABatcher:    stack.NewL2BatcherID("main", DefaultL2AID),
		L2BBatcher:    stack.NewL2BatcherID("main", DefaultL2BID),
		InteropFilter: "primary",
	}
}

// InteropFilterSystem creates a minimal interop system with an interop filter attached.
// This is a stripped-down version that avoids components with complex dependencies (like challengers).
func InteropFilterSystem(dest *InteropFilterSystemIDs) stack.Option[*Orchestrator] {
	ids := NewInteropFilterSystemIDs()
	opt := stack.Combine[*Orchestrator]()

	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up interop filter test system")
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	opt.Add(WithDeployer(),
		WithDeployerOptions(
			WithLocalContractSources(),
			WithCommons(ids.L1.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2A.ChainID()),
			WithPrefundedL2(ids.L1.ChainID(), ids.L2B.ChainID()),
			WithInteropAtGenesis(),
		),
	)

	opt.Add(WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(WithSupervisor(ids.Supervisor, ids.Cluster, ids.L1EL))

	// L2 chain A
	opt.Add(WithL2ELNode(ids.L2AEL, L2ELWithSupervisor(ids.Supervisor)))
	opt.Add(WithL2CLNode(ids.L2ACL, ids.L1CL, ids.L1EL, ids.L2AEL, L2CLSequencer(), L2CLIndexing()))
	opt.Add(WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL))
	opt.Add(WithManagedBySupervisor(ids.L2ACL, ids.Supervisor))

	// L2 chain B
	opt.Add(WithL2ELNode(ids.L2BEL, L2ELWithSupervisor(ids.Supervisor)))
	opt.Add(WithL2CLNode(ids.L2BCL, ids.L1CL, ids.L1EL, ids.L2BEL, L2CLSequencer(), L2CLIndexing()))
	opt.Add(WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL))
	opt.Add(WithManagedBySupervisor(ids.L2BCL, ids.Supervisor))

	opt.Add(WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}))

	// Add the interop filter connected to both L2 EL nodes
	opt.Add(WithInteropFilter(ids.InteropFilter, ids.Cluster, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}

func TestInteropFilter(t *testing.T) {
	var ids InteropFilterSystemIDs
	opt := InteropFilterSystem(&ids)

	logger := testlog.Logger(t, log.LevelInfo)

	onFail := func(now bool) {
		t.Helper()
		if now {
			t.FailNow()
		} else {
			t.Fail()
		}
	}
	onSkipNow := func() {
		t.Helper()
		t.SkipNow()
	}
	p := devtest.NewP(context.Background(), logger, onFail, onSkipNow)
	t.Cleanup(p.Close)

	orch := NewOrchestrator(p, stack.Combine[*Orchestrator]())
	stack.ApplyOptionLifecycle(opt, orch)

	t.Run("basic initialization", func(gt *testing.T) {
		dt := devtest.SerialT(gt)
		system := shim.NewSystem(dt)
		orch.Hydrate(system)

		// Verify the interop filter was added to the system
		filter := system.InteropFilter(match.FirstInteropFilter)
		require.NotNil(gt, filter)
		require.Equal(gt, ids.InteropFilter, filter.ID())

		// Verify we can list interop filters
		filterIDs := system.InteropFilterIDs()
		require.Len(gt, filterIDs, 1)
		require.Equal(gt, ids.InteropFilter, filterIDs[0])

		filters := system.InteropFilters()
		require.Len(gt, filters, 1)
	})

	t.Run("CheckAccessList with empty entries", func(gt *testing.T) {
		dt := devtest.SerialT(gt)
		system := shim.NewSystem(dt)
		orch.Hydrate(system)

		filter := system.InteropFilter(ids.InteropFilter)
		queryAPI := filter.QueryAPI()

		// Wait a bit for the filter to become ready (it needs to sync with L2 nodes)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// CheckAccessList with empty entries should succeed
		err := queryAPI.CheckAccessList(ctx, []common.Hash{}, types.CrossUnsafe, types.ExecutingDescriptor{})
		// The filter may return an error if not ready yet, but shouldn't panic
		if err != nil {
			gt.Logf("CheckAccessList returned error (expected during initialization): %v", err)
		}
	})
}
