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
	optionKindOpReth
	optionKindGlobalL2CL
	optionKindGlobalSyncTesterEL
	optionKindL1EL
	optionKindAddedGameType
	optionKindRespectedGameType
	optionKindTimeTravel
	optionKindMaxSequencingWindow
	optionKindRequireInteropNotAtGen
	optionKindAfterBuild
	optionKindProofValidation
	optionKindMessageExpiryWindow
	optionKindInteropLogBackfill
	optionKindInteropFilter
	optionKindPreGenesisSuperGame
	optionKindSkipHonestProposer
	optionKindSkipHonestChallenger
	optionKindSupernodeVerifierSyncMode
	optionKindInteropActivationDelay
	optionKindInteropAtGenesis
	optionKindSupernodeVNSequencerForBootstrap
	optionKindZKDisputeGame
	optionKindZKProposer
	optionKindConductorFastHealthChecks
)

const allOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindProposer |
	optionKindOPRBuilder |
	optionKindOpReth |
	optionKindGlobalL2CL |
	optionKindGlobalSyncTesterEL |
	optionKindL1EL |
	optionKindAddedGameType |
	optionKindRespectedGameType |
	optionKindTimeTravel |
	optionKindMaxSequencingWindow |
	optionKindRequireInteropNotAtGen |
	optionKindAfterBuild |
	optionKindProofValidation |
	optionKindMessageExpiryWindow |
	optionKindInteropLogBackfill |
	optionKindInteropFilter |
	optionKindPreGenesisSuperGame |
	optionKindSkipHonestProposer |
	optionKindSkipHonestChallenger |
	optionKindSupernodeVerifierSyncMode |
	optionKindInteropActivationDelay |
	optionKindInteropAtGenesis |
	optionKindSupernodeVNSequencerForBootstrap |
	optionKindZKDisputeGame |
	optionKindZKProposer |
	optionKindConductorFastHealthChecks

var optionKindLabels = []struct {
	kind  optionKinds
	label string
}{
	{kind: optionKindDeployer, label: "deployer options"},
	{kind: optionKindBatcher, label: "batcher options"},
	{kind: optionKindProposer, label: "proposer options"},
	{kind: optionKindOPRBuilder, label: "builder options"},
	{kind: optionKindOpReth, label: "op-reth options"},
	{kind: optionKindGlobalL2CL, label: "L2 CL options"},
	{kind: optionKindGlobalSyncTesterEL, label: "sync tester EL options"},
	{kind: optionKindL1EL, label: "L1 EL options"},
	{kind: optionKindAddedGameType, label: "added game types"},
	{kind: optionKindRespectedGameType, label: "respected game types"},
	{kind: optionKindTimeTravel, label: "time travel"},
	{kind: optionKindMaxSequencingWindow, label: "max sequencing window"},
	{kind: optionKindRequireInteropNotAtGen, label: "interop-not-at-genesis"},
	{kind: optionKindAfterBuild, label: "after-build hooks"},
	{kind: optionKindProofValidation, label: "proof-validation hooks"},
	{kind: optionKindMessageExpiryWindow, label: "message expiry window"},
	{kind: optionKindInteropLogBackfill, label: "interop log backfill depth"},
	{kind: optionKindInteropFilter, label: "interop filter"},
	{kind: optionKindPreGenesisSuperGame, label: "pre-genesis super game"},
	{kind: optionKindSkipHonestProposer, label: "skip honest proposer"},
	{kind: optionKindSkipHonestChallenger, label: "skip honest challenger"},
	{kind: optionKindSupernodeVerifierSyncMode, label: "supernode verifier sync mode"},
	{kind: optionKindInteropActivationDelay, label: "interop activation delay"},
	{kind: optionKindInteropAtGenesis, label: "interop at genesis"},
	{kind: optionKindSupernodeVNSequencerForBootstrap, label: "supernode VN sequencer for bootstrap"},
	{kind: optionKindZKDisputeGame, label: "ZK dispute game"},
	{kind: optionKindZKProposer, label: "ZK proposer options"},
	{kind: optionKindConductorFastHealthChecks, label: "conductor fast health checks"},
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
	t.Require().NoError(validatePresetConfig(cfg), "%s has invalid preset options", presetName)
	return cfg, combined
}

func validatePresetConfig(cfg sysgo.PresetConfig) error {
	if cfg.ZKDisputeGame != nil && cfg.ZKDisputeGame.MaxProveDuration == 0 {
		return fmt.Errorf("WithZKChallengeDuration requires a preceding WithZK")
	}
	if len(cfg.ZKProposerOptions) > 0 && cfg.ZKDisputeGame == nil {
		return fmt.Errorf("ZK proposer options require WithZK")
	}
	return nil
}

const minimalPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindProposer |
	optionKindOpReth |
	optionKindGlobalL2CL |
	optionKindL1EL |
	optionKindAddedGameType |
	optionKindRespectedGameType |
	optionKindTimeTravel |
	optionKindAfterBuild |
	optionKindProofValidation |
	optionKindSkipHonestProposer

const minimalWithConductorsPresetSupportedOptionKinds = minimalPresetSupportedOptionKinds |
	optionKindConductorFastHealthChecks

// Builds on the minimal set, op-reth options included: the added node is a sync-tester EL, not a
// sequencing candidate, so the sequencer-builds / stock-verifies split holds.
const simpleWithSyncTesterPresetSupportedOptionKinds = minimalPresetSupportedOptionKinds |
	optionKindGlobalSyncTesterEL

// singleSupernodeWithSyncTesterPresetSupportedOptionKinds covers exactly what
// the runtime in singlechain_supernode_synctester_variant.go actually wires.
// Proposer / game-type / proof-validation options are intentionally excluded:
// this preset has no proposer and no dispute game surface, so accepting them
// would be a silent no-op footgun.
const singleSupernodeWithSyncTesterPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindGlobalL2CL |
	optionKindGlobalSyncTesterEL |
	optionKindL1EL |
	optionKindTimeTravel |
	optionKindAfterBuild |
	optionKindSupernodeVerifierSyncMode |
	optionKindInteropActivationDelay

const supernodeProofsPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindProposer |
	optionKindL1EL |
	optionKindTimeTravel |
	optionKindMessageExpiryWindow |
	optionKindSkipHonestProposer

const singleChainInteropNoSupernodePresetSupportedOptionKinds = optionKindOpReth

const twoL2SupernodeProofsPresetSupportedOptionKinds = supernodeProofsPresetSupportedOptionKinds |
	optionKindPreGenesisSuperGame |
	optionKindZKDisputeGame |
	optionKindZKProposer |
	optionKindSkipHonestChallenger

const twoL2SupernodePresetSupportedOptionKinds = optionKindDeployer |
	optionKindL1EL

const twoL2SupernodeInteropPresetSupportedOptionKinds = optionKindDeployer |
	optionKindBatcher |
	optionKindTimeTravel |
	optionKindL1EL |
	optionKindInteropLogBackfill |
	optionKindInteropFilter |
	optionKindPreGenesisSuperGame |
	optionKindSupernodeVNSequencerForBootstrap

// twoL2SupernodeLightSequencerPresetSupportedOptionKinds additionally accepts
// L2 CL options: the light-sequencer runtime is the only two-L2 supernode
// variant that wires GlobalL2CLOptions (to the light sequencer CLs), so the
// option is accepted here and nowhere else to avoid a silent no-op.
const twoL2SupernodeLightSequencerPresetSupportedOptionKinds = twoL2SupernodeInteropPresetSupportedOptionKinds |
	optionKindGlobalL2CL

const singleChainWithFlashblocksPresetSupportedOptionKinds = optionKindDeployer |
	optionKindOPRBuilder |
	optionKindOpReth |
	optionKindInteropAtGenesis
