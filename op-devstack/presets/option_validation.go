package presets

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type optionKinds uint64

const (
	optionKindDeployer optionKinds = 1 << iota
	optionKindBatcher
	optionKindProposer
	optionKindOPRBuilder
	optionKindGlobalL2CL
	optionKindGlobalSyncTesterEL
	optionKindL1EL
	optionKindAddedGameType
	optionKindRespectedGameType
	optionKindChallengerCannonKona
	optionKindTimeTravel
	optionKindMaxSequencingWindow
	optionKindRequireInteropNotAtGen
	optionKindAfterBuild
	optionKindProofValidation
)

const allOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindProposer |
	optionKindOPRBuilder |
	optionKindGlobalL2CL |
	optionKindGlobalSyncTesterEL |
	optionKindL1EL |
	optionKindAddedGameType |
	optionKindRespectedGameType |
	optionKindChallengerCannonKona |
	optionKindTimeTravel |
	optionKindMaxSequencingWindow |
	optionKindRequireInteropNotAtGen |
	optionKindAfterBuild |
	optionKindProofValidation

var optionKindLabels = []struct {
	kind  optionKinds
	label string
}{
	{kind: optionKindDeployer, label: "deployer options"},
	{kind: optionKindBatcher, label: "batcher options"},
	{kind: optionKindProposer, label: "proposer options"},
	{kind: optionKindOPRBuilder, label: "builder options"},
	{kind: optionKindGlobalL2CL, label: "L2 CL options"},
	{kind: optionKindGlobalSyncTesterEL, label: "sync tester EL options"},
	{kind: optionKindL1EL, label: "L1 EL options"},
	{kind: optionKindAddedGameType, label: "added game types"},
	{kind: optionKindRespectedGameType, label: "respected game types"},
	{kind: optionKindChallengerCannonKona, label: "challenger cannon-kona"},
	{kind: optionKindTimeTravel, label: "time travel"},
	{kind: optionKindMaxSequencingWindow, label: "max sequencing window"},
	{kind: optionKindRequireInteropNotAtGen, label: "interop-not-at-genesis"},
	{kind: optionKindAfterBuild, label: "after-build hooks"},
	{kind: optionKindProofValidation, label: "proof-validation hooks"},
}

func (k optionKinds) String() string {
	if k == 0 {
		return "none"
	}

	names := make([]string, 0, len(optionKindLabels))
	for _, label := range optionKindLabels {
		if k&label.kind == 0 {
			continue
		}
		names = append(names, label.label)
	}
	if unknown := k &^ allOptionKinds; unknown != 0 {
		names = append(names, fmt.Sprintf("unknown(%#x)", uint64(unknown)))
	}
	return strings.Join(names, ", ")
}

func unsupportedPresetOptionKinds(opts Option, supported optionKinds) optionKinds {
	if opts == nil {
		return 0
	}
	return opts.optionKinds() &^ supported
}

func collectSupportedPresetConfig(t devtest.T, presetName string, opts []Option, supported optionKinds) (sysgo.PresetConfig, CombinedOption) {
	cfg, combined := collectPresetConfig(opts)
	if unsupported := unsupportedPresetOptionKinds(combined, supported); unsupported != 0 {
		t.Require().FailNowf("%s does not support preset options: %s", presetName, unsupported)
	}
	return cfg, combined
}

const minimalPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindProposer |
	optionKindGlobalL2CL |
	optionKindL1EL |
	optionKindAddedGameType |
	optionKindRespectedGameType |
	optionKindChallengerCannonKona |
	optionKindTimeTravel |
	optionKindAfterBuild |
	optionKindProofValidation

const minimalWithConductorsPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindProposer |
	optionKindGlobalL2CL |
	optionKindL1EL |
	optionKindAddedGameType |
	optionKindRespectedGameType |
	optionKindTimeTravel |
	optionKindAfterBuild |
	optionKindProofValidation

const simpleWithSyncTesterPresetSupportedOptionKinds = minimalPresetSupportedOptionKinds |
	optionKindGlobalSyncTesterEL

const singleChainInteropPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindProposer |
	optionKindGlobalL2CL |
	optionKindL1EL |
	optionKindAddedGameType |
	optionKindRespectedGameType |
	optionKindTimeTravel |
	optionKindMaxSequencingWindow |
	optionKindRequireInteropNotAtGen |
	optionKindAfterBuild |
	optionKindProofValidation

const simpleInteropSuperProofsPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindProposer |
	optionKindGlobalL2CL |
	optionKindL1EL |
	optionKindChallengerCannonKona |
	optionKindTimeTravel |
	optionKindMaxSequencingWindow |
	optionKindRequireInteropNotAtGen

const supernodeProofsPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindChallengerCannonKona |
	optionKindL1EL

const twoL2SupernodePresetSupportedOptionKinds = optionKindDeployer |
	optionKindL1EL

const twoL2SupernodeInteropPresetSupportedOptionKinds = optionKindDeployer |
	optionKindTimeTravel |
	optionKindL1EL

const singleChainWithFlashblocksPresetSupportedOptionKinds = optionKindDeployer |
	optionKindOPRBuilder
