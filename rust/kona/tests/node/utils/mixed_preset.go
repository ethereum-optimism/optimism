package node_utils

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	devpresets "github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
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
	opSequencerGethInt := parseEnvInt("OP_SEQUENCER_WITH_GETH", DefaultOpSequencerGeth)
	konaSequencerGethInt := parseEnvInt("KONA_SEQUENCER_WITH_GETH", DefaultKonaSequencerGeth)
	opSequencerRethInt := parseEnvInt("OP_SEQUENCER_WITH_RETH", DefaultOpSequencerReth)
	konaSequencerRethInt := parseEnvInt("KONA_SEQUENCER_WITH_RETH", DefaultKonaSequencerReth)
	opValidatorGethInt := parseEnvInt("OP_VALIDATOR_WITH_GETH", DefaultOpValidatorGeth)
	opValidatorRethInt := parseEnvInt("OP_VALIDATOR_WITH_RETH", DefaultOpValidatorReth)
	konaValidatorGethInt := parseEnvInt("KONA_VALIDATOR_WITH_GETH", DefaultKonaValidatorGeth)
	konaValidatorRethInt := parseEnvInt("KONA_VALIDATOR_WITH_RETH", DefaultKonaValidatorReth)

	return L2NodeConfig{
		OpSequencerNodesWithGeth:   opSequencerGethInt,
		OpSequencerNodesWithReth:   opSequencerRethInt,
		OpNodesWithGeth:            opValidatorGethInt,
		OpNodesWithReth:            opValidatorRethInt,
		KonaSequencerNodesWithGeth: konaSequencerGethInt,
		KonaSequencerNodesWithReth: konaSequencerRethInt,
		KonaNodesWithGeth:          konaValidatorGethInt,
		KonaNodesWithReth:          konaValidatorRethInt,
	}
}

func parseEnvInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return fallback
	}
	return value
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
	Log log.Logger
	T   devtest.T

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode
	L1CL      *dsl.L1CLNode

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

func (m *MixedOpKonaPreset) L2ELNodes() []dsl.L2ELNode {
	return append(m.L2ELSequencerNodes(), m.L2ELValidatorNodes()...)
}

func (m *MixedOpKonaPreset) L2CLNodes() []dsl.L2CLNode {
	return append(m.L2CLSequencerNodes(), m.L2CLValidatorNodes()...)
}

func (m *MixedOpKonaPreset) L2CLValidatorNodes() []dsl.L2CLNode {
	return append(m.L2CLOpValidatorNodes, m.L2CLKonaValidatorNodes...)
}

func (m *MixedOpKonaPreset) L2CLSequencerNodes() []dsl.L2CLNode {
	return append(m.L2CLOpSequencerNodes, m.L2CLKonaSequencerNodes...)
}

func (m *MixedOpKonaPreset) L2ELValidatorNodes() []dsl.L2ELNode {
	return append(m.L2ELOpValidatorNodes, m.L2ELKonaValidatorNodes...)
}

func (m *MixedOpKonaPreset) L2ELSequencerNodes() []dsl.L2ELNode {
	return append(m.L2ELOpSequencerNodes, m.L2ELKonaSequencerNodes...)
}

func (m *MixedOpKonaPreset) L2CLKonaNodes() []dsl.L2CLNode {
	return append(m.L2CLKonaValidatorNodes, m.L2CLKonaSequencerNodes...)
}

func (m *MixedOpKonaPreset) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{m.L2Chain}
}

func NewMixedOpKona(t devtest.T) *MixedOpKonaPreset {
	return NewMixedOpKonaForConfig(t, ParseL2NodeConfigFromEnv())
}

func NewMixedOpKonaForConfig(t devtest.T, l2NodeConfig L2NodeConfig) *MixedOpKonaPreset {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: mixedOpKonaNodeSpecs(l2NodeConfig),
	})
	return NewMixedOpKonaFromRuntime(t, runtime)
}

func NewMixedOpKonaFromRuntime(t devtest.T, runtime *sysgo.MixedSingleChainRuntime) *MixedOpKonaPreset {
	preset, _ := mixedOpKonaFromRuntime(t, runtime)
	return preset
}

func mixedOpKonaFromRuntime(t devtest.T, runtime *sysgo.MixedSingleChainRuntime) (*MixedOpKonaPreset, *devpresets.MixedSingleChainFrontends) {
	frontends := devpresets.NewMixedSingleChainFrontends(t, runtime)
	t.Gate().GreaterOrEqual(len(frontends.Nodes), 2, "expected at least two mixed L2 nodes")
	out := &MixedOpKonaPreset{
		Log:       t.Logger(),
		T:         t,
		L1Network: frontends.L1Network,
		L1EL:      frontends.L1EL,
		L1CL:      frontends.L1CL,
		L2Chain:   frontends.L2Network,
		L2Batcher: frontends.L2Batcher,
		Wallet:    dsl.NewHDWallet(t, devkeys.TestMnemonic, 30),
		FaucetL1:  frontends.FaucetL1,
		Faucet:    frontends.FaucetL2,
	}
	for _, node := range frontends.Nodes {
		switch {
		case node.Spec.CLKind == sysgo.MixedL2CLOpNode && node.Spec.IsSequencer:
			out.L2ELOpSequencerNodes = append(out.L2ELOpSequencerNodes, *node.EL)
			out.L2CLOpSequencerNodes = append(out.L2CLOpSequencerNodes, *node.CL)
		case node.Spec.CLKind == sysgo.MixedL2CLOpNode && !node.Spec.IsSequencer:
			out.L2ELOpValidatorNodes = append(out.L2ELOpValidatorNodes, *node.EL)
			out.L2CLOpValidatorNodes = append(out.L2CLOpValidatorNodes, *node.CL)
		case node.Spec.CLKind == sysgo.MixedL2CLKona && node.Spec.IsSequencer:
			out.L2ELKonaSequencerNodes = append(out.L2ELKonaSequencerNodes, *node.EL)
			out.L2CLKonaSequencerNodes = append(out.L2CLKonaSequencerNodes, *node.CL)
		case node.Spec.CLKind == sysgo.MixedL2CLKona && !node.Spec.IsSequencer:
			out.L2ELKonaValidatorNodes = append(out.L2ELKonaValidatorNodes, *node.EL)
			out.L2CLKonaValidatorNodes = append(out.L2CLKonaValidatorNodes, *node.CL)
		}
		if out.Funder == nil && node.Spec.IsSequencer {
			out.Funder = dsl.NewFunder(out.Wallet, out.Faucet, node.EL)
		}
	}
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)
	return out, frontends
}

func mixedOpKonaNodeSpecs(cfg L2NodeConfig) []sysgo.MixedSingleChainNodeSpec {
	var specs []sysgo.MixedSingleChainNodeSpec
	appendSpecs := func(count int, elPrefix, clPrefix string, elKind sysgo.MixedL2ELKind, clKind sysgo.MixedL2CLKind, isSequencer bool) {
		for i := 0; i < count; i++ {
			specs = append(specs, sysgo.MixedSingleChainNodeSpec{
				ELKey:       fmt.Sprintf("%s-%d", elPrefix, i),
				CLKey:       fmt.Sprintf("%s-%d", clPrefix, i),
				ELKind:      elKind,
				CLKind:      clKind,
				IsSequencer: isSequencer,
			})
		}
	}

	appendSpecs(cfg.OpSequencerNodesWithGeth, "el-geth-op-sequencer", "cl-geth-op-sequencer", sysgo.MixedL2ELOpGeth, sysgo.MixedL2CLOpNode, true)
	appendSpecs(cfg.OpSequencerNodesWithReth, "el-reth-op-sequencer", "cl-reth-op-sequencer", sysgo.MixedL2ELOpReth, sysgo.MixedL2CLOpNode, true)
	appendSpecs(cfg.KonaSequencerNodesWithGeth, "el-geth-kona-sequencer", "cl-geth-kona-sequencer", sysgo.MixedL2ELOpGeth, sysgo.MixedL2CLKona, true)
	appendSpecs(cfg.KonaSequencerNodesWithReth, "el-reth-kona-sequencer", "cl-reth-kona-sequencer", sysgo.MixedL2ELOpReth, sysgo.MixedL2CLKona, true)

	appendSpecs(cfg.OpNodesWithGeth, "el-geth-op-validator", "cl-geth-op-validator", sysgo.MixedL2ELOpGeth, sysgo.MixedL2CLOpNode, false)
	appendSpecs(cfg.OpNodesWithReth, "el-reth-op-validator", "cl-reth-op-validator", sysgo.MixedL2ELOpReth, sysgo.MixedL2CLOpNode, false)
	appendSpecs(cfg.KonaNodesWithGeth, "el-geth-kona-validator", "cl-geth-kona-validator", sysgo.MixedL2ELOpGeth, sysgo.MixedL2CLKona, false)
	appendSpecs(cfg.KonaNodesWithReth, "el-reth-kona-validator", "cl-reth-kona-validator", sysgo.MixedL2ELOpReth, sysgo.MixedL2CLKona, false)

	return specs
}
