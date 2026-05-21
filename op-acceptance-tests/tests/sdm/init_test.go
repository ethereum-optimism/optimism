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

// sdmDisabledInteropOffset pushes Interop far past any reasonable acceptance-test
// window so the chain spec gates SDM off. Used by sdmEnabled=false tests.
const sdmDisabledInteropOffset uint64 = 3_600

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
	// SDM activation tracks the Interop hardfork in chain spec: both op-node derivation
	// and op-reth execution read IsInterop(timestamp) from the same chain spec. To toggle
	// SDM in tests, we toggle Interop activation rather than a separate node-level flag.
	var interopOffset *uint64
	if !sdmEnabled {
		off := sdmDisabledInteropOffset
		interopOffset = &off
	}
	sys := newSDMRethSystemWithInteropOffset(t, interopOffset, batcherOpts...)

	if sdmEnabled {
		t.Require().True(sys.L2Network.IsForkActive(forks.Interop),
			"Interop must be active for SDM-enabled tests; otherwise chain-spec gates SDM off")
	} else {
		t.Require().False(sys.L2Network.IsForkActive(forks.Interop),
			"Interop must be inactive for SDM-disabled tests; otherwise chain-spec gates SDM on")
	}
	return sys
}

// newSDMRethSystemWithInteropOffset builds the SDM system with an explicit Interop activation
// offset. Pass nil to leave Interop at genesis (SDM active from the first block); pass a non-nil
// offset (in seconds from L2 genesis) to schedule Interop later. Used by the boundary test that
// exercises the chain-spec gate across the activation timestamp.
func newSDMRethSystemWithInteropOffset(
	t devtest.T,
	interopOffset *uint64,
	batcherOpts ...sysgo.BatcherOption,
) *sdmRethSystem {
	var deployerOpts []sysgo.DeployerOption
	if interopOffset != nil {
		offset := *interopOffset
		deployerOpts = append(deployerOpts, func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtOffset(forks.Interop, &offset)
			}
		})
	}

	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth",
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
			},
			{
				ELKey:       "verifier-op-reth",
				CLKey:       "verifier",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: false,
			},
		},
		BatcherOptions:  batcherOpts,
		DeployerOptions: deployerOpts,
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
	return &sdmRethSystem{
		L1EL:         frontends.L1EL,
		L2EL:         frontends.L2Network.PrimaryEL(),
		L2CL:         frontends.L2Network.PrimaryCL(),
		L2Network:    frontends.L2Network,
		L2ELVerifier: verifierEL,
		L2CLVerifier: verifierCL,
		L2Batcher:    frontends.L2Batcher,
		FunderL2:     dsl.NewFunder(wallet, frontends.FaucetL2, frontends.L2Network.PrimaryEL()),
	}
}

func withSingularBatcher(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
	cfg.BatchType = derive.SingularBatchType
}
