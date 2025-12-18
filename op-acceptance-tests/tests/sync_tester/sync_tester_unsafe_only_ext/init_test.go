package sync_tester_unsafe_only_ext

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/sync_tester_ext_el"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestMain(m *testing.M) {
	// Target op-sepolia
	networkName := "op-sepolia"
	config, _ := sync_tester_ext_el.GetNetworkPreset(networkName)
	chainCfg := chaincfg.ChainByName(networkName)
	presets.DoMain(m,
		presets.WithExternalELWithSuperchainRegistry(config),
		// CL connected to sync tester EL is verifier
		presets.WithExecutionLayerSyncOnVerifiers(),
		// Make sync tester EL mock EL Sync
		presets.WithELSyncActive(),
		// Only rely on EL sync for unsafe gap filling
		presets.WithReqRespSyncDisabled(),
		presets.WithNoDiscovery(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithUnsafeOnly(),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			// For stopping derivation, not to advance safe heads
			cfg.Stopped = true
		})),
		// Sync tester EL at genesis
		presets.WithSyncTesterELInitialState(eth.FCUState{
			Latest:    chainCfg.Genesis.L2.Number,
			Safe:      chainCfg.Genesis.L2.Number,
			Finalized: chainCfg.Genesis.L2.Number,
		}),
	)
}
