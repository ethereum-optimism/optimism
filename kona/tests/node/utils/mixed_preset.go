package node_utils

import (
	"fmt"
	"os"
	"strconv"
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
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2NodeKind string

const (
	OpNode    L2NodeKind = "op"
	KonaNode  L2NodeKind = "kona"
	Sequencer L2NodeKind = "sequencer"
	Validator L2NodeKind = "validator"
)

type L2NodeConfig struct {
	OpSequencerNodesWithGeth   int
	OpSequencerNodesWithReth   int
	KonaSequencerNodesWithGeth int
	KonaSequencerNodesWithReth int
	OpNodesWithGeth            int
	OpNodesWithReth            int
	KonaNodesWithGeth          int
	KonaNodesWithReth          int
}

const (
	DefaultOpSequencerGeth = 0
	DefaultOpSequencerReth = 0

	DefaultKonaSequencerGeth = 0
	DefaultKonaSequencerReth = 1

	DefaultOpValidatorGeth = 0
	DefaultOpValidatorReth = 0

	DefaultKonaValidatorGeth = 3
	DefaultKonaValidatorReth = 3
)

func ParseL2NodeConfigFromEnv() L2NodeConfig {
	// Get environment variable: OP_SEQUENCER_NODES. Convert to int.
	opSequencerGeth := os.Getenv("OP_SEQUENCER_WITH_GETH")
	opSequencerGethInt, err := strconv.Atoi(opSequencerGeth)
	if err != nil {
		opSequencerGethInt = DefaultOpSequencerGeth
	}
	// Get environment variable: KONA_SEQUENCER_NODES
	konaSequencerGeth := os.Getenv("KONA_SEQUENCER_WITH_GETH")
	konaSequencerGethInt, err := strconv.Atoi(konaSequencerGeth)
	if err != nil {
		konaSequencerGethInt = DefaultKonaSequencerGeth
	}
	// Get environment variable: OP_SEQUENCER_WITH_RETH
	opSequencerReth := os.Getenv("OP_SEQUENCER_WITH_RETH")
	opSequencerRethInt, err := strconv.Atoi(opSequencerReth)
	if err != nil {
		opSequencerRethInt = DefaultOpSequencerReth
	}
	// Get environment variable: KONA_SEQUENCER_WITH_RETH
	konaSequencerReth := os.Getenv("KONA_SEQUENCER_WITH_RETH")
	konaSequencerRethInt, err := strconv.Atoi(konaSequencerReth)
	if err != nil {
		konaSequencerRethInt = DefaultKonaSequencerReth
	}
	// Get environment variable: OP_VALIDATOR_WITH_GETH
	opValidatorGeth := os.Getenv("OP_VALIDATOR_WITH_GETH")
	opValidatorGethInt, err := strconv.Atoi(opValidatorGeth)
	if err != nil {
		opValidatorGethInt = DefaultOpValidatorGeth
	}
	// Get environment variable: OP_VALIDATOR_WITH_RETH
	opValidatorReth := os.Getenv("OP_VALIDATOR_WITH_RETH")
	opValidatorRethInt, err := strconv.Atoi(opValidatorReth)
	if err != nil {
		opValidatorRethInt = DefaultOpValidatorReth
	}
	// Get environment variable: KONA_VALIDATOR_WITH_GETH
	konaValidatorGeth := os.Getenv("KONA_VALIDATOR_WITH_GETH")
	konaValidatorGethInt, err := strconv.Atoi(konaValidatorGeth)
	if err != nil {
		konaValidatorGethInt = DefaultKonaValidatorGeth
	}
	// Get environment variable: KONA_VALIDATOR_WITH_RETH
	konaValidatorReth := os.Getenv("KONA_VALIDATOR_WITH_RETH")
	konaValidatorRethInt, err := strconv.Atoi(konaValidatorReth)
	if err != nil {
		konaValidatorRethInt = DefaultKonaValidatorReth
	}

	return L2NodeConfig{
		OpSequencerNodesWithGeth: opSequencerGethInt,
		OpSequencerNodesWithReth: opSequencerRethInt,

		OpNodesWithGeth: opValidatorGethInt,
		OpNodesWithReth: opValidatorRethInt,

		KonaSequencerNodesWithGeth: konaSequencerGethInt,
		KonaSequencerNodesWithReth: konaSequencerRethInt,

		KonaNodesWithGeth: konaValidatorGethInt,
		KonaNodesWithReth: konaValidatorRethInt,
	}
}

func (l2NodeConfig L2NodeConfig) TotalNodes() int {
	return l2NodeConfig.OpSequencerNodesWithGeth + l2NodeConfig.OpSequencerNodesWithReth + l2NodeConfig.KonaSequencerNodesWithGeth + l2NodeConfig.KonaSequencerNodesWithReth + l2NodeConfig.OpNodesWithGeth + l2NodeConfig.OpNodesWithReth + l2NodeConfig.KonaNodesWithGeth + l2NodeConfig.KonaNodesWithReth
}

func (l2NodeConfig L2NodeConfig) OpSequencerNodes() int {
	return l2NodeConfig.OpSequencerNodesWithGeth + l2NodeConfig.OpSequencerNodesWithReth
}

func (l2NodeConfig L2NodeConfig) KonaSequencerNodes() int {
	return l2NodeConfig.KonaSequencerNodesWithGeth + l2NodeConfig.KonaSequencerNodesWithReth
}

func (l2NodeConfig L2NodeConfig) OpValidatorNodes() int {
	return l2NodeConfig.OpNodesWithGeth + l2NodeConfig.OpNodesWithReth
}

func (l2NodeConfig L2NodeConfig) KonaValidatorNodes() int {
	return l2NodeConfig.KonaNodesWithGeth + l2NodeConfig.KonaNodesWithReth
}

type MixedOpKonaPreset struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2Chain   *dsl.L2Network
	L2Batcher *dsl.L2Batcher

	L2ELKonaSequencerNodes []dsl.L2ELNode
	L2CLKonaSequencerNodes []dsl.L2CLNode

	L2ELOpSequencerNodes []dsl.L2ELNode
	L2CLOpSequencerNodes []dsl.L2CLNode

	L2ELOpValidatorNodes []dsl.L2ELNode
	L2CLOpValidatorNodes []dsl.L2CLNode

	L2ELKonaValidatorNodes []dsl.L2ELNode
	L2CLKonaValidatorNodes []dsl.L2CLNode

	Wallet *dsl.HDWallet

	FaucetL1 *dsl.Faucet
	Faucet   *dsl.Faucet
	FunderL1 *dsl.Funder
	Funder   *dsl.Funder
}

// L2ELNodes returns all the L2EL nodes in the network (op-reth, op-geth, etc.), validator and sequencer.
func (m *MixedOpKonaPreset) L2ELNodes() []dsl.L2ELNode {
	return append(m.L2ELSequencerNodes(), m.L2ELValidatorNodes()...)
}

// L2CLNodes returns all the L2CL nodes in the network (op-nodes and kona-nodes), validator and sequencer.
func (m *MixedOpKonaPreset) L2CLNodes() []dsl.L2CLNode {
	return append(m.L2CLSequencerNodes(), m.L2CLValidatorNodes()...)
}

// L2CLValidatorNodes returns all the validator L2CL nodes in the network (op-nodes and kona-nodes).
func (m *MixedOpKonaPreset) L2CLValidatorNodes() []dsl.L2CLNode {
	return append(m.L2CLOpValidatorNodes, m.L2CLKonaValidatorNodes...)
}

// L2CLSequencerNodes returns all the sequencer L2CL nodes in the network (op-nodes and kona-nodes).
func (m *MixedOpKonaPreset) L2CLSequencerNodes() []dsl.L2CLNode {
	return append(m.L2CLOpSequencerNodes, m.L2CLKonaSequencerNodes...)
}

// L2ELValidatorNodes returns all the validator L2EL nodes in the network (op-reth, op-geth, etc.).
func (m *MixedOpKonaPreset) L2ELValidatorNodes() []dsl.L2ELNode {
	return append(m.L2ELOpValidatorNodes, m.L2ELKonaValidatorNodes...)
}

// L2ELSequencerNodes returns all the sequencer L2EL nodes in the network (op-reth, op-geth, etc.).
func (m *MixedOpKonaPreset) L2ELSequencerNodes() []dsl.L2ELNode {
	return append(m.L2ELOpSequencerNodes, m.L2ELKonaSequencerNodes...)
}

func (m *MixedOpKonaPreset) L2CLKonaNodes() []dsl.L2CLNode {
	return append(m.L2CLKonaValidatorNodes, m.L2CLKonaSequencerNodes...)
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

func (m *MixedOpKonaPreset) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func WithMixedOpKona(l2NodeConfig L2NodeConfig) stack.CommonOption {
	return stack.MakeCommon(DefaultMixedOpKonaSystem(&DefaultMixedOpKonaSystemIDs{}, l2NodeConfig))
}

func L2CLNodes(nodes []stack.L2CLNode, orch stack.Orchestrator) []dsl.L2CLNode {
	out := make([]dsl.L2CLNode, len(nodes))
	for i, node := range nodes {
		out[i] = *dsl.NewL2CLNode(node, orch.ControlPlane())
	}
	return out
}

func L2ELNodes(nodes []stack.L2ELNode, orch stack.Orchestrator) []dsl.L2ELNode {
	out := make([]dsl.L2ELNode, len(nodes))
	for i, node := range nodes {
		out[i] = *dsl.NewL2ELNode(node, orch.ControlPlane())
	}
	return out
}

func NewMixedOpKona(t devtest.T) *MixedOpKonaPreset {
	system := shim.NewSystem(t)
	orch := presets.Orchestrator()
	orch.Hydrate(system)

	t.Gate().Equal(len(system.L2Networks()), 1, "expected exactly one L2 network")
	t.Gate().Equal(len(system.L1Networks()), 1, "expected exactly one L1 network")

	l1Net := system.L1Network(match.FirstL1Network)
	l2Net := system.L2Network(match.Assume(t, match.L2ChainA))

	t.Gate().GreaterOrEqual(len(l2Net.L2CLNodes()), 2, "expected at least two L2CL nodes")

	opSequencerCLNodes := L2NodeMatcher[stack.L2CLNodeID, stack.L2CLNode](string(OpNode), string(Sequencer)).Match(l2Net.L2CLNodes())
	konaSequencerCLNodes := L2NodeMatcher[stack.L2CLNodeID, stack.L2CLNode](string(KonaNode), string(Sequencer)).Match(l2Net.L2CLNodes())

	opCLNodes := L2NodeMatcher[stack.L2CLNodeID, stack.L2CLNode](string(OpNode), string(Validator)).Match(l2Net.L2CLNodes())
	konaCLNodes := L2NodeMatcher[stack.L2CLNodeID, stack.L2CLNode](string(KonaNode), string(Validator)).Match(l2Net.L2CLNodes())

	opSequencerELNodes := L2NodeMatcher[stack.L2ELNodeID, stack.L2ELNode](string(OpNode), string(Sequencer)).Match(l2Net.L2ELNodes())
	konaSequencerELNodes := L2NodeMatcher[stack.L2ELNodeID, stack.L2ELNode](string(KonaNode), string(Sequencer)).Match(l2Net.L2ELNodes())
	opELNodes := L2NodeMatcher[stack.L2ELNodeID, stack.L2ELNode](string(OpNode), string(Validator)).Match(l2Net.L2ELNodes())
	konaELNodes := L2NodeMatcher[stack.L2ELNodeID, stack.L2ELNode](string(KonaNode), string(Validator)).Match(l2Net.L2ELNodes())

	out := &MixedOpKonaPreset{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L1EL:         dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
		L2Chain:      dsl.NewL2Network(l2Net, orch.ControlPlane()),
		L2Batcher:    dsl.NewL2Batcher(l2Net.L2Batcher(match.Assume(t, match.FirstL2Batcher))),

		L2ELOpSequencerNodes: L2ELNodes(opSequencerELNodes, orch),
		L2CLOpSequencerNodes: L2CLNodes(opSequencerCLNodes, orch),

		L2ELOpValidatorNodes: L2ELNodes(opELNodes, orch),
		L2CLOpValidatorNodes: L2CLNodes(opCLNodes, orch),

		L2ELKonaSequencerNodes: L2ELNodes(konaSequencerELNodes, orch),
		L2CLKonaSequencerNodes: L2CLNodes(konaSequencerCLNodes, orch),

		L2ELKonaValidatorNodes: L2ELNodes(konaELNodes, orch),
		L2CLKonaValidatorNodes: L2CLNodes(konaCLNodes, orch),

		Wallet: dsl.NewHDWallet(t, devkeys.TestMnemonic, 30),
		Faucet: dsl.NewFaucet(l2Net.Faucet(match.Assume(t, match.FirstFaucet))),
	}
	return out
}

type DefaultMixedOpKonaSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	L2 stack.L2NetworkID

	L2ELOpGethSequencerNodes []stack.L2ELNodeID
	L2ELOpRethSequencerNodes []stack.L2ELNodeID

	L2CLOpGethSequencerNodes []stack.L2CLNodeID
	L2CLOpRethSequencerNodes []stack.L2CLNodeID

	L2ELKonaGethSequencerNodes []stack.L2ELNodeID
	L2ELKonaRethSequencerNodes []stack.L2ELNodeID

	L2CLKonaGethSequencerNodes []stack.L2CLNodeID
	L2CLKonaRethSequencerNodes []stack.L2CLNodeID

	L2CLOpGethNodes []stack.L2CLNodeID
	L2ELOpGethNodes []stack.L2ELNodeID

	L2CLOpRethNodes []stack.L2CLNodeID
	L2ELOpRethNodes []stack.L2ELNodeID

	L2CLKonaGethNodes []stack.L2CLNodeID
	L2ELKonaGethNodes []stack.L2ELNodeID

	L2CLKonaRethNodes []stack.L2CLNodeID
	L2ELKonaRethNodes []stack.L2ELNodeID

	L2Batcher  stack.L2BatcherID
	L2Proposer stack.L2ProposerID
}

func (ids *DefaultMixedOpKonaSystemIDs) L2CLSequencerNodes() []stack.L2CLNodeID {
	list := append(ids.L2CLOpGethSequencerNodes, ids.L2CLOpRethSequencerNodes...)
	list = append(list, ids.L2CLKonaGethSequencerNodes...)
	list = append(list, ids.L2CLKonaRethSequencerNodes...)
	return list
}

func (ids *DefaultMixedOpKonaSystemIDs) L2ELSequencerNodes() []stack.L2ELNodeID {
	list := append(ids.L2ELOpGethSequencerNodes, ids.L2ELOpRethSequencerNodes...)
	list = append(list, ids.L2ELKonaGethSequencerNodes...)
	list = append(list, ids.L2ELKonaRethSequencerNodes...)
	return list
}

func (ids *DefaultMixedOpKonaSystemIDs) L2CLValidatorNodes() []stack.L2CLNodeID {
	list := append(ids.L2CLOpGethNodes, ids.L2CLOpRethNodes...)
	list = append(list, ids.L2CLKonaGethNodes...)
	list = append(list, ids.L2CLKonaRethNodes...)
	return list
}
func (ids *DefaultMixedOpKonaSystemIDs) L2ELValidatorNodes() []stack.L2ELNodeID {
	list := append(ids.L2ELOpGethNodes, ids.L2ELOpRethNodes...)
	list = append(list, ids.L2ELKonaGethNodes...)
	list = append(list, ids.L2ELKonaRethNodes...)
	return list
}

func (ids *DefaultMixedOpKonaSystemIDs) L2CLNodes() []stack.L2CLNodeID {
	return append(ids.L2CLSequencerNodes(), ids.L2CLValidatorNodes()...)
}

func (ids *DefaultMixedOpKonaSystemIDs) L2ELNodes() []stack.L2ELNodeID {
	return append(ids.L2ELSequencerNodes(), ids.L2ELValidatorNodes()...)
}

func NewDefaultMixedOpKonaSystemIDs(l1ID, l2ID eth.ChainID, l2NodeConfig L2NodeConfig) DefaultMixedOpKonaSystemIDs {
	rethOpCLNodes := make([]stack.L2CLNodeID, l2NodeConfig.OpNodesWithReth)
	rethOpELNodes := make([]stack.L2ELNodeID, l2NodeConfig.OpNodesWithReth)
	rethKonaCLNodes := make([]stack.L2CLNodeID, l2NodeConfig.KonaNodesWithReth)
	rethKonaELNodes := make([]stack.L2ELNodeID, l2NodeConfig.KonaNodesWithReth)

	gethOpCLNodes := make([]stack.L2CLNodeID, l2NodeConfig.OpNodesWithGeth)
	gethOpELNodes := make([]stack.L2ELNodeID, l2NodeConfig.OpNodesWithGeth)
	gethKonaCLNodes := make([]stack.L2CLNodeID, l2NodeConfig.KonaNodesWithGeth)
	gethKonaELNodes := make([]stack.L2ELNodeID, l2NodeConfig.KonaNodesWithGeth)

	gethOpSequencerCLNodes := make([]stack.L2CLNodeID, l2NodeConfig.OpSequencerNodesWithGeth)
	gethOpSequencerELNodes := make([]stack.L2ELNodeID, l2NodeConfig.OpSequencerNodesWithGeth)
	gethKonaSequencerCLNodes := make([]stack.L2CLNodeID, l2NodeConfig.KonaSequencerNodesWithGeth)
	gethKonaSequencerELNodes := make([]stack.L2ELNodeID, l2NodeConfig.KonaSequencerNodesWithGeth)

	rethOpSequencerCLNodes := make([]stack.L2CLNodeID, l2NodeConfig.OpSequencerNodesWithReth)
	rethOpSequencerELNodes := make([]stack.L2ELNodeID, l2NodeConfig.OpSequencerNodesWithReth)
	rethKonaSequencerCLNodes := make([]stack.L2CLNodeID, l2NodeConfig.KonaSequencerNodesWithReth)
	rethKonaSequencerELNodes := make([]stack.L2ELNodeID, l2NodeConfig.KonaSequencerNodesWithReth)

	for i := range l2NodeConfig.OpSequencerNodesWithGeth {
		gethOpSequencerCLNodes[i] = stack.NewL2CLNodeID(fmt.Sprintf("cl-geth-op-sequencer-%d", i), l2ID)
		gethOpSequencerELNodes[i] = stack.NewL2ELNodeID(fmt.Sprintf("el-geth-op-sequencer-%d", i), l2ID)
	}

	for i := range l2NodeConfig.KonaSequencerNodesWithGeth {
		gethKonaSequencerCLNodes[i] = stack.NewL2CLNodeID(fmt.Sprintf("cl-geth-kona-sequencer-%d", i), l2ID)
		gethKonaSequencerELNodes[i] = stack.NewL2ELNodeID(fmt.Sprintf("el-geth-kona-sequencer-%d", i), l2ID)
	}

	for i := range l2NodeConfig.OpSequencerNodesWithReth {
		rethOpSequencerCLNodes[i] = stack.NewL2CLNodeID(fmt.Sprintf("cl-reth-op-sequencer-%d", i), l2ID)
		rethOpSequencerELNodes[i] = stack.NewL2ELNodeID(fmt.Sprintf("el-reth-op-sequencer-%d", i), l2ID)
	}

	for i := range l2NodeConfig.KonaSequencerNodesWithReth {
		rethKonaSequencerCLNodes[i] = stack.NewL2CLNodeID(fmt.Sprintf("cl-reth-kona-sequencer-%d", i), l2ID)
		rethKonaSequencerELNodes[i] = stack.NewL2ELNodeID(fmt.Sprintf("el-reth-kona-sequencer-%d", i), l2ID)
	}

	for i := range l2NodeConfig.OpNodesWithGeth {
		gethOpCLNodes[i] = stack.NewL2CLNodeID(fmt.Sprintf("cl-geth-op-validator-%d", i), l2ID)
		gethOpELNodes[i] = stack.NewL2ELNodeID(fmt.Sprintf("el-geth-op-validator-%d", i), l2ID)
	}

	for i := range l2NodeConfig.OpNodesWithReth {
		rethOpCLNodes[i] = stack.NewL2CLNodeID(fmt.Sprintf("cl-reth-op-validator-%d", i), l2ID)
		rethOpELNodes[i] = stack.NewL2ELNodeID(fmt.Sprintf("el-reth-op-validator-%d", i), l2ID)
	}

	for i := range l2NodeConfig.KonaNodesWithGeth {
		gethKonaCLNodes[i] = stack.NewL2CLNodeID(fmt.Sprintf("cl-geth-kona-validator-%d", i), l2ID)
		gethKonaELNodes[i] = stack.NewL2ELNodeID(fmt.Sprintf("el-geth-kona-validator-%d", i), l2ID)
	}

	for i := range l2NodeConfig.KonaNodesWithReth {
		rethKonaCLNodes[i] = stack.NewL2CLNodeID(fmt.Sprintf("cl-reth-kona-validator-%d", i), l2ID)
		rethKonaELNodes[i] = stack.NewL2ELNodeID(fmt.Sprintf("el-reth-kona-validator-%d", i), l2ID)
	}

	ids := DefaultMixedOpKonaSystemIDs{
		L1:   stack.L1NetworkID(l1ID),
		L1EL: stack.NewL1ELNodeID("l1", l1ID),
		L1CL: stack.NewL1CLNodeID("l1", l1ID),
		L2:   stack.L2NetworkID(l2ID),

		L2CLOpGethSequencerNodes: gethOpSequencerCLNodes,
		L2ELOpGethSequencerNodes: gethOpSequencerELNodes,

		L2CLOpRethSequencerNodes: rethOpSequencerCLNodes,
		L2ELOpRethSequencerNodes: rethOpSequencerELNodes,

		L2CLOpGethNodes: gethOpCLNodes,
		L2ELOpGethNodes: gethOpELNodes,

		L2CLOpRethNodes: rethOpCLNodes,
		L2ELOpRethNodes: rethOpELNodes,

		L2CLKonaGethSequencerNodes: gethKonaSequencerCLNodes,
		L2ELKonaGethSequencerNodes: gethKonaSequencerELNodes,

		L2CLKonaRethSequencerNodes: rethKonaSequencerCLNodes,
		L2ELKonaRethSequencerNodes: rethKonaSequencerELNodes,

		L2CLKonaGethNodes: gethKonaCLNodes,
		L2ELKonaGethNodes: gethKonaELNodes,

		L2CLKonaRethNodes: rethKonaCLNodes,
		L2ELKonaRethNodes: rethKonaELNodes,

		L2Batcher:  stack.NewL2BatcherID("main", l2ID),
		L2Proposer: stack.NewL2ProposerID("main", l2ID),
	}
	return ids
}

func DefaultMixedOpKonaSystem(dest *DefaultMixedOpKonaSystemIDs, l2NodeConfig L2NodeConfig) stack.CombinedOption[*sysgo.Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(DefaultL1ID)
	l2ID := eth.ChainIDFromUInt64(DefaultL2ID)
	ids := NewDefaultMixedOpKonaSystemIDs(l1ID, l2ID, l2NodeConfig)

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
			sysgo.WithCommons(ids.L1.ChainID()),
			sysgo.WithPrefundedL2(ids.L1.ChainID(), ids.L2.ChainID()),
		),
	)

	opt.Add(sysgo.WithL1Nodes(ids.L1EL, ids.L1CL))

	// Spawn all nodes.
	for i := range ids.L2CLKonaGethSequencerNodes {
		opt.Add(sysgo.WithOpGeth(ids.L2ELKonaGethSequencerNodes[i]))
		opt.Add(sysgo.WithKonaNode(ids.L2CLKonaGethSequencerNodes[i], ids.L1CL, ids.L1EL, ids.L2ELKonaGethSequencerNodes[i], sysgo.L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
			cfg.IsSequencer = true
			cfg.SequencerSyncMode = sync.ELSync
			cfg.VerifierSyncMode = sync.ELSync
		})))
	}

	for i := range ids.L2CLOpGethSequencerNodes {
		opt.Add(sysgo.WithOpGeth(ids.L2ELOpGethSequencerNodes[i]))
		opt.Add(sysgo.WithOpNode(ids.L2CLOpGethSequencerNodes[i], ids.L1CL, ids.L1EL, ids.L2ELOpGethSequencerNodes[i], sysgo.L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
			cfg.IsSequencer = true
		})))
	}

	for i := range ids.L2CLKonaRethSequencerNodes {
		opt.Add(sysgo.WithOpReth(ids.L2ELKonaRethSequencerNodes[i]))
		opt.Add(sysgo.WithKonaNode(ids.L2CLKonaRethSequencerNodes[i], ids.L1CL, ids.L1EL, ids.L2ELKonaRethSequencerNodes[i], sysgo.L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
			cfg.IsSequencer = true
			cfg.SequencerSyncMode = sync.ELSync
			cfg.VerifierSyncMode = sync.ELSync
		})))
	}

	for i := range ids.L2CLOpRethSequencerNodes {
		opt.Add(sysgo.WithOpReth(ids.L2ELOpRethSequencerNodes[i]))
		opt.Add(sysgo.WithOpNode(ids.L2CLOpRethSequencerNodes[i], ids.L1CL, ids.L1EL, ids.L2ELOpRethSequencerNodes[i], sysgo.L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
			cfg.IsSequencer = true
		})))
	}

	for i := range ids.L2CLKonaGethNodes {
		opt.Add(sysgo.WithOpGeth(ids.L2ELKonaGethNodes[i]))
		opt.Add(sysgo.WithKonaNode(ids.L2CLKonaGethNodes[i], ids.L1CL, ids.L1EL, ids.L2ELKonaGethNodes[i], sysgo.L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
			cfg.SequencerSyncMode = sync.ELSync
			cfg.VerifierSyncMode = sync.ELSync
		})))
	}

	for i := range ids.L2ELOpGethNodes {
		opt.Add(sysgo.WithOpGeth(ids.L2ELOpGethNodes[i]))
		opt.Add(sysgo.WithOpNode(ids.L2CLOpGethNodes[i], ids.L1CL, ids.L1EL, ids.L2ELOpGethNodes[i]))
	}

	for i := range ids.L2CLKonaRethNodes {
		opt.Add(sysgo.WithOpReth(ids.L2ELKonaRethNodes[i]))
		opt.Add(sysgo.WithKonaNode(ids.L2CLKonaRethNodes[i], ids.L1CL, ids.L1EL, ids.L2ELKonaRethNodes[i], sysgo.L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
			cfg.SequencerSyncMode = sync.ELSync
			cfg.VerifierSyncMode = sync.ELSync
		})))
	}

	for i := range ids.L2ELOpRethNodes {
		opt.Add(sysgo.WithOpReth(ids.L2ELOpRethNodes[i]))
		opt.Add(sysgo.WithOpNode(ids.L2CLOpRethNodes[i], ids.L1CL, ids.L1EL, ids.L2ELOpRethNodes[i]))
	}

	// Connect all nodes to each other in the p2p network.
	CLNodeIDs := ids.L2CLNodes()
	ELNodeIDs := ids.L2ELNodes()

	for i := range CLNodeIDs {
		for j := range i {
			opt.Add(sysgo.WithL2CLP2PConnection(CLNodeIDs[i], CLNodeIDs[j]))
			opt.Add(sysgo.WithL2ELP2PConnection(ELNodeIDs[i], ELNodeIDs[j]))
		}
	}

	opt.Add(sysgo.WithBatcher(ids.L2Batcher, ids.L1EL, CLNodeIDs[0], ELNodeIDs[0]))
	opt.Add(sysgo.WithProposer(ids.L2Proposer, ids.L1EL, &CLNodeIDs[0], nil))

	opt.Add(sysgo.WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ELNodeIDs[0]}))

	opt.Add(stack.Finally(func(orch *sysgo.Orchestrator) {
		*dest = ids
	}))

	return opt
}
