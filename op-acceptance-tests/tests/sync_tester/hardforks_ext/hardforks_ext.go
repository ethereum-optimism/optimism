package hardforks_ext

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
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
func setupOrchestrator(gt *testing.T, t devtest.T, blk uint64) *sysgo.Orchestrator {
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

	// Create orchestrator with the same configuration that was in TestMain
	opt := sysgo.DefaultMinimalExternalELSystemWithEndpointAndSuperchainRegistry(&sysgo.DefaultMinimalExternalELSystemIDs{}, l1CLBeaconEndpoint, l1ELEndpoint, l2ELEndpoint, L1ChainID, L2NetworkName, eth.FCUState{
		Latest:    blk,
		Safe:      blk,
		Finalized: blk,
	})

	orch := sysgo.NewOrchestrator(p, stack.SystemHook(opt))
	stack.ApplyOptionLifecycle[*sysgo.Orchestrator](opt, orch)

	return orch
}

func SyncTesterHFSExt(gt *testing.T, upgradeName rollup.ForkName) {
	t := devtest.SerialT(gt)
	l := t.Logger()

	// Initial block number to sync from before the upgrade
	blk := networkUpgradeBlocks[upgradeName] - 5

	// Initialize orchestrator
	orch := setupOrchestrator(gt, t, blk)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	l2 := system.L2Network(match.L2ChainA)
	verifierCL := l2.L2CLNode(match.FirstL2CL)
	syncTester := l2.SyncTester(match.FirstSyncTester)

	sys := &struct {
		L2CL       *dsl.L2CLNode
		L2EL       *dsl.L2ELNode
		SyncTester *dsl.SyncTester
		L2         *dsl.L2Network
	}{
		L2CL:       dsl.NewL2CLNode(verifierCL, orch.ControlPlane()),
		L2EL:       dsl.NewL2ELNode(l2.L2ELNode(match.FirstL2EL), orch.ControlPlane()),
		SyncTester: dsl.NewSyncTester(syncTester),
		L2:         dsl.NewL2Network(l2, orch.ControlPlane()),
	}
	require := t.Require()

	l2CLSyncStatus := sys.L2CL.WaitForNonZeroUnsafeTime(t.Ctx())

	ft := sys.L2.Escape().RollupConfig().ActivationTimeFor(upgradeName)
	require.Less(l2CLSyncStatus.UnsafeL2.Time, *ft, "L2CL unsafe time should be less than fork timestamp before upgrade")

	blocksToSync := uint64(10)
	targetBlock := blk + blocksToSync
	sys.L2CL.Reached(types.LocalUnsafe, targetBlock, 1000)
	l.Info("L2CL unsafe reached", "targetBlock", targetBlock, "upgrade_name", upgradeName)
	sys.L2CL.Reached(types.LocalSafe, targetBlock, 1000)
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
