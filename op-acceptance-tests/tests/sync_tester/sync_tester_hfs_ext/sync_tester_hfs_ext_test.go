package sync_tester_hfs_ext

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Configuration defaults for op-sepolia
const (
	DefaultL2NetworkName      = "op-sepolia"
	DefaultL1ChainID          = 11155111
	DefaultL2ELEndpoint       = "https://ci-sepolia-l2.optimism.io"
	DefaultL1CLBeaconEndpoint = "https://ci-sepolia-beacon.optimism.io"
	DefaultL1ELEndpoint       = "https://ci-sepolia-l1.optimism.io"

	// Tailscale networking endpoints
	DefaultL2ELEndpointTailscale       = "https://proxyd-l2-sepolia.primary.client.dev.oplabs.cloud"
	DefaultL1CLBeaconEndpointTailscale = "https://beacon-api-proxy-sepolia.primary.client.dev.oplabs.cloud"
	DefaultL1ELEndpointTailscale       = "https://proxyd-l1-sepolia.primary.client.dev.oplabs.cloud"
)

var (
	// Network upgrade block numbers for op-sepolia
	networkUpgradeBlocks = map[rollup.ForkName]uint64{
		rollup.Canyon:   4089330,
		rollup.Delta:    5700330,
		rollup.Ecotone:  8366130,
		rollup.Fjord:    12597930,
		rollup.Granite:  15837930,
		rollup.Holocene: 20415330,
		rollup.Isthmus:  26551530,
	}

	// Load configuration from environment variables with defaults
	L2NetworkName = getEnvOrDefault("L2_NETWORK_NAME", DefaultL2NetworkName)
	L1ChainID     = eth.ChainIDFromUInt64(getEnvUint64OrDefault("L1_CHAIN_ID", DefaultL1ChainID))

	// Default endpoints
	L2ELEndpoint       = getEnvOrDefault("L2_EL_ENDPOINT", DefaultL2ELEndpoint)
	L1CLBeaconEndpoint = getEnvOrDefault("L1_CL_BEACON_ENDPOINT", DefaultL1CLBeaconEndpoint)
	L1ELEndpoint       = getEnvOrDefault("L1_EL_ENDPOINT", DefaultL1ELEndpoint)
)

func TestSyncTesterHFS_Canyon_CLSync(gt *testing.T) {
	hfsExt(gt, rollup.Canyon, sync.CLSync)
}

func TestSyncTesterHFS_Canyon_ELSync(gt *testing.T) {
	hfsExt(gt, rollup.Canyon, sync.ELSync)
}

func TestSyncTesterHFS_Delta_CLSync(gt *testing.T) {
	hfsExt(gt, rollup.Delta, sync.CLSync)
}

func TestSyncTesterHFS_Delta_ELSync(gt *testing.T) {
	hfsExt(gt, rollup.Delta, sync.ELSync)
}

func TestSyncTesterHFS_Ecotone_CLSync(gt *testing.T) {
	hfsExt(gt, rollup.Ecotone, sync.CLSync)
}

func TestSyncTesterHFS_Ecotone_ELSync(gt *testing.T) {
	hfsExt(gt, rollup.Ecotone, sync.ELSync)
}

func TestSyncTesterHFS_Fjord_CLSync(gt *testing.T) {
	hfsExt(gt, rollup.Fjord, sync.CLSync)
}

func TestSyncTesterHFS_Fjord_ELSync(gt *testing.T) {
	hfsExt(gt, rollup.Fjord, sync.ELSync)
}

func TestSyncTesterHFS_Granite_CLSync(gt *testing.T) {
	hfsExt(gt, rollup.Granite, sync.CLSync)
}

func TestSyncTesterHFS_Granite_ELSync(gt *testing.T) {
	hfsExt(gt, rollup.Granite, sync.ELSync)
}

func TestSyncTesterHFS_Holocene_CLSync(gt *testing.T) {
	hfsExt(gt, rollup.Holocene, sync.CLSync)
}

func TestSyncTesterHFS_Holocene_ELSync(gt *testing.T) {
	hfsExt(gt, rollup.Holocene, sync.ELSync)
}

func TestSyncTesterHFS_Isthmus_CLSync(gt *testing.T) {
	hfsExt(gt, rollup.Isthmus, sync.CLSync)
}

func TestSyncTesterHFS_Isthmus_ELSync(gt *testing.T) {
	hfsExt(gt, rollup.Isthmus, sync.ELSync)
}

// getEnvOrDefault returns the environment variable value or the default if not set
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

// getEnvUint64OrDefault returns the environment variable value as uint64 or the default if not set
func getEnvUint64OrDefault(envVar string, defaultValue uint64) uint64 {
	if value := os.Getenv(envVar); value != "" {
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// setupOrchestrator initializes and configures the orchestrator for the test
func setupOrchestrator(gt *testing.T, t devtest.T, blk, targetBlock uint64, l2CLSyncMode sync.Mode) *sysgo.Orchestrator {
	l := t.Logger()

	// Override configuration with Tailscale endpoints if Tailscale networking is enabled
	l2ELEndpoint := L2ELEndpoint
	l1CLBeaconEndpoint := L1CLBeaconEndpoint
	l1ELEndpoint := L1ELEndpoint

	if os.Getenv("TAILSCALE_NETWORKING") == "true" {
		l2ELEndpoint = getEnvOrDefault("L2_EL_ENDPOINT_TAILSCALE", DefaultL2ELEndpointTailscale)
		l1CLBeaconEndpoint = getEnvOrDefault("L1_CL_BEACON_ENDPOINT_TAILSCALE", DefaultL1CLBeaconEndpointTailscale)
		l1ELEndpoint = getEnvOrDefault("L1_EL_ENDPOINT_TAILSCALE", DefaultL1ELEndpointTailscale)
	}

	// Setup orchestrator directly without TestMain
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
	p := devtest.NewP(context.Background(), logger, onFail, onSkipNow)
	gt.Cleanup(p.Close)

	// Runtime configuration values
	l.Info("Runtime configuration values for TestSyncTesterExtEL")
	l.Info("L2_NETWORK_NAME", "value", L2NetworkName)
	l.Info("L1_CHAIN_ID", "value", L1ChainID)
	l.Info("L2_EL_ENDPOINT", "value", l2ELEndpoint)
	l.Info("L1_CL_BEACON_ENDPOINT", "value", l1CLBeaconEndpoint)
	l.Info("L1_EL_ENDPOINT", "value", l1ELEndpoint)
	l.Info("TAILSCALE_NETWORKING", "value", os.Getenv("TAILSCALE_NETWORKING"))
	l.Info("L2_CL_SYNCMODE", "value", l2CLSyncMode)

	config := stack.ExtNetworkConfig{
		L2NetworkName:      L2NetworkName,
		L1ChainID:          L1ChainID,
		L2ELEndpoint:       L2ELEndpoint,
		L1CLBeaconEndpoint: L1CLBeaconEndpoint,
		L1ELEndpoint:       L1ELEndpoint,
	}

	// Create orchestrator with the same configuration that was in TestMain
	opt := presets.WithExternalELWithSuperchainRegistry(config)
	if l2CLSyncMode == sync.ELSync {
		chainCfg := chaincfg.ChainByName(config.L2NetworkName)
		if chainCfg == nil {
			panic(fmt.Sprintf("network %s not found in superchain registry", config.L2NetworkName))
		}
		opt = stack.Combine(opt,
			presets.WithExecutionLayerSyncOnVerifiers(),
			presets.WithELSyncTarget(targetBlock),
			presets.WithSyncTesterELInitialState(eth.FCUState{
				Latest: blk,
				Safe:   0,
				// Need to set finalized to genesis to unskip EL Sync
				Finalized: chainCfg.Genesis.L2.Number,
			}),
		)
	} else {
		opt = stack.Combine(opt,
			presets.WithSyncTesterELInitialState(eth.FCUState{
				Latest:    blk,
				Safe:      blk,
				Finalized: blk,
			}),
		)
	}

	var orch stack.Orchestrator = sysgo.NewOrchestrator(p, stack.SystemHook(opt))
	stack.ApplyOptionLifecycle(opt, orch)

	return orch.(*sysgo.Orchestrator)
}

func hfsExt(gt *testing.T, upgradeName rollup.ForkName, l2CLSyncMode sync.Mode) {
	t := devtest.ParallelT(gt)
	l := t.Logger()

	// Initial block number to sync from before the upgrade
	blk := networkUpgradeBlocks[upgradeName] - 5

	blocksToSync := uint64(10)
	targetBlock := blk + blocksToSync
	// Initialize orchestrator

	orch := setupOrchestrator(gt, t, blk, targetBlock, l2CLSyncMode)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	l2 := system.L2Network(match.L2ChainA)
	verifierCL := l2.L2CLNode(match.FirstL2CL)
	syncTester := l2.SyncTester(match.FirstSyncTester)

	sys := &struct {
		L2CL         *dsl.L2CLNode
		L2ELReadOnly *dsl.L2ELNode
		L2EL         *dsl.L2ELNode
		SyncTester   *dsl.SyncTester
		L2           *dsl.L2Network
	}{
		L2CL:         dsl.NewL2CLNode(verifierCL, orch.ControlPlane()),
		L2ELReadOnly: dsl.NewL2ELNode(l2.L2ELNode(match.FirstL2EL), orch.ControlPlane()),
		L2EL:         dsl.NewL2ELNode(l2.L2ELNode(match.SecondL2EL), orch.ControlPlane()),
		SyncTester:   dsl.NewSyncTester(syncTester),
		L2:           dsl.NewL2Network(l2, orch.ControlPlane()),
	}
	require := t.Require()

	ft := sys.L2.Escape().RollupConfig().ActivationTimeFor(upgradeName)
	var l2CLSyncStatus *eth.SyncStatus
	attempts := 1000
	if l2CLSyncMode == sync.ELSync {
		// After EL Sync is finished, the FCU state will advance to target immediately so less attempts
		attempts = 5
		// Signal L2CL for finishing EL Sync
		sys.L2CL.SignalTarget(sys.L2ELReadOnly, targetBlock)
	} else {
		l2CLSyncStatus := sys.L2CL.WaitForNonZeroUnsafeTime(t.Ctx())
		require.Less(l2CLSyncStatus.UnsafeL2.Time, *ft, "L2CL unsafe time should be less than fork timestamp before upgrade")
	}

	sys.L2CL.Reached(types.LocalUnsafe, targetBlock, attempts)
	l.Info("L2CL unsafe reached", "targetBlock", targetBlock, "upgrade_name", upgradeName)
	sys.L2CL.Reached(types.LocalSafe, targetBlock, attempts)
	l.Info("L2CL safe reached", "targetBlock", targetBlock, "upgrade_name", upgradeName)

	l2CLSyncStatus = sys.L2CL.SyncStatus()
	require.NotNil(l2CLSyncStatus, "L2CL should have sync status")
	require.Greater(l2CLSyncStatus.UnsafeL2.Time, *ft, "L2CL unsafe time should be greater than fork timestamp after upgrade")

	unsafeL2Ref := l2CLSyncStatus.UnsafeL2
	ref := sys.L2EL.BlockRefByNumber(unsafeL2Ref.Number)
	require.Equal(unsafeL2Ref.Hash, ref.Hash, "L2EL should be on the same block as L2CL")

	stSessions := sys.SyncTester.ListSessions()
	require.Equal(len(stSessions), 1, "expect exactly one session")

	stSession := sys.SyncTester.GetSession(stSessions[0])
	require.GreaterOrEqualf(stSession.CurrentState.Latest, stSession.InitialState.Latest+blocksToSync, "SyncTester session CurrentState.Latest only advanced %d", stSession.CurrentState.Latest-stSession.InitialState.Latest)
	require.GreaterOrEqualf(stSession.CurrentState.Safe, stSession.InitialState.Safe+blocksToSync, "SyncTester session CurrentState.Safe only advanced %d", stSession.CurrentState.Safe-stSession.InitialState.Safe)

	l.Info("SyncTester HFS Ext test completed successfully", "l2cl_chain_id", sys.L2CL.ID().ChainID(), "l2cl_sync_status", l2CLSyncStatus, "upgrade_name", upgradeName)
}
