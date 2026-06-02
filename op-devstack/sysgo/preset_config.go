package sysgo

import (
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PreGenesisSuperGameConfig struct {
	ClaimedOutputs []eth.Bytes32
}

// PresetConfig captures preset constructor mutations.
// It is independent from orchestrator lifecycle hooks.
type PresetConfig struct {
	LocalContractArtifactsPath string
	DeployerOptions            []DeployerOption
	BatcherOptions             []BatcherOption
	ProposerOptions            []ProposerOption
	OPRBuilderOptions          []OPRBuilderNodeOption
	GlobalL2CLOptions          []L2CLOption
	GlobalSyncTesterELOptions  []SyncTesterELOption
	L1ELKind                   string
	L1GethExecPath             string
	AddedGameTypes             []gameTypes.GameType
	RespectedGameTypes         []gameTypes.GameType
	EnableTimeTravel           bool
	MaxSequencingWindow        *uint64
	RequireInteropNotAtGen     bool
	MessageExpiryWindow        *uint64
	UseInteropFilter           bool
	// InteropLogBackfillDepth, if non-zero, configures the supernode to backfill
	// initiating-message logs backward from the tip by this duration at startup.
	InteropLogBackfillDepth time.Duration
	PreGenesisSuperGame     *PreGenesisSuperGameConfig
	// SkipHonestProposer skips starting op-proposer.
	SkipHonestProposer bool
	// SupernodeVerifierSyncMode overrides the supernode VN's sync mode when set.
	SupernodeVerifierSyncMode *nodeSync.Mode
	// InteropActivationDelaySeconds offsets Interop activation past genesis (0 = at genesis).
	InteropActivationDelaySeconds uint64
}

func NewPresetConfig() PresetConfig {
	return PresetConfig{}
}
