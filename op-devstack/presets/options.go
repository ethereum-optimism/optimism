package presets

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
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

func WithCannonKonaGameTypeAdded() Option {
	return option{
		kinds: optionKindAddedGameType | optionKindChallengerCannonKona,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.EnableCannonKonaForChall = true
			cfg.AddedGameTypes = append(cfg.AddedGameTypes, gameTypes.CannonKonaGameType)
		},
	}
}

func WithChallengerCannonKonaEnabled() Option {
	return option{
		kinds: optionKindChallengerCannonKona,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.EnableCannonKonaForChall = true
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

func WithRequireInteropNotAtGenesis() Option {
	return option{
		kinds: optionKindRequireInteropNotAtGen,
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.RequireInteropNotAtGen = true
		},
	}
}
