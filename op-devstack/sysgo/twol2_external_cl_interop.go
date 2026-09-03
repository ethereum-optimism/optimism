package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

const twoL2VerifierNodeKey = "verifier"

// NewTwoL2ExternalCLInteropRuntime builds a two-chain interop world with a
// stock sequencer and batcher per chain plus a dedicated verifier EL/CL slot.
// It deliberately starts no supernode or supervisor.
func NewTwoL2ExternalCLInteropRuntime(t devtest.T, delaySeconds uint64) *MultiChainRuntime {
	return NewTwoL2ExternalCLInteropRuntimeWithConfig(t, delaySeconds, PresetConfig{})
}

// NewTwoL2ExternalCLInteropRuntimeWithConfig is the configured form of
// NewTwoL2ExternalCLInteropRuntime. L2CLFactory is consulted for each verifier
// slot and ordinary client selection is used when it is nil or declines.
func NewTwoL2ExternalCLInteropRuntimeWithConfig(t devtest.T, delaySeconds uint64, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	wb, l1Net, l2ANet, l2BNet := buildTwoL2RuntimeWorld(
		t, keys, true, delaySeconds, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...,
	)
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
		// Apply target-scoped op-reth options before either EL starts. This is
		// where callers attach stable interop-admission proxies without an EL
		// restart/reconfiguration hook.
		elOpts := append(append([]OpRethOption{}, ResolveMixedL2ELOpts(t)...), cfg.OpRethOptions...)
		seqEL := startSequencerEL(t, l2Net, jwtPath, jwtSecret, NewELNodeIdentity(0), elOpts...)
		seqCL := startL2CLNode(t, keys, l1Net, l2Net, l1EL, l1CL, seqEL, jwtSecret, l2CLNodeStartConfig{
			Key:           "sequencer",
			IsSequencer:   true,
			NoDiscovery:   true,
			EnableReqResp: true,
			DependencySet: runtimeDepSet,
			L2CLOptions:   cfg.GlobalL2CLOptions,
		})
		batcher := startMinimalBatcher(t, keys, l2Net, l1EL, seqCL, seqEL, cfg.BatcherOptions...)

		verifierEL := startL2ELForKey(
			t, l2Net, jwtPath, jwtSecret, twoL2VerifierNodeKey, NewELNodeIdentity(0), elOpts...,
		)
		// The stock follow path imports forkchoice from the sequencer CL but
		// still needs the paired execution peer for block bodies. Keep the same
		// edge for external verifiers so handled and fallback slots share one
		// topology.
		connectL2ELPeers(t, t.Logger(), seqEL.UserRPC(), verifierEL.UserRPC())
		verifierCL := startInteropVerifierCL(
			t, keys, l1Net, l2Net, l1EL, l1CL, verifierEL, jwtSecret,
			seqEL, seqCL, runtimeDepSet, cfg.GlobalL2CLOptions, cfg.L2CLFactory,
		)

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
	base := &MultiChainRuntime{
		Keys:          keys,
		Migration:     migration,
		FullConfigSet: wb.outFullCfgSet,
		DependencySet: runtimeDepSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains: map[string]*MultiChainNodeRuntime{
			"l2a": chainA,
			"l2b": chainB,
		},
		TimeTravel:   timeTravelClock,
		DelaySeconds: delaySeconds,
	}
	attachTestSequencerToRuntime(t, base, "test-sequencer-2l2-external-cl")
	return base
}

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
	unsafeSourceCL L2CLNode,
	depSet depset.DependencySet,
	l2CLOpts []L2CLOption,
	factory L2CLFactory,
) L2CLNode {
	stockFollowSource := interopVerifierFollowSource(unsafeSourceCL)
	return startL2CLForKeyWithKind(
		t, keys, l1Net, l2Net, l1EL, l1CL, verifierEL, jwtSecret,
		twoL2VerifierNodeKey, twoL2VerifierNodeKey, false, stockFollowSource,
		depSet, l2CLOpts, factory, unsafeSourceEL, false, MixedL2CLOpNode,
	).Node
}

func interopVerifierFollowSource(unsafeSourceCL L2CLNode) string {
	return unsafeSourceCL.UserRPC()
}
