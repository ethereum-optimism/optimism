package sysgo

import (
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
)

// AddedSuperGameType requests that a super dispute game type be registered on
// the chain's DisputeGameFactory after initial deploy. Super game impls live
// on the shared OPCMContainer, so this records only the (type, prestate) pair.
type AddedSuperGameType struct {
	GameType gameTypes.GameType
	Prestate common.Hash
}

// PresetConfig captures preset constructor mutations.
// It is independent from orchestrator lifecycle hooks.
type PresetConfig struct {
	LocalContractArtifactsPath    string
	DeployerOptions               []DeployerOption
	BatcherOptions                []BatcherOption
	ProposerOptions               []ProposerOption
	OPRBuilderOptions             []OPRBuilderNodeOption
	GlobalL2CLOptions             []L2CLOption
	GlobalSyncTesterELOptions     []SyncTesterELOption
	L1ELKind                      string
	L1GethExecPath                string
	AddedGameTypes                []gameTypes.GameType
	AddedSuperGameTypes           []AddedSuperGameType
	RespectedGameTypes            []gameTypes.GameType
	EnableCannonKonaForChallenger bool
	EnableTimeTravel              bool
	MaxSequencingWindow           *uint64
	RequireInteropNotAtGen        bool
	MessageExpiryWindow           *uint64
	UseInteropFilter              bool
	// InteropLogBackfillDepth, if non-zero, configures the supernode to backfill
	// initiating-message logs backward from the tip by this duration at startup.
	InteropLogBackfillDepth time.Duration
}

func NewPresetConfig() PresetConfig {
	return PresetConfig{}
}
