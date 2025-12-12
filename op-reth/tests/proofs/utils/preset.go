package utils

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2ELClient string

const (
	L2ELClientGeth L2ELClient = "geth"
	L2ELClientReth L2ELClient = "reth"
)

type L2ELNodeID struct {
	stack.L2ELNodeID
	Client L2ELClient
}

type L2ELNode struct {
	*dsl.L2ELNode
	Client L2ELClient
}

type MixedOpProofPreset struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2Chain   *dsl.L2Network
	L2Batcher *dsl.L2Batcher

	L2ELSequencer *L2ELNode
	L2CLSequencer *dsl.L2CLNode

	L2ELValidator *L2ELNode
	L2CLValidator *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FaucetL1 *dsl.Faucet
	FaucetL2 *dsl.Faucet
	FunderL1 *dsl.Funder
	FunderL2 *dsl.Funder

	TestSequencer *dsl.TestSequencer
}

func (m *MixedOpProofPreset) L2Network() *dsl.L2Network {
	return m.L2Chain
}

func (m *MixedOpProofPreset) L2ELSequencerNode() *dsl.L2ELNode {
	return m.L2ELSequencer.L2ELNode
}

func (m *MixedOpProofPreset) L2ELValidatorNode() *dsl.L2ELNode {
	return m.L2ELValidator.L2ELNode
}

// GethL2ELNode returns first L2 EL nodes that are running op-geth
func (m *MixedOpProofPreset) GethL2ELNode() *dsl.L2ELNode {
	if m.L2ELSequencer.Client == L2ELClientGeth {
		return m.L2ELSequencer.L2ELNode
	}

	if m.L2ELValidator.Client == L2ELClientGeth {
		return m.L2ELValidator.L2ELNode
	}

	return nil
}

// RethL2ELNode returns first L2 EL nodes that are running op-reth
func (m *MixedOpProofPreset) RethL2ELNode() *dsl.L2ELNode {
	if m.L2ELSequencer.Client == L2ELClientReth {
		return m.L2ELSequencer.L2ELNode
	}

	if m.L2ELValidator.Client == L2ELClientReth {
		return m.L2ELValidator.L2ELNode
	}
	return nil
}

func WithMixedOpProofPreset() stack.CommonOption {
	return stack.MakeCommon(DefaultMixedOpProofSystem(&DefaultMixedOpProofSystemIDs{}))
}

func L2NodeMatcher[
	I interface {
		comparable
		Key() string
	}, E stack.Identifiable[I]](value ...string) stack.Matcher[I, E] {
	return match.MatchElemFn[I, E](func(elem E) bool {
		for _, v := range value {
			if !strings.Contains(elem.ID().Key(), v) {
				return false
			}
		}
		return true
	})
}

func NewMixedOpProofPreset(t devtest.T) *MixedOpProofPreset {
	system := shim.NewSystem(t)
	orch := presets.Orchestrator()
	orch.Hydrate(system)

	t.Gate().Equal(len(system.L2Networks()), 1, "expected exactly one L2 network")
	t.Gate().Equal(len(system.L1Networks()), 1, "expected exactly one L1 network")

	l1Net := system.L1Network(match.FirstL1Network)
	l2Net := system.L2Network(match.Assume(t, match.L2ChainA))

	t.Gate().GreaterOrEqual(len(l2Net.L2CLNodes()), 2, "expected at least two L2CL nodes")

	sequencerCL := l2Net.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))
	sequencerELInner := l2Net.L2ELNode(match.Assume(t, match.EngineFor(sequencerCL)))
	var sequencerEL *L2ELNode
	if strings.Contains(sequencerELInner.ID().String(), "op-reth") {
		sequencerEL = &L2ELNode{
			L2ELNode: dsl.NewL2ELNode(sequencerELInner, orch.ControlPlane()),
			Client:   L2ELClientReth,
		}
	} else if strings.Contains(sequencerELInner.ID().String(), "op-geth") {
		sequencerEL = &L2ELNode{
			L2ELNode: dsl.NewL2ELNode(sequencerELInner, orch.ControlPlane()),
			Client:   L2ELClientGeth,
		}
	} else {
		t.Error("unexpected L2EL client for sequencer")
		t.FailNow()
	}

	verifierCL := l2Net.L2CLNode(match.Assume(t,
		match.And(
			match.Not(match.WithSequencerActive(t.Ctx())),
			match.Not(sequencerCL.ID()),
		)))
	verifierELInner := l2Net.L2ELNode(match.Assume(t,
		match.And(
			match.EngineFor(verifierCL),
			match.Not(sequencerEL.ID()),
		)))
	var verifierEL *L2ELNode
	if strings.Contains(verifierELInner.ID().String(), "op-reth") {
		verifierEL = &L2ELNode{
			L2ELNode: dsl.NewL2ELNode(verifierELInner, orch.ControlPlane()),
			Client:   L2ELClientReth,
		}
	} else if strings.Contains(verifierELInner.ID().String(), "op-geth") {
		verifierEL = &L2ELNode{
			L2ELNode: dsl.NewL2ELNode(verifierELInner, orch.ControlPlane()),
			Client:   L2ELClientGeth,
		}
	} else {
		t.Error("unexpected L2EL client for verifier")
		t.FailNow()
	}

	out := &MixedOpProofPreset{
		Log:           t.Logger(),
		T:             t,
		ControlPlane:  orch.ControlPlane(),
		L1Network:     dsl.NewL1Network(l1Net),
		L1EL:          dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
		L2Chain:       dsl.NewL2Network(l2Net, orch.ControlPlane()),
		L2Batcher:     dsl.NewL2Batcher(l2Net.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		L2ELSequencer: sequencerEL,
		L2CLSequencer: dsl.NewL2CLNode(sequencerCL, orch.ControlPlane()),
		L2ELValidator: verifierEL,
		L2CLValidator: dsl.NewL2CLNode(verifierCL, orch.ControlPlane()),
		Wallet:        dsl.NewRandomHDWallet(t, 30), // Random for test isolation
		FaucetL2:      dsl.NewFaucet(l2Net.Faucet(match.Assume(t, match.FirstFaucet))),

		TestSequencer: dsl.NewTestSequencer(system.TestSequencer(match.Assume(t, match.FirstTestSequencer))),
	}
	out.FaucetL1 = dsl.NewFaucet(out.L1Network.Escape().Faucet(match.Assume(t, match.FirstFaucet)))
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	out.FunderL2 = dsl.NewFunder(out.Wallet, out.FaucetL2, out.L2ELSequencer)
	return out
}

type DefaultMixedOpProofSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	L2 stack.L2NetworkID

	L2CLSequencer stack.L2CLNodeID
	L2ELSequencer L2ELNodeID

	L2CLValidator stack.L2CLNodeID
	L2ELValidator L2ELNodeID

	L2Batcher    stack.L2BatcherID
	L2Proposer   stack.L2ProposerID
	L2Challenger stack.L2ChallengerID

	TestSequencer stack.TestSequencerID
}

func NewDefaultMixedOpProofSystemIDs(l1ID, l2ID eth.ChainID) DefaultMixedOpProofSystemIDs {
	ids := DefaultMixedOpProofSystemIDs{
		L1:            stack.L1NetworkID(l1ID),
		L1EL:          stack.NewL1ELNodeID("l1", l1ID),
		L1CL:          stack.NewL1CLNodeID("l1", l1ID),
		L2:            stack.L2NetworkID(l2ID),
		L2CLSequencer: stack.NewL2CLNodeID("sequencer", l2ID),
		L2CLValidator: stack.NewL2CLNodeID("validator", l2ID),
		L2Batcher:     stack.NewL2BatcherID("main", l2ID),
		L2Proposer:    stack.NewL2ProposerID("main", l2ID),
		L2Challenger:  stack.NewL2ChallengerID("main", l2ID),
		TestSequencer: "test-sequencer",
	}

	// default to op-geth for sequencer and op-reth for validator
	if os.Getenv("OP_DEVSTACK_PROOF_SEQUENCER_EL") == "op-reth" {
		ids.L2ELSequencer = L2ELNodeID{
			L2ELNodeID: stack.NewL2ELNodeID("sequencer-op-reth", l2ID),
			Client:     L2ELClientReth,
		}
	} else {
		ids.L2ELSequencer = L2ELNodeID{
			L2ELNodeID: stack.NewL2ELNodeID("sequencer-op-geth", l2ID),
			Client:     L2ELClientGeth,
		}
	}

	if os.Getenv("OP_DEVSTACK_PROOF_VALIDATOR_EL") == "op-geth" {
		ids.L2ELValidator = L2ELNodeID{
			L2ELNodeID: stack.NewL2ELNodeID("validator-op-geth", l2ID),
			Client:     L2ELClientGeth,
		}
	} else {
		ids.L2ELValidator = L2ELNodeID{
			L2ELNodeID: stack.NewL2ELNodeID("validator-op-reth", l2ID),
			Client:     L2ELClientReth,
		}
	}

	return ids
}

func DefaultMixedOpProofSystem(dest *DefaultMixedOpProofSystemIDs) stack.Option[*sysgo.Orchestrator] {
	ids := NewDefaultMixedOpProofSystemIDs(sysgo.DefaultL1ID, sysgo.DefaultL2AID)
	return defaultMixedOpProofSystemOpts(&ids, dest)
}

func defaultMixedOpProofSystemOpts(src, dest *DefaultMixedOpProofSystemIDs) stack.CombinedOption[*sysgo.Orchestrator] {
	opt := stack.Combine[*sysgo.Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *sysgo.Orchestrator) {
		o.P().Logger().Info("Setting up")
	}))

	opt.Add(sysgo.WithMnemonicKeys(devkeys.TestMnemonic))

	// Get artifacts path
	artifactsPath := os.Getenv("OP_DEPLOYER_ARTIFACTS")
	if artifactsPath == "" {
		panic("OP_DEPLOYER_ARTIFACTS is not set")
	}

	opt.Add(sysgo.WithDeployer(),
		sysgo.WithDeployerPipelineOption(
			sysgo.WithDeployerCacheDir(artifactsPath),
		),
		sysgo.WithDeployerOptions(
			func(_ devtest.P, _ devkeys.Keys, builder intentbuilder.Builder) {
				builder.WithL1ContractsLocator(artifacts.MustNewFileLocator(artifactsPath))
				builder.WithL2ContractsLocator(artifacts.MustNewFileLocator(artifactsPath))
			},
			sysgo.WithCommons(src.L1.ChainID()),
			sysgo.WithPrefundedL2(src.L1.ChainID(), src.L2.ChainID()),
		),
	)

	opt.Add(sysgo.WithL1Nodes(src.L1EL, src.L1CL))

	// Spawn L2 sequencer nodes
	if src.L2ELSequencer.Client == L2ELClientReth {
		opt.Add(sysgo.WithOpReth(src.L2ELSequencer.L2ELNodeID))
	} else {
		opt.Add(sysgo.WithOpGeth(src.L2ELSequencer.L2ELNodeID))
	}
	opt.Add(sysgo.WithL2CLNode(src.L2CLSequencer, src.L1CL, src.L1EL, src.L2ELSequencer.L2ELNodeID, sysgo.L2CLSequencer()))

	// Spawn L2 validator nodes
	if src.L2ELValidator.Client == L2ELClientReth {
		opt.Add(sysgo.WithOpReth(src.L2ELValidator.L2ELNodeID))
	} else {
		opt.Add(sysgo.WithOpGeth(src.L2ELValidator.L2ELNodeID))
	}
	opt.Add(sysgo.WithL2CLNode(src.L2CLValidator, src.L1CL, src.L1EL, src.L2ELValidator.L2ELNodeID))

	opt.Add(sysgo.WithBatcher(src.L2Batcher, src.L1EL, src.L2CLSequencer, src.L2ELSequencer.L2ELNodeID))
	opt.Add(sysgo.WithProposer(src.L2Proposer, src.L1EL, &src.L2CLSequencer, nil))

	opt.Add(sysgo.WithFaucets([]stack.L1ELNodeID{src.L1EL}, []stack.L2ELNodeID{src.L2ELSequencer.L2ELNodeID}))

	opt.Add(sysgo.WithTestSequencer(src.TestSequencer, src.L1CL, src.L2CLSequencer, src.L1EL, src.L2ELSequencer.L2ELNodeID))

	opt.Add(stack.Finally(func(orch *sysgo.Orchestrator) {
		*dest = *src
	}))

	return opt
}
