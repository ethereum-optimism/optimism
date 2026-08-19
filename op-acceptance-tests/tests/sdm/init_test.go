package sdm

import (
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sdm/sdmtest"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

func newSDMRethSystem(t devtest.T, sdmEnabled bool) *sdmtest.RethSystem {
	return newSDMRethSystemWithBatcherOptions(t, sdmEnabled)
}

func newSDMRethSystemWithBatcherOptions(t devtest.T, sdmEnabled bool, batcherOpts ...sysgo.BatcherOption) *sdmtest.RethSystem {
	// SDM rides the Lagoon hardfork. The stock constructor covers the disabled and
	// null-policy regressions while still provisioning the dependency set required
	// whenever Lagoon is scheduled.
	return buildSDMRethSystem(t, sdmEnabled, false, false, nil, batcherOpts...)
}

func newFixtureSDMRethSystem(t devtest.T, batcherOpts ...sysgo.BatcherOption) *sdmtest.RethSystem {
	return buildSDMRethSystem(t, true, true, false, nil, batcherOpts...)
}

// newSDMRethSystemWithIsolatedVerifier builds the SDM system (Interop/SDM at genesis) with the
// verifier kept off the L2 P2P mesh. The verifier receives no gossiped unsafe blocks and must
// advance its safe head purely by deriving from L1, which forces op-node down the force-build path
// (FCU-with-attributes, `no_tx_pool = true`) instead of consolidating against an already-present
// unsafe block. That is the only path on which the verifier's EL rebuilds a derived PostExec block
// locally.
func newSDMRethSystemWithIsolatedVerifier(t devtest.T) *sdmtest.RethSystem {
	// kona-node gates derivation behind EL-sync completion, and a verifier only marks EL-sync
	// complete after its first engine forkchoiceUpdated — which is bootstrapped by an unsafe
	// payload received over L2 P2P. With P2P fully isolated, that bootstrap never happens: the
	// derivation actor stays in AwaitingELSyncCompletion, the EL gets no engine calls, and the
	// safe head never leaves genesis. kona-node has no L1-only bootstrap of the force-build path
	// today, so this test cannot pass under kona-node and is op-node-only for now.
	if sysgo.ResolveMixedL2CLKind() == sysgo.MixedL2CLKona {
		t.Skip("isolated-verifier force-build path is not supported by kona-node (no L1-only EL-sync bootstrap); op-node only")
	}
	return buildSDMRethSystem(t, true, true, true, nil)
}

// newSDMRethSystemWithLagoonOffset builds the SDM system with Lagoon scheduled at the given
// offset (in seconds) from L2 genesis. Used by the boundary test that exercises the chain-spec
// gate across the activation timestamp; pass `nil` for genesis activation.
func newSDMRethSystemWithLagoonOffset(
	t devtest.T,
	lagoonOffset *uint64,
	batcherOpts ...sysgo.BatcherOption,
) *sdmtest.RethSystem {
	var deployerOpts []sysgo.DeployerOption
	if lagoonOffset != nil {
		offset := *lagoonOffset
		// Take the InteropAtGenesis path so the runtime builds an interop
		// dependency set for op-node; then override the Lagoon fork offset
		// to schedule activation in the future rather than at genesis.
		deployerOpts = append(deployerOpts, func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtOffset(forks.Lagoon, &offset)
			}
		})
		return buildSDMRethSystem(t, true, true, false, deployerOpts, batcherOpts...)
	}
	return buildSDMRethSystem(t, false, true, false, deployerOpts, batcherOpts...)
}

func buildSDMRethSystem(
	t devtest.T,
	interopAtGenesis bool,
	fixtureSequencer bool,
	isolateVerifier bool,
	deployerOpts []sysgo.DeployerOption,
	batcherOpts ...sysgo.BatcherOption,
) *sdmtest.RethSystem {
	sysgo.SkipOnOpGeth(t, "SDM acceptance tests require op-reth post-exec support")

	// Honor DEVSTACK_L2CL_KIND so the kona acceptance suite exercises this test with
	// kona-node on both the sequencer and verifier (defaults to op-node when unset).
	clKind := sysgo.ResolveMixedL2CLKind()

	sequencerKey := "sequencer-op-reth"
	// The two sequencer paths are disjoint, and neither is an extension point for a
	// refund-producing binary:
	//
	//   - Stock path (below): plain op-reth on both nodes, backing the disabled and null-policy
	//     regressions. These assert that op-reth produces no PostExec transaction at all, so a
	//     refund-producing DEVSTACK_L2EL_OVERRIDE_BINARY would fail them by construction.
	//   - Fixture path (`fixtureSequencer`): pins op-reth-sdm-fixture as the producer against a
	//     stock op-reth verifier, so any block the fixture produces that stock op-reth rejects
	//     fails the run.
	//
	// A downstream suite exercising its own refund policy does not run this package; it mirrors
	// the `sdmtest` workload semantics in its own harness.
	sequencerOpts := sysgo.ResolveMixedL2ELOpts(t)
	if fixtureSequencer {
		sequencerKey = "sequencer-op-reth-sdm-fixture"
		sequencerOpts = []sysgo.OpRethOption{
			sysgo.OpRethWithBinary("op-reth-sdm-fixture"),
			sysgo.OpRethWithoutProofsHistory(),
		}
	}

	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       sequencerKey,
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      clKind,
				IsSequencer: true,
				OpRethOpts:  sequencerOpts,
			},
			{
				ELKey:            "verifier-op-reth",
				CLKey:            "verifier",
				ELKind:           sysgo.MixedL2ELOpReth,
				CLKind:           clKind,
				IsSequencer:      false,
				IsolateFromL2P2P: isolateVerifier,
			},
		},
		BatcherOptions:   batcherOpts,
		DeployerOptions:  deployerOpts,
		InteropAtGenesis: interopAtGenesis,
	})
	return sdmtest.FinishRethSystem(t, runtime, interopAtGenesis)
}

func withSingularBatcher(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
	cfg.BatchType = derive.SingularBatchType
}

func withCrossActivationSpanBatcher(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
	cfg.BatchType = derive.SpanBatchType
	// The batcher starts stopped, then catches up from genesis after Lagoon activates. Keep all
	// accumulated blocks in one span so the submitted span necessarily crosses the boundary.
	cfg.MaxBlocksPerSpanBatch = 1_000
}
