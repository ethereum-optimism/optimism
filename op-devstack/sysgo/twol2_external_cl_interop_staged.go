package sysgo

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// StagedTwoL2ExternalCLInteropRuntime owns the immutable configuration for a
// two-chain interop world while leaving every network service stopped until an
// explicit Start method is called. It is intended for interactive launchers
// that need process state to match a user-visible sequence of actions.
//
// Construction performs only deterministic genesis/config generation and file
// preparation. It does not bind an RPC listener or start an L1, EL, CL,
// batcher, faucet, supervisor, or test sequencer.
type StagedTwoL2ExternalCLInteropRuntime struct {
	t   devtest.T
	cfg PresetConfig

	keys       devkeys.Keys
	dependency depset.DependencySet
	l1Net      *L1Network
	l2Nets     map[string]*L2Network
	jwtPath    string
	jwtSecret  [32]byte

	l1EL L1ELNode
	l1CL *L1CLNode

	sequencerEL map[string]L2ELNode
	verifierEL  map[string]L2ELNode
	sequencerCL map[string]L2CLNode
	verifierCL  map[string]L2CLNode
}

// NewStagedTwoL2ExternalCLInteropRuntime prepares the same chain configuration
// as NewTwoL2ExternalCLInteropRuntimeWithConfig without starting services.
func NewStagedTwoL2ExternalCLInteropRuntime(t devtest.T, delaySeconds uint64, cfg PresetConfig) *StagedTwoL2ExternalCLInteropRuntime {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err, "failed to derive dev keys from mnemonic")
	wb, l1Net, l2ANet, l2BNet := buildTwoL2RuntimeWorld(
		t, keys, true, delaySeconds, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...,
	)
	dependency := wb.outFullCfgSet.DependencySet
	t.Require().NotNil(dependency, "two-L2 interop world must provide a dependency set")
	jwtPath, jwtSecret := writeJWTSecret(t)
	return &StagedTwoL2ExternalCLInteropRuntime{
		t: t, cfg: cfg, keys: keys, dependency: dependency,
		l1Net:   l1Net,
		l2Nets:  map[string]*L2Network{"901": l2ANet, "902": l2BNet},
		jwtPath: jwtPath, jwtSecret: jwtSecret,
		sequencerEL: make(map[string]L2ELNode), verifierEL: make(map[string]L2ELNode),
		sequencerCL: make(map[string]L2CLNode), verifierCL: make(map[string]L2CLNode),
	}
}

func (r *StagedTwoL2ExternalCLInteropRuntime) network(chainID string) *L2Network {
	network := r.l2Nets[chainID]
	r.t.Require().NotNil(network, "unknown staged L2 chain %s", chainID)
	return network
}

// StartL1 starts the shared execution and beacon services exactly once.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartL1() (L1ELNode, *L1CLNode) {
	r.t.Require().Nil(r.l1EL, "staged L1 is already running")
	r.l1EL, r.l1CL = startInProcessL1WithClockConfig(r.t, r.l1Net, r.jwtPath, clock.SystemClock, r.cfg)
	return r.l1EL, r.l1CL
}

// StartSequencerEL starts one chain's sequencer execution engine.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartSequencerEL(chainID string, opts ...OpRethOption) L2ELNode {
	r.t.Require().NotNil(r.l1EL, "start the staged L1 before L2 execution")
	r.t.Require().Nil(r.sequencerEL[chainID], "sequencer EL for chain %s already exists", chainID)
	node := startSequencerEL(r.t, r.network(chainID), r.jwtPath, r.jwtSecret, NewELNodeIdentity(0), opts...)
	r.sequencerEL[chainID] = node
	return node
}

// StartVerifierEL starts one chain's developer-facing verifier engine.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartVerifierEL(chainID string, opts ...OpRethOption) L2ELNode {
	r.t.Require().NotNil(r.l1EL, "start the staged L1 before L2 execution")
	r.t.Require().Nil(r.verifierEL[chainID], "verifier EL for chain %s already exists", chainID)
	node := startL2ELForKey(r.t, r.network(chainID), r.jwtPath, r.jwtSecret, twoL2VerifierNodeKey, NewELNodeIdentity(0), opts...)
	r.verifierEL[chainID] = node
	return node
}

// StartSequencerCL attaches an external sequencer CL to one started sequencer
// engine. Shared-process factories may defer the actual process launch until
// this method has been called for every configured chain.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartSequencerCL(chainID string, factory L2CLFactory) L2CLNode {
	r.requireL1()
	el := r.sequencerEL[chainID]
	r.t.Require().NotNil(el, "start chain %s sequencer EL before its CL", chainID)
	r.t.Require().Nil(r.sequencerCL[chainID], "sequencer CL for chain %s already exists", chainID)
	node := startL2CLForKey(
		r.t, r.keys, r.l1Net, r.network(chainID), r.l1EL, r.l1CL, el, r.jwtSecret,
		"sequencer", "sequencer", true, "", r.dependency, r.cfg.GlobalL2CLOptions, factory, nil,
	)
	r.sequencerCL[chainID] = node
	return node
}

// StartVerifierCL attaches an external verifier CL to one started verifier
// engine and identifies the matching sequencer EL as its chain-local unsafe
// source. The factory remains responsible for selecting Direct Sync or EL
// follow policy from this product-neutral context.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartVerifierCL(chainID string, factory L2CLFactory) L2CLNode {
	r.requireL1()
	verifier := r.verifierEL[chainID]
	sequencer := r.sequencerEL[chainID]
	r.t.Require().NotNil(verifier, "start chain %s verifier EL before its CL", chainID)
	r.t.Require().NotNil(sequencer, "start chain %s sequencer EL before its verifier CL", chainID)
	r.t.Require().NotNil(r.sequencerCL[chainID], "start chain %s sequencer CL before its verifier CL", chainID)
	r.t.Require().Nil(r.verifierCL[chainID], "verifier CL for chain %s already exists", chainID)
	node := startL2CLForKey(
		r.t, r.keys, r.l1Net, r.network(chainID), r.l1EL, r.l1CL, verifier, r.jwtSecret,
		twoL2VerifierNodeKey, twoL2VerifierNodeKey, false, sequencer.UserRPC(), r.dependency,
		r.cfg.GlobalL2CLOptions, factory, sequencer,
	)
	r.verifierCL[chainID] = node
	return node
}

func (r *StagedTwoL2ExternalCLInteropRuntime) requireL1() {
	r.t.Require().NotNil(r.l1EL, "start the staged L1 first")
	r.t.Require().NotNil(r.l1CL, "start the staged L1 beacon first")
}

func (r *StagedTwoL2ExternalCLInteropRuntime) L1EL() L1ELNode  { return r.l1EL }
func (r *StagedTwoL2ExternalCLInteropRuntime) L1CL() *L1CLNode { return r.l1CL }

func (r *StagedTwoL2ExternalCLInteropRuntime) L2Network(chainID string) *L2Network {
	return r.network(chainID)
}

func (r *StagedTwoL2ExternalCLInteropRuntime) SequencerEL(chainID string) L2ELNode {
	return r.sequencerEL[chainID]
}

func (r *StagedTwoL2ExternalCLInteropRuntime) VerifierEL(chainID string) L2ELNode {
	return r.verifierEL[chainID]
}

func (r *StagedTwoL2ExternalCLInteropRuntime) SequencerCL(chainID string) L2CLNode {
	return r.sequencerCL[chainID]
}

func (r *StagedTwoL2ExternalCLInteropRuntime) VerifierCL(chainID string) L2CLNode {
	return r.verifierCL[chainID]
}

func (r *StagedTwoL2ExternalCLInteropRuntime) Keys() devkeys.Keys { return r.keys }

func (r *StagedTwoL2ExternalCLInteropRuntime) DependencySet() depset.DependencySet {
	return r.dependency
}

func (r *StagedTwoL2ExternalCLInteropRuntime) ChainIDs() []eth.ChainID {
	return []eth.ChainID{r.network("901").ChainID(), r.network("902").ChainID()}
}

func (r *StagedTwoL2ExternalCLInteropRuntime) String() string {
	return fmt.Sprintf("staged-two-l2-interop(%s,%s)", r.network("901").ChainID(), r.network("902").ChainID())
}
