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
	l1Proposer *dsl.EOA

	SuperRoots    dsl.SuperRootSource
	TestSequencer *dsl.TestSequencer

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode
	L1CL      *dsl.L1CLNode

	L2ChainA   *dsl.L2Network
	L2BatcherA *dsl.L2Batcher
	L2ELA      *dsl.L2ELNode
	L2CLA      *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FunderL1 *dsl.FunderEOA
	FunderA  *dsl.FunderEOA

	// May be nil if not using sysgo
	challengerConfig *challengerConfig.Config
	startZKProposer  func()
}

func (s *SingleChainInterop) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		s.L2ChainA,
	}
}

func (s *SingleChainInterop) DisputeGameFactory() *proofs.DisputeGameFactory {
	s.T.Require().NotNil(s.SuperRoots, "supernode not configured for this preset")
	return proofs.NewDisputeGameFactory(s.T, s.L1Network, s.L1EL.EthClient(), s.L2ChainA.DisputeGameFactoryProxyAddr(), nil, nil, s.SuperRoots, s.l1Proposer, s.challengerConfig)
}

func (s *SingleChainInterop) AnchorStateRegistry(l2Chain *dsl.L2Network) *dsl.AnchorStateRegistry {
	return dsl.NewAnchorStateRegistry(s.T, l2Chain, s.L1EL)
}

func (s *SingleChainInterop) AdvanceTime(amount time.Duration) {
	s.T.Require().NotNil(s.timeTravel, "attempting to advance time on incompatible system")
	s.L1EL.AdvanceTime(s.timeTravel, amount)
}

// StartZKProposer starts the kona-sp1-proposer after a system configured with
// WithZK and WithoutHonestProposer has seeded its initial dispute games.
func (s *SingleChainInterop) StartZKProposer() {
	s.T.Require().NotNil(s.startZKProposer, "ZK proposer is not configured")
	s.startZKProposer()
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

	FunderB *dsl.FunderEOA
}

func (s *SimpleInterop) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		s.L2ChainA, s.L2ChainB,
	}
}

func (s *SimpleInterop) proofValidationContext() (devtest.T, *dsl.L1ELNode, []*dsl.L2Network) {
	return s.T, s.L1EL, s.L2Networks()
}

// Supernode returns the op-supernode backing this system's super roots. SimpleInterop
// is always supernode-backed, so this exposes supernode-only test controls (e.g.
// interop pause/resume) that are not part of the SuperRootSource interface.
func (s *SimpleInterop) Supernode() *dsl.Supernode {
	sn, ok := s.SuperRoots.(*dsl.Supernode)
	s.T.Require().True(ok, "SimpleInterop super roots are not supernode-backed")
	return sn
}

func (s *SingleChainInterop) StandardBridge(l2Chain *dsl.L2Network) *dsl.StandardBridge {
	return dsl.NewStandardBridge(s.T, l2Chain, s.L1EL)
}

// NewSimpleInterop creates a fresh SimpleInterop target for the current
// test using the super-root proofs system backed by op-supernode.
func NewSimpleInterop(t devtest.T, opts ...Option) *SimpleInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSimpleInterop", opts, twoL2SupernodeProofsPresetSupportedOptionKinds)
	return simpleInteropFromSupernodeProofsRuntime(t, sysgo.NewTwoL2SupernodeProofsRuntimeWithConfig(t, true, presetCfg))
}

// NewSingleChainInterop creates a fresh SingleChainInterop target for the
// current test using the single-chain super-root proofs system backed by op-supernode.
func NewSingleChainInterop(t devtest.T, opts ...Option) *SingleChainInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSingleChainInterop", opts, supernodeProofsPresetSupportedOptionKinds)
	return singleChainInteropFromSupernodeProofsRuntime(t, sysgo.NewSingleChainSupernodeProofsRuntimeWithConfig(t, true, presetCfg))
}

// NewSimpleInteropIsthmusSuper creates a fresh SimpleInterop target for the current test
// using the Isthmus super-root system backed by op-supernode.
func NewSimpleInteropIsthmusSuper(t devtest.T, opts ...Option) *SimpleInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSimpleInteropIsthmusSuper", opts, twoL2SupernodeProofsPresetSupportedOptionKinds)
	return simpleInteropFromSupernodeProofsRuntime(t, sysgo.NewTwoL2SupernodeProofsRuntimeWithConfig(t, false, presetCfg))
}

// NewSingleChainInteropIsthmusSuper creates a fresh SingleChainInterop target for the
// current test using the single-chain Isthmus super-root system backed by op-supernode.
func NewSingleChainInteropIsthmusSuper(t devtest.T, opts ...Option) *SingleChainInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSingleChainInteropIsthmusSuper", opts, supernodeProofsPresetSupportedOptionKinds)
	return singleChainInteropFromSupernodeProofsRuntime(t, sysgo.NewSingleChainSupernodeProofsRuntimeWithConfig(t, false, presetCfg))
}

// NewSingleChainInteropNoSupernode creates a fresh SingleChainInterop target whose
// super roots are served by the single op-node's superroot_atTimestamp endpoint (no
// op-supernode). The op-challenger plays super-cannon-kona games against this op-node
// source. This exercises the "op-node as super root RPC" path end-to-end.
func NewSingleChainInteropNoSupernode(t devtest.T, opts ...Option) *SingleChainInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSingleChainInteropNoSupernode", opts, 0)
	return singleChainInteropNoSupernodeFromRuntime(t, sysgo.NewSingleChainInteropNoSupernodeSuperRootRuntimeWithConfig(t, presetCfg))
}

// NewSingleChainInteropNoSupernodeZKDispute creates a fresh SingleChainInterop target whose super
// roots are served by the single op-node's superroot_atTimestamp endpoint (no op-supernode),
// running an op-challenger that plays ZK dispute games against that op-node source. This exercises
// the "op-node as super root RPC" path for the ZK game end-to-end.
func NewSingleChainInteropNoSupernodeZKDispute(t devtest.T, opts ...Option) *SingleChainInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSingleChainInteropNoSupernodeZKDispute", opts, 0)
	return singleChainInteropNoSupernodeFromRuntime(t, sysgo.NewSingleChainInteropNoSupernodeZKDisputeRuntimeWithConfig(t, presetCfg))
}

// NewSingleChainInteropSuperRootAtGenesis creates a fresh SingleChainInterop
// target where SuperPermissionedDisputeGame is installed in the permissioned
// slot as part of the initial op-deployer apply - no post-deploy OPCMv2
// migration runs. This exercises the initial-deploy path for super-root
// dispute games tracked by ethereum-optimism/optimism#18729.
func NewSingleChainInteropSuperRootAtGenesis(t devtest.T, opts ...Option) *SingleChainInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSingleChainInteropSuperRootAtGenesis", opts, supernodeProofsPresetSupportedOptionKinds)
	return singleChainInteropFromSupernodeProofsRuntime(t, sysgo.NewSingleChainSuperRootAtGenesisRuntimeWithConfig(t, presetCfg))
}

// WithSuggestedInteropActivationOffset suggests a hardfork time offset to use.
// This is applied e.g. to the deployment if running against sysgo.
func WithSuggestedInteropActivationOffset(offset uint64) Option {
	return WithDeployerOptions(
		func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtOffset(forks.Lagoon, &offset)
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

// MinimalInteropNoSupernode is like Minimal but with interop contracts deployed.
// No supernode is running - this tests interop contract deployment with local finality.
type MinimalInteropNoSupernode struct {
	Minimal
}

// NewMinimalInteropNoSupernode creates a fresh MinimalInteropNoSupernode target for the
// current test.
func NewMinimalInteropNoSupernode(t devtest.T, opts ...Option) *MinimalInteropNoSupernode {
	_, _ = collectSupportedPresetConfig(t, "NewMinimalInteropNoSupernode", opts, 0)
	return &MinimalInteropNoSupernode{
		Minimal: *minimalFromRuntime(t, sysgo.NewMinimalInteropNoSupernodeRuntime(t)),
	}
}
