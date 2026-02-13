package sysgo

import gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"

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
	EnableCannonKonaForChall   bool
	EnableTimeTravel           bool
	MaxSequencingWindow        *uint64
	RequireInteropNotAtGen     bool
}

type PresetOption interface {
	apply(cfg *PresetConfig)
}

type presetOptionFn func(cfg *PresetConfig)

func (fn presetOptionFn) apply(cfg *PresetConfig) {
	fn(cfg)
}

func NewPresetConfig(opts ...PresetOption) PresetConfig {
	cfg := PresetConfig{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.apply(&cfg)
	}
	return cfg
}

func WithDeployerOptions(opts ...DeployerOption) PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		cfg.DeployerOptions = append(cfg.DeployerOptions, opts...)
	})
}

func WithBatcherOption(opt BatcherOption) PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		if opt == nil {
			return
		}
		cfg.BatcherOptions = append(cfg.BatcherOptions, opt)
	})
}

func WithProposerOption(opt ProposerOption) PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		if opt == nil {
			return
		}
		cfg.ProposerOptions = append(cfg.ProposerOptions, opt)
	})
}

func WithOPRBuilderOption(opt OPRBuilderNodeOption) PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		if opt == nil {
			return
		}
		cfg.OPRBuilderOptions = append(cfg.OPRBuilderOptions, opt)
	})
}

func WithGlobalL2CLOption(opt L2CLOption) PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		if opt == nil {
			return
		}
		cfg.GlobalL2CLOptions = append(cfg.GlobalL2CLOptions, opt)
	})
}

func WithGlobalSyncTesterELOption(opt SyncTesterELOption) PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		if opt == nil {
			return
		}
		cfg.GlobalSyncTesterELOptions = append(cfg.GlobalSyncTesterELOptions, opt)
	})
}

func WithGameTypeAdded(gameType gameTypes.GameType) PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		cfg.AddedGameTypes = append(cfg.AddedGameTypes, gameType)
	})
}

func WithRespectedGameTypeOverride(gameType gameTypes.GameType) PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		cfg.RespectedGameTypes = append(cfg.RespectedGameTypes, gameType)
	})
}

func WithCannonKonaGameTypeAdded() PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		cfg.EnableCannonKonaForChall = true
		cfg.AddedGameTypes = append(cfg.AddedGameTypes, gameTypes.CannonKonaGameType)
	})
}

func WithChallengerCannonKonaEnabled() PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		cfg.EnableCannonKonaForChall = true
	})
}

func WithTimeTravelEnabled() PresetOption {
	return presetOptionFn(func(cfg *PresetConfig) {
		cfg.EnableTimeTravel = true
	})
}
