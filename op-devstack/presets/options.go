package presets

import (
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Option interface {
	applyConfig(cfg *sysgo.PresetConfig)
	applyPreset(target any)
	optionKinds() optionKinds
}

type option struct {
	applyFn       func(cfg *sysgo.PresetConfig)
	applyPresetFn func(target any)
	kinds         optionKinds
}

func (o option) applyConfig(cfg *sysgo.PresetConfig) {
	if o.applyFn == nil {
		return
	}
	o.applyFn(cfg)
}

func (o option) applyPreset(target any) {
	if o.applyPresetFn != nil {
		o.applyPresetFn(target)
	}
}

func (o option) optionKinds() optionKinds {
	return o.kinds
}

type CombinedOption []Option

func Combine(opts ...Option) CombinedOption {
	return CombinedOption(opts)
}

func (c CombinedOption) applyConfig(cfg *sysgo.PresetConfig) {
	for _, opt := range c {
		if opt == nil {
			continue
		}
		opt.applyConfig(cfg)
	}
}

func (c CombinedOption) applyPreset(target any) {
	for _, opt := range c {
		if opt == nil {
			continue
		}
		opt.applyPreset(target)
	}
}

func (c CombinedOption) optionKinds() optionKinds {
	var kinds optionKinds
	for _, opt := range c {
		if opt == nil {
			continue
		}
		kinds |= opt.optionKinds()
	}
	return kinds
}

func AfterBuild(fn func(target any)) Option {
	var kinds optionKinds
	if fn != nil {
		kinds = optionKindAfterBuild
	}
	return option{applyPresetFn: fn, kinds: kinds}
}

func collectPresetConfig(opts []Option) (sysgo.PresetConfig, CombinedOption) {
	cfg := sysgo.NewPresetConfig()
	combined := Combine(opts...)
	combined.applyConfig(&cfg)
	return cfg, combined
}

func WithDeployerOptions(opts ...sysgo.DeployerOption) Option {
	var kinds optionKinds
	for _, opt := range opts {
		if opt != nil {
			kinds = optionKindDeployer
			break
		}
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.DeployerOptions = append(cfg.DeployerOptions, opts...)
		},
	}
}

// WithLocalContractSourcesAt configures a preset to load local contracts-bedrock
// artifacts from the supplied directory instead of resolving them relative to
// the process working directory.
func WithLocalContractSourcesAt(path string) Option {
	var kinds optionKinds
	if path != "" {
		kinds = optionKindDeployer
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			if path == "" {
				return
			}
			cfg.LocalContractArtifactsPath = path
		},
	}
}

func WithBatcherOption(opt sysgo.BatcherOption) Option {
	var kinds optionKinds
	if opt != nil {
		kinds = optionKindBatcher
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.BatcherOptions = append(cfg.BatcherOptions, opt)
		},
	}
}

func WithGlobalL2CLOption(opt sysgo.L2CLOption) Option {
	var kinds optionKinds
	if opt != nil {
		kinds = optionKindGlobalL2CL
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.GlobalL2CLOptions = append(cfg.GlobalL2CLOptions, opt)
		},
	}
}

func WithGlobalSyncTesterELOption(opt sysgo.SyncTesterELOption) Option {
	var kinds optionKinds
	if opt != nil {
		kinds = optionKindGlobalSyncTesterEL
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.GlobalSyncTesterELOptions = append(cfg.GlobalSyncTesterELOptions, opt)
		},
	}
}

func WithL1Geth(execPath string) Option {
	return option{
		kinds: optionKindL1EL,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.L1ELKind = "geth"
			cfg.L1GethExecPath = execPath
		},
	}
}

func WithProposerOption(opt sysgo.ProposerOption) Option {
	var kinds optionKinds
	if opt != nil {
		kinds = optionKindProposer
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.ProposerOptions = append(cfg.ProposerOptions, opt)
		},
	}
}

func WithOPRBuilderOption(opt sysgo.OPRBuilderNodeOption) Option {
	var kinds optionKinds
	if opt != nil {
		kinds = optionKindOPRBuilder
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.OPRBuilderOptions = append(cfg.OPRBuilderOptions, opt)
		},
	}
}

func WithGameTypeAdded(gameType gameTypes.GameType) Option {
	return option{
		kinds: optionKindAddedGameType,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.AddedGameTypes = append(cfg.AddedGameTypes, gameType)
		},
	}
}

func WithRespectedGameTypeOverride(gameType gameTypes.GameType) Option {
	return option{
		kinds: optionKindRespectedGameType,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.RespectedGameTypes = append(cfg.RespectedGameTypes, gameType)
		},
	}
}

func WithTimeTravelEnabled() Option {
	return option{
		kinds: optionKindTimeTravel,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.EnableTimeTravel = true
		},
	}
}

func WithMaxSequencingWindow(max uint64) Option {
	return option{
		kinds: optionKindMaxSequencingWindow,
		applyFn: func(cfg *sysgo.PresetConfig) {
			v := max
			cfg.MaxSequencingWindow = &v
		},
	}
}

// WithInteropFilter enables the in-process op-interop-filter for EL transaction
// validation. Only supported on supernode interop presets.
func WithInteropFilter() Option {
	return option{
		kinds: optionKindInteropFilter,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.UseInteropFilter = true
		},
	}
}

func WithRequireInteropNotAtGenesis() Option {
	return option{
		kinds: optionKindRequireInteropNotAtGen,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.RequireInteropNotAtGen = true
		},
	}
}

// WithMessageExpiryWindow configures the message expiry window (in seconds)
// used by the dependency set. This controls how long cross-chain messages
// remain valid before they expire.
func WithMessageExpiryWindow(window uint64) Option {
	return option{
		kinds: optionKindMessageExpiryWindow,
		applyFn: func(cfg *sysgo.PresetConfig) {
			v := window
			cfg.MessageExpiryWindow = &v
		},
	}
}

// WithL2BlockTimes configures per-chain L2 block times via the deployer.
// The blockTimes map keys are L2 chain IDs and values are the desired block
// time in seconds for that chain.
func WithL2BlockTimes(blockTimes map[eth.ChainID]uint64) Option {
	return WithDeployerOptions(sysgo.WithL2BlockTimes(blockTimes))
}

// WithInteropLogBackfillDepth configures the supernode to pre-ingest
// initiating-message logs backward from the tip by the given duration at
// startup. Zero disables backfill (the default).
func WithInteropLogBackfillDepth(d time.Duration) Option {
	var kinds optionKinds
	if d > 0 {
		kinds = optionKindInteropLogBackfill
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.InteropLogBackfillDepth = d
		},
	}
}

// WithoutHonestProposer skips starting op-proposer.
func WithoutHonestProposer() Option {
	return option{
		kinds: optionKindSkipHonestProposer,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.SkipHonestProposer = true
		},
	}
}

// WithPreGenesisSuperGame seeds one invalid super dispute game before the
// rollup start block so tests can exercise supernode/challenger behaviour
// when a game's L1 head predates rollup genesis. The claimed outputs follow
// the preset chain order (`l2a`, `l2b` for two-chain presets).
func WithPreGenesisSuperGame(claimedOutputs ...eth.Bytes32) Option {
	var kinds optionKinds
	if len(claimedOutputs) > 0 {
		kinds = optionKindPreGenesisSuperGame
	}
	return option{
		kinds: kinds,
		applyFn: func(cfg *sysgo.PresetConfig) {
			if len(claimedOutputs) == 0 {
				return
			}
			cfg.PreGenesisSuperGame = &sysgo.PreGenesisSuperGameConfig{
				ClaimedOutputs: append([]eth.Bytes32(nil), claimedOutputs...),
			}
		},
	}
}
