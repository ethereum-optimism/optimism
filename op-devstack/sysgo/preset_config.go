package sysgo

import (
	"fmt"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PreGenesisSuperGameConfig struct {
	ClaimedOutputs []eth.Bytes32
}

// ZKDisputeGameConfig configures the shared ZK dispute game installed after
// the interop migration. OPCM injects the release-owned SP1 adapter.
type ZKDisputeGameConfig struct {
	MaxChallengeDuration time.Duration
	MaxProveDuration     time.Duration
}

func (c ZKDisputeGameConfig) validate() error {
	if c.MaxChallengeDuration <= 0 {
		return fmt.Errorf("ZK maximum challenge duration must be positive")
	}
	if c.MaxChallengeDuration%time.Second != 0 {
		return fmt.Errorf("ZK maximum challenge duration must use whole seconds")
	}
	if c.MaxProveDuration <= 0 {
		return fmt.Errorf("ZK maximum prove duration must be positive")
	}
	if c.MaxProveDuration%time.Second != 0 {
		return fmt.Errorf("ZK maximum prove duration must use whole seconds")
	}
	return nil
}

// PresetConfig captures preset constructor mutations.
// It is independent from orchestrator lifecycle hooks.
type PresetConfig struct {
	LocalContractArtifactsPath string
	DeployerOptions            []DeployerOption
	BatcherOptions             []BatcherOption
	ProposerOptions            []ProposerOption
	OpRethOptions              []OpRethOption
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
	ZKDisputeGame           *ZKDisputeGameConfig
	ZKProposerOptions       []ZKProposerOption
	// SkipHonestProposer skips starting the honest proposer (op-proposer, or kona-sp1-proposer for the ZK preset).
	SkipHonestProposer bool
	// SkipHonestChallenger skips starting the honest challenger.
	SkipHonestChallenger bool
	// SupernodeVerifierSyncMode overrides the supernode VN's sync mode when set.
	SupernodeVerifierSyncMode *nodeSync.Mode
	// InteropActivationDelaySeconds offsets Interop activation past genesis (0 = at genesis).
	InteropActivationDelaySeconds uint64
	// InteropAtGenesis activates Interop on the L2 chain at genesis and provisions a
	// DependencySet for op-node startup (without a supervisor). Required by tests that
	// exercise Interop-gated consensus features (e.g. SDM PostExec) on the default
	// single-chain runtime.
	InteropAtGenesis bool
	// SilhouetteChain, when set, names the runtime chain ("l2a" / "l2b") that is run as a
	// SILHOUETTE chain: no batcher, its history posted to L1 as proof batches, and a second
	// supernode that derives it from those proofs alone. See silhouette_runtime.go for what that
	// costs and why a second supernode is not optional.
	SilhouetteChain string
	// SilhouetteSequencerPosture puts the preset's OWN supernode — the one that sequences the
	// silhouette chain on its real execution client — into the sequencer posture, restarting it with
	// a `proven-head` manifest.
	//
	// It is opt-in rather than implied by SilhouetteChain because the difference it makes is the
	// thing worth testing on both sides. Without it the sequencer side runs the chain ordinarily and
	// its cross-safe frontier freezes from a perfectly healthy chain (hazard 3), which is what
	// TestSilhouetteCrossChainPinsThenAdvances asserts as its control. Turning it on by default
	// would delete that control.
	SilhouetteSequencerPosture bool
	// SupernodeVNSequencerForBootstrap, in the light-sequencer supernode interop preset,
	// enables sequencing on the supernode VN and starts the light follow-mode ELSync
	// sequencers stopped, so the VN can bootstrap the chain the light sequencers EL-sync
	// from before a test hands off sequencing to them.
	SupernodeVNSequencerForBootstrap bool
}

func NewPresetConfig() PresetConfig {
	return PresetConfig{}
}
