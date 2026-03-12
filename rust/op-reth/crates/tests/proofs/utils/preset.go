package utils

import (
	"os"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	devpresets "github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
)

type L2ELClient string

const (
	L2ELClientGeth           L2ELClient = "geth"
	L2ELClientReth           L2ELClient = "reth"
	L2ELClientRethWithProofs L2ELClient = "reth-with-proof"
)

type L2ELNode struct {
	*dsl.L2ELNode
	Client L2ELClient
}

type MixedOpProofPreset struct {
	Log log.Logger
	T   devtest.T

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode
	L1CL      *dsl.L1CLNode

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

func (m *MixedOpProofPreset) GethL2ELNode() *dsl.L2ELNode {
	if m.L2ELSequencer.Client == L2ELClientGeth {
		return m.L2ELSequencer.L2ELNode
	}
	if m.L2ELValidator.Client == L2ELClientGeth {
		return m.L2ELValidator.L2ELNode
	}
	return nil
}

func (m *MixedOpProofPreset) RethL2ELNode() *dsl.L2ELNode {
	if m.L2ELSequencer.Client == L2ELClientReth {
		return m.L2ELSequencer.L2ELNode
	}
	if m.L2ELValidator.Client == L2ELClientReth {
		return m.L2ELValidator.L2ELNode
	}
	return nil
}

func (m *MixedOpProofPreset) RethWithProofL2ELNode() *dsl.L2ELNode {
	if m.L2ELSequencer.Client == L2ELClientRethWithProofs {
		return m.L2ELSequencer.L2ELNode
	}
	if m.L2ELValidator.Client == L2ELClientRethWithProofs {
		return m.L2ELValidator.L2ELNode
	}
	return nil
}

func NewMixedOpProofPreset(t devtest.T) *MixedOpProofPreset {
	nodeSpecs := mixedOpProofNodeSpecs()
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs:         nodeSpecs,
		WithTestSequencer: true,
		TestSequencerName: "test-sequencer",
		DeployerOptions:   proofDeployerOptions(t),
	})

	frontends := devpresets.NewMixedSingleChainFrontends(t, runtime)
	return mixedOpProofFromFrontends(t, frontends)
}

func mixedOpProofFromFrontends(t devtest.T, frontends *devpresets.MixedSingleChainFrontends) *MixedOpProofPreset {
	t.Require().NotNil(frontends.TestSequencer, "expected test sequencer frontend")

	var l2ELSequencer *L2ELNode
	var l2CLSequencer *dsl.L2CLNode
	var l2ELValidator *L2ELNode
	var l2CLValidator *dsl.L2CLNode
	for _, node := range frontends.Nodes {
		client := mixedOpProofClientFromSpec(node.Spec)
		if node.Spec.IsSequencer {
			l2ELSequencer = &L2ELNode{L2ELNode: node.EL, Client: client}
			l2CLSequencer = node.CL
			continue
		}
		l2ELValidator = &L2ELNode{L2ELNode: node.EL, Client: client}
		l2CLValidator = node.CL
	}
	t.Require().NotNil(l2ELSequencer, "missing sequencer EL frontend")
	t.Require().NotNil(l2CLSequencer, "missing sequencer CL frontend")
	t.Require().NotNil(l2ELValidator, "missing validator EL frontend")
	t.Require().NotNil(l2CLValidator, "missing validator CL frontend")

	out := &MixedOpProofPreset{
		Log:           t.Logger(),
		T:             t,
		L1Network:     frontends.L1Network,
		L1EL:          frontends.L1EL,
		L1CL:          frontends.L1CL,
		L2Chain:       frontends.L2Network,
		L2Batcher:     frontends.L2Batcher,
		L2ELSequencer: l2ELSequencer,
		L2CLSequencer: l2CLSequencer,
		L2ELValidator: l2ELValidator,
		L2CLValidator: l2CLValidator,
		Wallet:        dsl.NewRandomHDWallet(t, 30),
		FaucetL1:      frontends.FaucetL1,
		FaucetL2:      frontends.FaucetL2,
		TestSequencer: frontends.TestSequencer,
	}
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	out.FunderL2 = dsl.NewFunder(out.Wallet, out.FaucetL2, out.L2ELSequencer)
	return out
}

func proofDeployerOptions(t devtest.T) []sysgo.DeployerOption {
	artifactsPath := os.Getenv("OP_DEPLOYER_ARTIFACTS")
	t.Require().NotEmpty(artifactsPath, "OP_DEPLOYER_ARTIFACTS is not set")
	return []sysgo.DeployerOption{
		func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
			locator := artifacts.MustNewFileLocator(artifactsPath)
			builder.WithL1ContractsLocator(locator)
			builder.WithL2ContractsLocator(locator)
		},
	}
}

func mixedOpProofNodeSpecs() []sysgo.MixedSingleChainNodeSpec {
	sequencerClient := mixedOpProofClientFromEnv("OP_DEVSTACK_PROOF_SEQUENCER_EL", L2ELClientGeth)
	validatorClient := mixedOpProofClientFromEnv("OP_DEVSTACK_PROOF_VALIDATOR_EL", L2ELClientRethWithProofs)
	return []sysgo.MixedSingleChainNodeSpec{
		{
			ELKey:          mixedOpProofELKey("sequencer", sequencerClient),
			CLKey:          "sequencer",
			ELKind:         mixedOpProofELKind(sequencerClient),
			ELProofHistory: sequencerClient == L2ELClientRethWithProofs,
			CLKind:         sysgo.MixedL2CLOpNode,
			IsSequencer:    true,
		},
		{
			ELKey:          mixedOpProofELKey("validator", validatorClient),
			CLKey:          "validator",
			ELKind:         mixedOpProofELKind(validatorClient),
			ELProofHistory: validatorClient == L2ELClientRethWithProofs,
			CLKind:         sysgo.MixedL2CLOpNode,
			IsSequencer:    false,
		},
	}
}

func mixedOpProofELKey(role string, client L2ELClient) string {
	switch client {
	case L2ELClientGeth:
		return role + "-op-geth"
	case L2ELClientReth:
		return role + "-op-reth"
	case L2ELClientRethWithProofs:
		return role + "-op-reth-with-proof"
	default:
		panic("unknown mixed proof L2 EL client")
	}
}

func mixedOpProofClientFromEnv(name string, fallback L2ELClient) L2ELClient {
	switch os.Getenv(name) {
	case "op-geth":
		return L2ELClientGeth
	case "op-reth":
		return L2ELClientReth
	case "op-reth-with-proof":
		return L2ELClientRethWithProofs
	default:
		return fallback
	}
}

func mixedOpProofClientFromSpec(spec sysgo.MixedSingleChainNodeSpec) L2ELClient {
	if spec.ELKind == sysgo.MixedL2ELOpGeth {
		return L2ELClientGeth
	}
	if spec.ELProofHistory {
		return L2ELClientRethWithProofs
	}
	return L2ELClientReth
}

func mixedOpProofELKind(client L2ELClient) sysgo.MixedL2ELKind {
	switch client {
	case L2ELClientGeth:
		return sysgo.MixedL2ELOpGeth
	case L2ELClientReth, L2ELClientRethWithProofs:
		return sysgo.MixedL2ELOpReth
	default:
		panic("unknown mixed proof L2 EL client")
	}
}
