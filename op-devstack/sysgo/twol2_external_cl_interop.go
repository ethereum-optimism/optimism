package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// twoL2VerifierNodeKey names the per-chain verifier slot registered under
// MultiChainNodeRuntime.Followers by the external-CL interop runtime.
const twoL2VerifierNodeKey = "verifier"

// NewTwoL2ExternalCLInteropRuntime builds a two-L2 interop world (shared L1,
// interop dependency set spanning both chains) where each chain runs a stock
// op-node sequencer with its own EL and batcher, plus a dedicated verifier EL
// whose consensus client comes from the injected L2CLFactory when one is set.
// Use delaySeconds=0 for interop at genesis, or a positive value to offset the
// Interop fork activation past genesis.
//
// No supernode and no supervisor are started: cross-chain safety verification
// is left to the (external) verifier consensus clients under test. Block
// production never depends on the external clients — the sequencers are stock
// op-nodes, mirroring the light-sequencer style of the supernode runtime.
func NewTwoL2ExternalCLInteropRuntime(t devtest.T, delaySeconds uint64) *MultiChainRuntime {
	return NewTwoL2ExternalCLInteropRuntimeWithConfig(t, delaySeconds, PresetConfig{})
}

// NewTwoL2ExternalCLInteropRuntimeWithConfig is NewTwoL2ExternalCLInteropRuntime
// with explicit preset configuration. cfg.L2CLFactory is consulted once per
// chain for the verifier CL slot; when it is nil or declines a slot, a stock
// op-node verifier is started against the same verifier EL so the runtime
// remains usable without any external product.
func NewTwoL2ExternalCLInteropRuntimeWithConfig(t devtest.T, delaySeconds uint64, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	wb, l1Net, l2ANet, l2BNet := buildTwoL2RuntimeWorld(t, keys, true, delaySeconds, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...)
	migration := newInteropMigrationState(wb)
	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock()
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClockConfig(t, l1Net, jwtPath, l1Clock, cfg)

	runtimeDepSet := wb.outFullCfgSet.DependencySet
	require.NotNil(runtimeDepSet, "two-L2 interop world must provide a dependency set")

	startChain := func(l2Net *L2Network) *MultiChainNodeRuntime {
		// Stock op-node sequencer with its own EL: block production never
		// depends on the externally provided verifier CL.
		seqEL := startSequencerEL(t, l2Net, jwtPath, jwtSecret, NewELNodeIdentity(0))
		seqCL := startL2CLNode(t, keys, l1Net, l2Net, l1EL, l1CL, seqEL, jwtSecret, l2CLNodeStartConfig{
			Key:           "sequencer",
			IsSequencer:   true,
			NoDiscovery:   true,
			EnableReqResp: true,
			DependencySet: runtimeDepSet,
			L2CLOptions:   cfg.GlobalL2CLOptions,
		})
		batcher := startMinimalBatcher(t, keys, l2Net, l1EL, seqCL, seqEL, cfg.BatcherOptions...)

		// Dedicated verifier EL (default kind selection, op-reth unless
		// overridden), driven by the factory-provided CL when one is set.
		verifierEL := startL2ELForKey(t, l2Net, jwtPath, jwtSecret, twoL2VerifierNodeKey, NewELNodeIdentity(0))
		verifierCL := startInteropVerifierCL(t, keys, l1Net, l2Net, l1EL, l1CL, verifierEL, jwtSecret, seqEL, runtimeDepSet, cfg.GlobalL2CLOptions, cfg.L2CLFactory)

		return &MultiChainNodeRuntime{
			Name:    l2Net.Name(),
			Network: l2Net,
			EL:      seqEL,
			CL:      seqCL,
			Batcher: batcher,
			Followers: map[string]*SingleChainNodeRuntime{
				twoL2VerifierNodeKey: newSingleChainNodeRuntime(twoL2VerifierNodeKey, false, verifierEL, verifierCL),
			},
		}
	}

	chainA := startChain(l2ANet)
	chainB := startChain(l2BNet)

	faucetService := startFaucetsForRPCs(t, keys, map[eth.ChainID]string{
		l1Net.ChainID():  l1EL.UserRPC(),
		l2ANet.ChainID(): chainA.EL.UserRPC(),
		l2BNet.ChainID(): chainB.EL.UserRPC(),
	})

	base := &MultiChainRuntime{
		Keys:          keys,
		Migration:     migration,
		DependencySet: runtimeDepSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains: map[string]*MultiChainNodeRuntime{
			"l2a": chainA,
			"l2b": chainB,
		},
		FaucetService: faucetService,
		TimeTravel:    timeTravelClock,
		DelaySeconds:  delaySeconds,
	}
	attachTestSequencerToRuntime(t, base, "test-sequencer-2l2-external-cl")

	t.Logger().Info("configured external-CL interop runtime",
		"genesis_time", l2ANet.RollupConfig().Genesis.L2Time,
		"delay_seconds", delaySeconds,
		"external_cl", cfg.L2CLFactory != nil,
	)
	return base
}

// startInteropVerifierCL starts the verifier CL slot for one chain of an
// interop world. The optional factory is consulted first, receiving the
// world's shared dependency set in the launch context; when the factory is nil
// or declines the slot, the ordinary client selection starts a stock op-node
// verifier against the same EL. This is the exact per-chain call the two-L2
// external-CL interop runtime makes for each verifier slot.
func startInteropVerifierCL(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	verifierEL L2ELNode,
	jwtSecret [32]byte,
	unsafeSourceEL L2ELNode,
	depSet depset.DependencySet,
	l2CLOpts []L2CLOption,
	factory L2CLFactory,
) L2CLNode {
	return startL2CLForKey(t, keys, l1Net, l2Net, l1EL, l1CL, verifierEL, jwtSecret, twoL2VerifierNodeKey, twoL2VerifierNodeKey, false, unsafeSourceEL.UserRPC(), depSet, l2CLOpts, factory, unsafeSourceEL)
}
