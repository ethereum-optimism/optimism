package presets

import (
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	challengerConfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

type SingleChainInterop struct {
	Log        log.Logger
	T          devtest.T
	timeTravel *clock.AdvancingClock

	Supervisor    *dsl.Supervisor
	SuperRoots    *dsl.Supernode
	TestSequencer *dsl.TestSequencer

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode
	L1CL      *dsl.L1CLNode

	L2ChainA   *dsl.L2Network
	L2BatcherA *dsl.L2Batcher
	L2ELA      *dsl.L2ELNode
	L2CLA      *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FaucetA  *dsl.Faucet
	FaucetL1 *dsl.Faucet
	FunderL1 *dsl.Funder
	FunderA  *dsl.Funder

	// May be nil if not using sysgo
	challengerConfig *challengerConfig.Config
}

// NewSingleChainInterop creates a fresh SingleChainInterop target for the current test.
//
// The target is created from the single-chain interop runtime plus any additional preset options.
func NewSingleChainInterop(t devtest.T, opts ...Option) *SingleChainInterop {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainInterop", opts, singleChainInteropPresetSupportedOptionKinds)
	out := singleChainInteropFromRuntime(t, sysgo.NewSingleChainInteropRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}

func (s *SingleChainInterop) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		s.L2ChainA,
	}
}

func (s *SingleChainInterop) DisputeGameFactory() *proofs.DisputeGameFactory {
	s.T.Require().NotNil(s.SuperRoots, "supernode not configured for this preset")
	return proofs.NewDisputeGameFactory(s.T, s.L1Network, s.L1EL.EthClient(), s.L2ChainA.DisputeGameFactoryProxyAddr(), nil, nil, s.SuperRoots, s.challengerConfig)
}

func (s *SingleChainInterop) AdvanceTime(amount time.Duration) {
	s.T.Require().NotNil(s.timeTravel, "attempting to advance time on incompatible system")
	s.timeTravel.AdvanceTime(amount)
}

func (s *SingleChainInterop) proofValidationContext() (devtest.T, *dsl.L1ELNode, []*dsl.L2Network) {
	return s.T, s.L1EL, []*dsl.L2Network{s.L2ChainA}
}

type SimpleInterop struct {
	SingleChainInterop

	L2ChainB   *dsl.L2Network
	L2BatcherB *dsl.L2Batcher
	L2ELB      *dsl.L2ELNode
	L2CLB      *dsl.L2CLNode

	FaucetB *dsl.Faucet
	FunderB *dsl.Funder
}

func (s *SimpleInterop) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		s.L2ChainA, s.L2ChainB,
	}
}

func (s *SimpleInterop) proofValidationContext() (devtest.T, *dsl.L1ELNode, []*dsl.L2Network) {
	return s.T, s.L1EL, s.L2Networks()
}

func (s *SingleChainInterop) StandardBridge(l2Chain *dsl.L2Network) *dsl.StandardBridge {
	return dsl.NewStandardBridge(s.T, l2Chain, s.L1EL)
}

// NewSimpleInteropSuperProofs creates a fresh SimpleInterop target for the current test
// using the default super-root proofs system.
func NewSimpleInteropSuperProofs(t devtest.T, opts ...Option) *SimpleInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSimpleInteropSuperProofs", opts, simpleInteropSuperProofsPresetSupportedOptionKinds)
	return simpleInteropFromRuntime(t, sysgo.NewSimpleInteropSuperProofsRuntimeWithConfig(t, presetCfg))
}

// NewSimpleInteropSupernodeProofs creates a fresh SimpleInterop target for the current
// test using the super-root proofs system backed by op-supernode.
func NewSimpleInteropSupernodeProofs(t devtest.T, opts ...Option) *SimpleInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSimpleInteropSupernodeProofs", opts, supernodeProofsPresetSupportedOptionKinds)
	return simpleInteropFromSupernodeProofsRuntime(t, sysgo.NewTwoL2SupernodeProofsRuntimeWithConfig(t, true, presetCfg))
}

// NewSingleChainInteropSupernodeProofs creates a fresh SingleChainInterop target for the
// current test using the single-chain super-root proofs system backed by op-supernode.
func NewSingleChainInteropSupernodeProofs(t devtest.T, opts ...Option) *SingleChainInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSingleChainInteropSupernodeProofs", opts, supernodeProofsPresetSupportedOptionKinds)
	return singleChainInteropFromSupernodeProofsRuntime(t, sysgo.NewSingleChainSupernodeProofsRuntimeWithConfig(t, true, presetCfg))
}

// NewSimpleInteropIsthmusSuper creates a fresh SimpleInterop target for the current test
// using the Isthmus super-root system backed by op-supernode.
func NewSimpleInteropIsthmusSuper(t devtest.T, opts ...Option) *SimpleInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSimpleInteropIsthmusSuper", opts, supernodeProofsPresetSupportedOptionKinds)
	return simpleInteropFromSupernodeProofsRuntime(t, sysgo.NewTwoL2SupernodeProofsRuntimeWithConfig(t, false, presetCfg))
}

// NewSingleChainInteropIsthmusSuper creates a fresh SingleChainInterop target for the
// current test using the single-chain Isthmus super-root system backed by op-supernode.
func NewSingleChainInteropIsthmusSuper(t devtest.T, opts ...Option) *SingleChainInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSingleChainInteropIsthmusSuper", opts, supernodeProofsPresetSupportedOptionKinds)
	return singleChainInteropFromSupernodeProofsRuntime(t, sysgo.NewSingleChainSupernodeProofsRuntimeWithConfig(t, false, presetCfg))
}

// NewSimpleInterop creates a fresh SimpleInterop target for the current test.
//
// The target is created from the interop runtime plus any additional preset options.
func NewSimpleInterop(t devtest.T, opts ...Option) *SimpleInterop {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSimpleInterop", opts, singleChainInteropPresetSupportedOptionKinds)
	out := simpleInteropFromRuntime(t, sysgo.NewSimpleInteropRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}

// WithSuggestedInteropActivationOffset suggests a hardfork time offset to use.
// This is applied e.g. to the deployment if running against sysgo.
func WithSuggestedInteropActivationOffset(offset uint64) Option {
	return WithDeployerOptions(
		func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtOffset(forks.Interop, &offset)
			}
		},
	)
}

// WithSequencingWindow suggests a sequencing window to use, and checks the maximum sequencing window.
// The sequencing windows are expressed in number of L1 execution-layer blocks till sequencing window expiry.
// This is applied to runtime deployment/config validation.
func WithSequencingWindow(suggestedSequencingWindow uint64, maxSequencingWindow uint64) Option {
	return option{
		kinds: optionKindDeployer | optionKindMaxSequencingWindow,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.DeployerOptions = append(cfg.DeployerOptions, sysgo.WithSequencingWindow(suggestedSequencingWindow))
			v := maxSequencingWindow
			cfg.MaxSequencingWindow = &v
		},
	}
}

// WithInteropNotAtGenesis adds a test-gate that checks
// if the interop hardfork is configured at a non-genesis time.
func WithInteropNotAtGenesis() Option {
	return WithRequireInteropNotAtGenesis()
}

type MultiSupervisorInterop struct {
	SimpleInterop

	// Supervisor does not support multinode so need a additional supervisor for verifier nodes
	SupervisorSecondary *dsl.Supervisor

	L2ELA2 *dsl.L2ELNode
	L2CLA2 *dsl.L2CLNode
	L2ELB2 *dsl.L2ELNode
	L2CLB2 *dsl.L2CLNode
}

// NewMultiSupervisorInterop initializes a fresh multi-supervisor interop target for the
// current test.
func NewMultiSupervisorInterop(t devtest.T, opts ...Option) *MultiSupervisorInterop {
	_, _ = collectSupportedPresetConfig(t, "NewMultiSupervisorInterop", opts, 0)
	return multiSupervisorInteropFromRuntime(t, sysgo.NewMultiSupervisorInteropRuntime(t))
}

// MinimalInteropNoSupervisor is like Minimal but with interop contracts deployed.
// No supervisor is running - this tests interop contract deployment with local finality.
type MinimalInteropNoSupervisor struct {
	Minimal
}

// NewMinimalInteropNoSupervisor creates a fresh MinimalInteropNoSupervisor target for the
// current test.
func NewMinimalInteropNoSupervisor(t devtest.T, opts ...Option) *MinimalInteropNoSupervisor {
	_, _ = collectSupportedPresetConfig(t, "NewMinimalInteropNoSupervisor", opts, 0)
	return &MinimalInteropNoSupervisor{
		Minimal: *minimalFromRuntime(t, sysgo.NewMinimalInteropNoSupervisorRuntime(t)),
	}
}
