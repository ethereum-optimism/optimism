package presets

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type Option interface {
	applyConfig(cfg *sysgo.PresetConfig)
	applyPreset(target any)
}

type option struct {
	applyFn       func(cfg *sysgo.PresetConfig)
	applyPresetFn func(target any)
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

func AfterBuild(fn func(target any)) Option {
	return option{applyPresetFn: fn}
}

func collectPresetConfig(opts []Option) (sysgo.PresetConfig, CombinedOption) {
	cfg := sysgo.NewPresetConfig()
	combined := Combine(opts...)
	combined.applyConfig(&cfg)
	return cfg, combined
}

func WithDeployerOptions(opts ...sysgo.DeployerOption) Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.DeployerOptions = append(cfg.DeployerOptions, opts...)
		},
	}
}

func WithBatcherOption(opt sysgo.BatcherOption) Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.BatcherOptions = append(cfg.BatcherOptions, opt)
		},
	}
}

func WithGlobalL2CLOption(opt sysgo.L2CLOption) Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.GlobalL2CLOptions = append(cfg.GlobalL2CLOptions, opt)
		},
	}
}

func WithGlobalSyncTesterELOption(opt sysgo.SyncTesterELOption) Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.GlobalSyncTesterELOptions = append(cfg.GlobalSyncTesterELOptions, opt)
		},
	}
}

func WithProposerOption(opt sysgo.ProposerOption) Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			if opt == nil {
				return
			}
			cfg.ProposerOptions = append(cfg.ProposerOptions, opt)
		},
	}
}

func WithOPRBuilderOption(opt sysgo.OPRBuilderNodeOption) Option {
	return option{
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
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.AddedGameTypes = append(cfg.AddedGameTypes, gameType)
		},
	}
}

func WithRespectedGameTypeOverride(gameType gameTypes.GameType) Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.RespectedGameTypes = append(cfg.RespectedGameTypes, gameType)
		},
	}
}

func WithCannonKonaGameTypeAdded() Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.EnableCannonKonaForChall = true
			cfg.AddedGameTypes = append(cfg.AddedGameTypes, gameTypes.CannonKonaGameType)
		},
	}
}

func WithChallengerCannonKonaEnabled() Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.EnableCannonKonaForChall = true
		},
	}
}

func WithTimeTravelEnabled() Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.EnableTimeTravel = true
		},
	}
}

func WithMaxSequencingWindow(max uint64) Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			v := max
			cfg.MaxSequencingWindow = &v
		},
	}
}

func WithRequireInteropNotAtGenesis() Option {
	return option{
		applyFn: func(cfg *sysgo.PresetConfig) {
			cfg.RequireInteropNotAtGen = true
		},
	}
}
