package sdm

import (
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type sdmRethSystem struct {
	L1EL         *dsl.L1ELNode
	L2EL         *dsl.L2ELNode
	L2CL         *dsl.L2CLNode
	L2Network    *dsl.L2Network
	L2ELVerifier *dsl.L2ELNode
	L2CLVerifier *dsl.L2CLNode
	L2Batcher    *dsl.L2Batcher
	FunderL2     *dsl.Funder
}

func newSDMRethSystem(t devtest.T, sdmEnabled bool) *sdmRethSystem {
	return newSDMRethSystemWithBatcherOptions(t, sdmEnabled)
}

func newSDMRethSystemWithBatcherOptions(t devtest.T, sdmEnabled bool, batcherOpts ...sysgo.BatcherOption) *sdmRethSystem {
	// SDM rides the Interop hardfork: enabling Interop at genesis turns SDM on across
	// op-node derivation, op-reth execution, and the op-rbuilder payload builder. The
	// runtime also provisions a DependencySet for op-node, required whenever Interop is
	// scheduled, even in single-chain setups without a supervisor.
	return buildSDMRethSystem(t, sdmEnabled, nil, batcherOpts...)
}

// newSDMRethSystemWithInteropOffset builds the SDM system with Interop scheduled at the given
// offset (in seconds) from L2 genesis. Used by the boundary test that exercises the chain-spec
// gate across the activation timestamp; pass `nil` for genesis activation.
func newSDMRethSystemWithInteropOffset(
	t devtest.T,
	interopOffset *uint64,
	batcherOpts ...sysgo.BatcherOption,
) *sdmRethSystem {
	var deployerOpts []sysgo.DeployerOption
	if interopOffset != nil {
		offset := *interopOffset
		// Take the InteropAtGenesis path so the runtime builds an Interop
		// dependency set for op-node; then override the Interop fork offset
		// to schedule activation in the future rather than at genesis.
		deployerOpts = append(deployerOpts, func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtOffset(forks.Lagoon, &offset)
			}
		})
		return buildSDMRethSystem(t, true, deployerOpts, batcherOpts...)
	}
	return buildSDMRethSystem(t, false, deployerOpts, batcherOpts...)
}

func buildSDMRethSystem(t devtest.T, interopAtGenesis bool, deployerOpts []sysgo.DeployerOption, batcherOpts ...sysgo.BatcherOption) *sdmRethSystem {
	sysgo.SkipOnOpGeth(t, "SDM acceptance tests require op-reth post-exec support")

	// Honor DEVSTACK_L2CL_KIND so the kona acceptance suite exercises this test with
	// kona-node on both the sequencer and verifier (defaults to op-node when unset).
	clKind := sysgo.ResolveMixedL2CLKind()

	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth",
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      clKind,
				IsSequencer: true,
			},
			{
				ELKey:       "verifier-op-reth",
				CLKey:       "verifier",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      clKind,
				IsSequencer: false,
			},
		},
		BatcherOptions:   batcherOpts,
		DeployerOptions:  deployerOpts,
		InteropAtGenesis: interopAtGenesis,
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	frontends.L2Batcher.Stop()
	t.Require().Len(frontends.Nodes, 2, "SDM op-reth system must include sequencer and verifier nodes")

	var verifierEL *dsl.L2ELNode
	var verifierCL *dsl.L2CLNode
	for _, node := range frontends.Nodes {
		if !node.Spec.IsSequencer {
			verifierEL = node.EL
			verifierCL = node.CL
			break
		}
	}
	t.Require().NotNil(verifierEL, "missing SDM verifier EL node")
	t.Require().NotNil(verifierCL, "missing SDM verifier CL node")

	wallet := dsl.NewRandomHDWallet(t, 30)
	sys := &sdmRethSystem{
		L1EL:         frontends.L1EL,
		L2EL:         frontends.L2Network.PrimaryEL(),
		L2CL:         frontends.L2Network.PrimaryCL(),
		L2Network:    frontends.L2Network,
		L2ELVerifier: verifierEL,
		L2CLVerifier: verifierCL,
		L2Batcher:    frontends.L2Batcher,
		FunderL2:     dsl.NewFunder(wallet, frontends.FaucetL2, frontends.L2Network.PrimaryEL()),
	}

	// The protocol gate (Interop hardfork) is already scheduled above when
	// interopAtGenesis is true. Local PostExec production additionally requires the
	// sequencer's op-reth to be opted in via admin_setSdmPostExecOptIn; nothing else
	// flips this on. Verifier nodes do not need to opt in — they accept PostExec
	// txs by chain spec rule alone.
	if interopAtGenesis {
		setSDMEnabled(t, sys.L2EL, true)
	}
	return sys
}

func withSingularBatcher(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
	cfg.BatchType = derive.SingularBatchType
}
