package sysgo

import (
	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
)

// withAltDA starts a DA server backed by cfg.AltDAStore and returns cfg extended to run every L2
// chain on alt-DA with keccak commitments, with the batcher and every op-node started from the
// returned cfg using that DA server.
func withAltDA(t devtest.T, cfg PresetConfig) PresetConfig {
	t.Require().NotEqual(MixedL2CLKona, devstackL2CLKind(), "kona-node does not support alt-DA")

	server := altda.NewDAServer("127.0.0.1", 0, cfg.AltDAStore, t.Logger().New("component", "da-server"), false)
	t.Require().NoError(server.Start(), "failed to start DA server")
	t.Cleanup(func() {
		t.Require().NoError(server.Stop(), "failed to stop DA server")
	})
	url := server.HttpEndpoint()

	cfg.DeployerOptions = append(cfg.DeployerOptions, func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithAltDA(genesis.AltDADeployConfig{
				UseAltDA:          true,
				DACommitmentType:  altda.KeccakCommitmentString,
				DAChallengeWindow: 16,
				DAResolveWindow:   16,
				DABondSize:        1_000_000,
			})
		}
	})
	cfg.BatcherOptions = append(cfg.BatcherOptions, func(_ ComponentTarget, batcherCfg *bss.CLIConfig) {
		batcherCfg.AltDA = altda.CLIConfig{
			Enabled:               true,
			DAServerURL:           url,
			MaxConcurrentRequests: 1,
		}
	})
	cfg.GlobalL2CLOptions = append(cfg.GlobalL2CLOptions, L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, clCfg *L2CLConfig) {
		clCfg.AltDAServerURL = url
	}))
	return cfg
}
