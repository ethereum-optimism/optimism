package sync_tester_ext_el

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var L2CLSyncMode = getSyncMode("L2_CL_SYNCMODE")

func TestSyncTesterExtEL(gt *testing.T) {
	t := devtest.SerialT(gt)

	if os.Getenv("CIRCLECI_PIPELINE_SCHEDULE_NAME") != "build_daily" && os.Getenv("CIRCLECI_PARAMETERS_SYNC_TEST_OP_NODE_DISPATCH") != "true" {
		t.Skipf("TestSyncTesterExtEL only runs on daily scheduled pipeline jobs: schedule=%s dispatch=%s", os.Getenv("CIRCLECI_PIPELINE_SCHEDULE_NAME"), os.Getenv("CIRCLECI_PARAMETERS_SYNC_TEST_OP_NODE_DISPATCH"))
	}

	l := t.Logger()
	require := t.Require()
	blocksToSync := uint64(20)
	sys, target := setupSystem(gt, t, blocksToSync)

	attempts := 500
	if L2CLSyncMode == sync.ELSync {
		// After EL Sync is finished, the FCU state will advance to target immediately so less attempts
		attempts = 5
		// Signal L2CL for triggering EL Sync
		// Must send consecutive three payloads due to default EL Sync policy
		for i := 2; i >= 0; i-- {
			sys.L2CL.SignalTarget(sys.L2ELReadOnly, target-uint64(i))
		}
	}

	// Test that we can get sync status from L2CL node
	l2CLSyncStatus := sys.L2CL.SyncStatus()
	require.NotNil(l2CLSyncStatus, "L2CL should have sync status")

	sys.L2CL.Reached(types.LocalUnsafe, target, attempts)

	l2CLSyncStatus = sys.L2CL.SyncStatus()
	require.NotNil(l2CLSyncStatus, "L2CL should have sync status")

	unsafeL2Ref := l2CLSyncStatus.UnsafeL2
	blk := sys.L2EL.BlockRefByNumber(unsafeL2Ref.Number)
	require.Equal(unsafeL2Ref.Hash, blk.Hash, "L2EL should be on the same block as L2CL")

	stSessions := sys.SyncTester.ListSessions()
	require.Equal(len(stSessions), 1, "expect exactly one session")

	stSession := sys.SyncTester.GetSession(stSessions[0])
	require.GreaterOrEqual(stSession.CurrentState.Latest, stSession.InitialState.Latest+blocksToSync, "SyncTester session Latest should be on the same block as L2CL")
	require.GreaterOrEqual(stSession.CurrentState.Safe, stSession.InitialState.Safe+blocksToSync, "SyncTester session Safe should be on the same block as L2CL")

	l.Info("SyncTester ExtEL test completed successfully", "l2cl_chain_id", sys.L2CL.ID().ChainID(), "l2cl_sync_status", l2CLSyncStatus)
}

// setupSystem initializes the system for the test and returns the system and the target block number of the session
func setupSystem(gt *testing.T, t devtest.T, blocksToSync uint64) (*presets.MinimalExternalEL, uint64) {
	// Initialize orchestrator
	orch, target := setupOrchestrator(gt, t, blocksToSync)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	// Extract the system components
	l2 := system.L2Network(match.L2ChainA)
	verifierCL := l2.L2CLNode(match.FirstL2CL)
	syncTester := l2.SyncTester(match.FirstSyncTester)

	sys := &presets.MinimalExternalEL{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L1EL:         dsl.NewL1ELNode(system.L1Network(match.FirstL1Network).L1ELNode(match.FirstL1EL)),
		L2Chain:      dsl.NewL2Network(l2, orch.ControlPlane()),
		L2CL:         dsl.NewL2CLNode(verifierCL, orch.ControlPlane()),
		L2ELReadOnly: dsl.NewL2ELNode(l2.L2ELNode(match.FirstL2EL), orch.ControlPlane()),
		L2EL:         dsl.NewL2ELNode(l2.L2ELNode(match.SecondL2EL), orch.ControlPlane()),
		SyncTester:   dsl.NewSyncTester(syncTester),
	}

	return sys, target
}

// setupOrchestrator initializes and configures the orchestrator for the test and returns the orchestrator and the target block number of the session
func setupOrchestrator(gt *testing.T, t devtest.T, blocksToSync uint64) (*sysgo.Orchestrator, uint64) {
	l := t.Logger()
	ctx := t.Ctx()
	require := t.Require()

	config, err := GetNetworkPreset(os.Getenv("NETWORK_PRESET"))
	require.NoError(err, "failed to initialize network preset")

	// Runtime configuration values
	l.Info("Runtime configuration values for TestSyncTesterExtEL")
	l.Info("NETWORK_PRESET", "value", os.Getenv("NETWORK_PRESET"))
	l.Info("L2_NETWORK_NAME", "value", config.L2NetworkName)
	l.Info("L1_CHAIN_ID", "value", config.L1ChainID)
	l.Info("L2_EL_ENDPOINT", "value", config.L2ELEndpoint)
	l.Info("L1_CL_BEACON_ENDPOINT", "value", config.L1CLBeaconEndpoint)
	l.Info("L1_EL_ENDPOINT", "value", config.L1ELEndpoint)
	l.Info("TAILSCALE_NETWORKING", "value", os.Getenv("TAILSCALE_NETWORKING"))
	l.Info("L2_CL_SYNCMODE", "value", L2CLSyncMode)

	// Setup orchestrator
	logger := testlog.Logger(gt, log.LevelInfo)
	onFail := func(now bool) {
		if now {
			gt.FailNow()
		} else {
			gt.Fail()
		}
	}
	onSkipNow := func() {
		gt.SkipNow()
	}
	p := devtest.NewP(ctx, logger, onFail, onSkipNow)
	gt.Cleanup(p.Close)

	// Fetch the latest block number from the remote L2EL node
	cl, err := ethclient.DialContext(ctx, config.L2ELEndpoint)
	require.NoError(err)
	latestBlock, err := cl.BlockByNumber(ctx, nil)
	require.NoError(err)

	initial := latestBlock.NumberU64() - 1000
	target := initial + blocksToSync
	l.Info("LATEST_BLOCK", "latest_block", latestBlock.NumberU64(), "session_initial_block", initial, "target_block", target)

	opt := presets.WithExternalELWithSuperchainRegistry(config)
	if L2CLSyncMode == sync.ELSync {
		chainCfg := chaincfg.ChainByName(config.L2NetworkName)
		if chainCfg == nil {
			panic(fmt.Sprintf("network %s not found in superchain registry", config.L2NetworkName))
		}
		opt = stack.Combine(opt,
			presets.WithExecutionLayerSyncOnVerifiers(),
			presets.WithELSyncActive(),
			presets.WithSyncTesterELInitialState(eth.FCUState{
				Latest: initial,
				Safe:   0,
				// Need to set finalized to genesis to unskip EL Sync
				Finalized: chainCfg.Genesis.L2.Number,
			}),
		)
	} else {
		opt = stack.Combine(opt,
			presets.WithSyncTesterELInitialState(eth.FCUState{
				Latest:    initial,
				Safe:      initial,
				Finalized: initial,
			}),
		)
	}

	var orch stack.Orchestrator = sysgo.NewOrchestrator(p, stack.SystemHook(opt))
	stack.ApplyOptionLifecycle(opt, orch)

	return orch.(*sysgo.Orchestrator), target
}

func getSyncMode(envVar string) sync.Mode {
	if value := os.Getenv(envVar); value == sync.ELSyncString {
		return sync.ELSync
	}
	return sync.CLSync
}
