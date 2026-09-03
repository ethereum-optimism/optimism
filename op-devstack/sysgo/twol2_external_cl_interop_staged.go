package sysgo

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// StagedTwoL2ExternalCLInteropRuntime prepares an interop world without
// starting or binding L1, EL, CL, batcher, supervisor, or test-sequencer
// services. Interactive callers explicitly start each required service.
type StagedTwoL2ExternalCLInteropRuntime struct {
	t   devtest.T
	cfg PresetConfig

	keys       devkeys.Keys
	dependency depset.DependencySet
	l1Net      *L1Network
	chainIDs   []eth.ChainID
	l2Nets     map[eth.ChainID]*L2Network
	jwtPath    string
	jwtSecret  [32]byte

	l1EL L1ELNode
	l1CL *L1CLNode

	sequencerEL map[eth.ChainID]L2ELNode
	verifierEL  map[eth.ChainID]L2ELNode
	sequencerCL map[eth.ChainID]L2CLNode
	verifierCL  map[eth.ChainID]L2CLNode
	timeTravel  *clock.AdvancingClock
}

// NewStagedTwoL2ExternalCLInteropRuntime prepares the same immutable network
// configuration as NewTwoL2ExternalCLInteropRuntimeWithConfig without starting
// network services.
func NewStagedTwoL2ExternalCLInteropRuntime(t devtest.T, delaySeconds uint64, cfg PresetConfig) *StagedTwoL2ExternalCLInteropRuntime {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err, "failed to derive dev keys from mnemonic")
	wb, l1Net, l2ANet, l2BNet := buildTwoL2RuntimeWorld(
		t, keys, true, delaySeconds, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...,
	)
	dependency := wb.outFullCfgSet.DependencySet
	t.Require().NotNil(dependency, "two-L2 interop world must provide a dependency set")
	jwtPath, jwtSecret := writeJWTSecret(t)
	var timeTravel *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravel = clock.NewAdvancingClock()
	}
	return &StagedTwoL2ExternalCLInteropRuntime{
		t: t, cfg: cfg, keys: keys, dependency: dependency,
		l1Net:       l1Net,
		chainIDs:    []eth.ChainID{l2ANet.ChainID(), l2BNet.ChainID()},
		l2Nets:      map[eth.ChainID]*L2Network{l2ANet.ChainID(): l2ANet, l2BNet.ChainID(): l2BNet},
		jwtPath:     jwtPath,
		jwtSecret:   jwtSecret,
		sequencerEL: make(map[eth.ChainID]L2ELNode), verifierEL: make(map[eth.ChainID]L2ELNode),
		sequencerCL: make(map[eth.ChainID]L2CLNode), verifierCL: make(map[eth.ChainID]L2CLNode),
		timeTravel: timeTravel,
	}
}

func (r *StagedTwoL2ExternalCLInteropRuntime) network(chainID eth.ChainID) *L2Network {
	network := r.l2Nets[chainID]
	r.t.Require().NotNil(network, "unknown staged L2 chain %s", chainID)
	return network
}

func (r *StagedTwoL2ExternalCLInteropRuntime) checkStartL1() error {
	if r.l1EL != nil || r.l1CL != nil {
		return fmt.Errorf("staged L1 is already running")
	}
	return nil
}

func (r *StagedTwoL2ExternalCLInteropRuntime) checkL1Ready() error {
	if r.l1EL == nil || r.l1CL == nil {
		return fmt.Errorf("start the staged L1 first")
	}
	return nil
}

func (r *StagedTwoL2ExternalCLInteropRuntime) checkKnownChain(chainID eth.ChainID) error {
	if r.l2Nets[chainID] == nil {
		return fmt.Errorf("unknown staged L2 chain %s", chainID)
	}
	return nil
}

func (r *StagedTwoL2ExternalCLInteropRuntime) checkStartEL(chainID eth.ChainID, role string, nodes map[eth.ChainID]L2ELNode) error {
	if err := r.checkL1Ready(); err != nil {
		return err
	}
	if err := r.checkKnownChain(chainID); err != nil {
		return err
	}
	if nodes[chainID] != nil {
		return fmt.Errorf("%s EL for chain %s already exists", role, chainID)
	}
	return nil
}

func (r *StagedTwoL2ExternalCLInteropRuntime) checkStartSequencerCL(chainID eth.ChainID) error {
	if err := r.checkL1Ready(); err != nil {
		return err
	}
	if err := r.checkKnownChain(chainID); err != nil {
		return err
	}
	if r.sequencerEL[chainID] == nil {
		return fmt.Errorf("start chain %s sequencer EL before its CL", chainID)
	}
	if r.sequencerCL[chainID] != nil {
		return fmt.Errorf("sequencer CL for chain %s already exists", chainID)
	}
	return nil
}

func (r *StagedTwoL2ExternalCLInteropRuntime) checkStartVerifierCL(chainID eth.ChainID) error {
	if err := r.checkL1Ready(); err != nil {
		return err
	}
	if err := r.checkKnownChain(chainID); err != nil {
		return err
	}
	if r.verifierEL[chainID] == nil {
		return fmt.Errorf("start chain %s verifier EL before its CL", chainID)
	}
	if r.sequencerEL[chainID] == nil {
		return fmt.Errorf("start chain %s sequencer EL before its verifier CL", chainID)
	}
	if r.sequencerCL[chainID] == nil {
		return fmt.Errorf("start chain %s sequencer CL before its verifier CL", chainID)
	}
	if r.verifierCL[chainID] != nil {
		return fmt.Errorf("verifier CL for chain %s already exists", chainID)
	}
	return nil
}

// StartL1 starts the shared execution and beacon services exactly once.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartL1() (L1ELNode, *L1CLNode) {
	r.t.Require().NoError(r.checkStartL1())
	l1Clock := clock.SystemClock
	if r.timeTravel != nil {
		l1Clock = r.timeTravel
	}
	r.l1EL, r.l1CL = startInProcessL1WithClockConfig(r.t, r.l1Net, r.jwtPath, l1Clock, r.cfg)
	return r.l1EL, r.l1CL
}

func (r *StagedTwoL2ExternalCLInteropRuntime) elOptions(opts []OpRethOption) []OpRethOption {
	base := append(append([]OpRethOption{}, ResolveMixedL2ELOpts(r.t)...), r.cfg.OpRethOptions...)
	return append(base, opts...)
}

// StartSequencerEL starts one chain's sequencer execution engine.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartSequencerEL(chainID eth.ChainID, opts ...OpRethOption) L2ELNode {
	r.t.Require().NoError(r.checkStartEL(chainID, "sequencer", r.sequencerEL))
	node := startSequencerEL(
		r.t, r.network(chainID), r.jwtPath, r.jwtSecret, NewELNodeIdentity(0), r.elOptions(opts)...,
	)
	r.sequencerEL[chainID] = node
	return node
}

// StartVerifierEL starts one chain's verifier execution engine.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartVerifierEL(chainID eth.ChainID, opts ...OpRethOption) L2ELNode {
	r.t.Require().NoError(r.checkStartEL(chainID, "verifier", r.verifierEL))
	node := startL2ELForKey(
		r.t, r.network(chainID), r.jwtPath, r.jwtSecret, twoL2VerifierNodeKey,
		NewELNodeIdentity(0), r.elOptions(opts)...,
	)
	r.verifierEL[chainID] = node
	return node
}

// StartSequencerCL attaches a factory-selected sequencer CL to one started EL.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartSequencerCL(chainID eth.ChainID, factory L2CLFactory) L2CLNode {
	r.t.Require().NoError(r.checkStartSequencerCL(chainID))
	el := r.sequencerEL[chainID]
	result := startL2CLForKeyWithKind(
		r.t, r.keys, r.l1Net, r.network(chainID), r.l1EL, r.l1CL, el, r.jwtSecret,
		"sequencer", "sequencer", true, "", r.dependency, r.cfg.GlobalL2CLOptions,
		factory, nil, false, devstackL2CLKind(),
	)
	node := result.Node
	r.sequencerCL[chainID] = node
	return node
}

// StartVerifierCL attaches a factory-selected verifier CL and identifies the
// matching sequencer EL as its unsafe source.
func (r *StagedTwoL2ExternalCLInteropRuntime) StartVerifierCL(chainID eth.ChainID, factory L2CLFactory) L2CLNode {
	r.t.Require().NoError(r.checkStartVerifierCL(chainID))
	verifier := r.verifierEL[chainID]
	sequencer := r.sequencerEL[chainID]
	sequencerCL := r.sequencerCL[chainID]
	connectL2ELPeers(r.t, r.t.Logger(), sequencer.UserRPC(), verifier.UserRPC())
	stockFollowSource := interopVerifierFollowSource(sequencerCL)
	node := startL2CLForKeyWithKind(
		r.t, r.keys, r.l1Net, r.network(chainID), r.l1EL, r.l1CL, verifier, r.jwtSecret,
		twoL2VerifierNodeKey, twoL2VerifierNodeKey, false, stockFollowSource, r.dependency,
		r.cfg.GlobalL2CLOptions, factory, sequencer, false, MixedL2CLOpNode,
	).Node
	r.verifierCL[chainID] = node
	return node
}

func (r *StagedTwoL2ExternalCLInteropRuntime) L1EL() L1ELNode  { return r.l1EL }
func (r *StagedTwoL2ExternalCLInteropRuntime) L1CL() *L1CLNode { return r.l1CL }

func (r *StagedTwoL2ExternalCLInteropRuntime) L2Network(chainID eth.ChainID) *L2Network {
	return r.network(chainID)
}

func (r *StagedTwoL2ExternalCLInteropRuntime) SequencerEL(chainID eth.ChainID) L2ELNode {
	return r.sequencerEL[chainID]
}

func (r *StagedTwoL2ExternalCLInteropRuntime) VerifierEL(chainID eth.ChainID) L2ELNode {
	return r.verifierEL[chainID]
}

func (r *StagedTwoL2ExternalCLInteropRuntime) SequencerCL(chainID eth.ChainID) L2CLNode {
	return r.sequencerCL[chainID]
}

func (r *StagedTwoL2ExternalCLInteropRuntime) VerifierCL(chainID eth.ChainID) L2CLNode {
	return r.verifierCL[chainID]
}

func (r *StagedTwoL2ExternalCLInteropRuntime) Keys() devkeys.Keys { return r.keys }

func (r *StagedTwoL2ExternalCLInteropRuntime) DependencySet() depset.DependencySet {
	return r.dependency
}

// TimeTravel returns the advancing L1 clock configured at construction, or
// nil when the staged world uses wall-clock time.
func (r *StagedTwoL2ExternalCLInteropRuntime) TimeTravel() *clock.AdvancingClock {
	return r.timeTravel
}

func (r *StagedTwoL2ExternalCLInteropRuntime) ChainIDs() []eth.ChainID {
	return append([]eth.ChainID(nil), r.chainIDs...)
}

func (r *StagedTwoL2ExternalCLInteropRuntime) String() string {
	return fmt.Sprintf("staged-two-l2-interop(%s,%s)", r.chainIDs[0], r.chainIDs[1])
}
