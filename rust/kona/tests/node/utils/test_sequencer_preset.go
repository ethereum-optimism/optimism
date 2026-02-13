package node_utils

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type MinimalWithTestSequencersPreset struct {
	*MixedOpKonaPreset

	TestSequencer dsl.TestSequencer
}

func NewMixedOpKonaWithTestSequencer(t devtest.T) *MinimalWithTestSequencersPreset {
	return NewMixedOpKonaWithTestSequencerForConfig(t, ParseL2NodeConfigFromEnv())
}

func NewMixedOpKonaWithTestSequencerForConfig(t devtest.T, l2Config L2NodeConfig) *MinimalWithTestSequencersPreset {
	l2Config = withRequiredOpSequencerForTestSequencer(l2Config)

	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs:         mixedOpKonaNodeSpecs(l2Config),
		WithTestSequencer: true,
		TestSequencerName: "test-sequencer",
	})
	mixedPreset, frontends := mixedOpKonaFromRuntime(t, runtime)
	t.Require().NotNil(frontends.TestSequencer, "expected test sequencer frontend")

	return &MinimalWithTestSequencersPreset{
		MixedOpKonaPreset: mixedPreset,
		TestSequencer:     *frontends.TestSequencer,
	}
}

func withRequiredOpSequencerForTestSequencer(l2Config L2NodeConfig) L2NodeConfig {
	if l2Config.OpSequencerNodes() == 0 {
		l2Config.OpSequencerNodesWithGeth = 1
	}
	return l2Config
}
