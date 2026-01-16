package config

import (
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrMissingGameType                   = errors.New("no supported game types specified")
	ErrMissingDatadir                    = errors.New("missing datadir")
	ErrMaxConcurrencyZero                = errors.New("max concurrency must not be 0")
	ErrMissingL2Rpc                      = errors.New("missing L2 rpc url")
	ErrMissingCannonAbsolutePreState     = errors.New("missing cannon absolute pre-state")
	ErrMissingL1EthRPC                   = errors.New("missing l1 eth rpc url")
	ErrMissingL1RPCKind                  = errors.New("missing l1 eth rpc kind")
	ErrMissingL1Beacon                   = errors.New("missing l1 beacon url")
	ErrMissingGameFactoryAddress         = errors.New("missing game factory address")
	ErrMissingCannonSnapshotFreq         = errors.New("missing cannon snapshot freq")
	ErrMissingCannonInfoFreq             = errors.New("missing cannon info freq")
	ErrMissingCannonKonaAbsolutePreState = errors.New("missing cannon kona absolute pre-state")
	ErrMissingCannonKonaSnapshotFreq     = errors.New("missing cannon kona snapshot freq")
	ErrMissingCannonKonaInfoFreq         = errors.New("missing cannon kona info freq")
	ErrMissingDepsetConfig               = errors.New("missing network or depset config path")

	ErrMissingRollupRpc = errors.New("missing rollup rpc url")
	ErrMissingSuperRpc  = errors.New("missing super rpc url")
)

const (
	DefaultPollInterval       = time.Second * 12
	DefaultCannonSnapshotFreq = uint(1_000_000_000)
	DefaultCannonInfoFreq     = uint(10_000_000)
	// DefaultGameWindow is the default maximum time duration in the past
	// that the challenger will look for games to progress.
	// The default value is 28 days. The worst case duration for a game is 16 days
	// (due to clock extension), plus 7 days WETH withdrawal delay leaving a 5 day
	// buffer to monitor games to ensure bonds are claimed.
	DefaultGameWindow         = 28 * 24 * time.Hour
	DefaultMaxPendingTx       = 10
	DefaultResponseDelay      = 0 // No delay by default
	DefaultResponseDelayAfter = 0 // Apply delay from first response by default
)

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	L1EthRpc             string                  // L1 RPC Url
	L1RPCKind            sources.RPCProviderKind // L1 RPC kind
	L1Beacon             string                  // L1 Beacon API Url
	GameFactoryAddress   common.Address          // Address of the dispute game factory
	GameAllowlist        []common.Address        // Allowlist of fault game addresses
	GameWindow           time.Duration           // Maximum time duration to look for games to progress
	Datadir              string                  // Data Directory
	MaxConcurrency       uint                    // Maximum number of threads to use when progressing games
	PollInterval         time.Duration           // Polling interval for latest-block subscription when using an HTTP RPC provider
	AllowInvalidPrestate bool                    // Whether to allow responding to games where the prestate does not match
	MinUpdateInterval    time.Duration           // Minimum duration the L1 head block time must advance before scheduling a new update cycle

	AdditionalBondClaimants []common.Address // List of addresses to claim bonds for in addition to the tx manager sender

	SelectiveClaimResolution bool // Whether to only resolve claims for the claimants in AdditionalBondClaimants union [TxSender.From()]

	GameTypes []gameTypes.GameType // Type of games supported

	RollupRpc    string   // L2 Rollup RPC Url
	SuperRPC     string   // L2 RPC URL for super roots
	UseSuperNode bool     // Temporary: True to use op-supernode APIs, false for op-supervisor APIs
	L2Rpcs       []string // L2 RPC Url

	// Specific to the cannon trace provider
	Cannon                            vm.Config
	CannonAbsolutePreState            string   // File to load the absolute pre-state for Cannon traces from
	CannonAbsolutePreStateBaseURL     *url.URL // Base URL to retrieve absolute pre-states for Cannon traces from
	CannonKona                        vm.Config
	CannonKonaAbsolutePreState        string   // File to load the absolute pre-state for CannonKona traces from
	CannonKonaAbsolutePreStateBaseURL *url.URL // Base URL to retrieve absolute pre-states for CannonKona traces from

	MaxPendingTx uint64 // Maximum number of pending transactions (0 == no limit)

	TxMgrConfig   txmgr.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig

	ResponseDelay time.Duration /* Delay before responding to each game action to slow down game progression.
	   Note: set with caution, since the challenger can end up using more resources if it has to wait to respond
	   to an attacker generating many claims. Consider using the additional ResponseDelayAfter config option.
	   Also note that the delay is only applied when:
	   	1) delaying will not lead to a timeout of the game,
	   	2) the challenger is not in a clock extension period and
	   	3) delaying will not lead to the challenger having to respond inside of a clock extension period
	       (thus ensuring that the challenger always has enough remaining time to respond to the game action). */
	ResponseDelayAfter uint64 /* Number of responses after which to start applying the delay.
	   Set to 0 to apply delay from the first response, 1 to skip the first response, etc.
	   Note: the delay is only applied from the next round after which this `responseDelayAfter` value
	   is surpassed (not from the exact response after which its surpassed, but from the next round). */
}

func NewInteropConfig(
	gameFactoryAddress common.Address,
	l1EthRpc string,
	l1BeaconApi string,
	superRpc string,
	l2Rpcs []string,
	datadir string,
	supportedGameTypes ...gameTypes.GameType,
) Config {
	return Config{
		L1EthRpc:           l1EthRpc,
		L1RPCKind:          sources.RPCKindStandard,
		L1Beacon:           l1BeaconApi,
		SuperRPC:           superRpc,
		L2Rpcs:             l2Rpcs,
		GameFactoryAddress: gameFactoryAddress,
		MaxConcurrency:     uint(runtime.NumCPU()),
		PollInterval:       DefaultPollInterval,

		GameTypes: supportedGameTypes,

		MaxPendingTx: DefaultMaxPendingTx,

		TxMgrConfig:   txmgr.NewCLIConfig(l1EthRpc, txmgr.DefaultChallengerFlagValues),
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),

		Datadir: datadir,

		Cannon: vm.Config{
			VmType:          gameTypes.CannonGameType,
			L1:              l1EthRpc,
			L1Beacon:        l1BeaconApi,
			L2s:             l2Rpcs,
			SnapshotFreq:    DefaultCannonSnapshotFreq,
			InfoFreq:        DefaultCannonInfoFreq,
			DebugInfo:       true,
			BinarySnapshots: true,
		},
		CannonKona: vm.Config{
			VmType:          gameTypes.CannonKonaGameType,
			L1:              l1EthRpc,
			L1Beacon:        l1BeaconApi,
			L2s:             l2Rpcs,
			SnapshotFreq:    DefaultCannonSnapshotFreq,
			InfoFreq:        DefaultCannonInfoFreq,
			DebugInfo:       true,
			BinarySnapshots: true,
		},
		GameWindow: DefaultGameWindow,
	}
}

func NewConfig(
	gameFactoryAddress common.Address,
	l1EthRpc string,
	l1BeaconApi string,
	l2RollupRpc string,
	l2EthRpc string,
	datadir string,
	supportedGameTypes ...gameTypes.GameType,
) Config {
	return Config{
		L1EthRpc:           l1EthRpc,
		L1RPCKind:          sources.RPCKindStandard,
		L1Beacon:           l1BeaconApi,
		RollupRpc:          l2RollupRpc,
		L2Rpcs:             []string{l2EthRpc},
		GameFactoryAddress: gameFactoryAddress,
		MaxConcurrency:     uint(runtime.NumCPU()),
		PollInterval:       DefaultPollInterval,

		GameTypes: supportedGameTypes,

		MaxPendingTx: DefaultMaxPendingTx,

		TxMgrConfig:   txmgr.NewCLIConfig(l1EthRpc, txmgr.DefaultChallengerFlagValues),
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),

		Datadir: datadir,

		Cannon: vm.Config{
			VmType:          gameTypes.CannonGameType,
			L1:              l1EthRpc,
			L1Beacon:        l1BeaconApi,
			L2s:             []string{l2EthRpc},
			SnapshotFreq:    DefaultCannonSnapshotFreq,
			InfoFreq:        DefaultCannonInfoFreq,
			DebugInfo:       true,
			BinarySnapshots: true,
		},
		CannonKona: vm.Config{
			VmType:          gameTypes.CannonKonaGameType,
			L1:              l1EthRpc,
			L1Beacon:        l1BeaconApi,
			L2s:             []string{l2EthRpc},
			SnapshotFreq:    DefaultCannonSnapshotFreq,
			InfoFreq:        DefaultCannonInfoFreq,
			DebugInfo:       true,
			BinarySnapshots: true,
		},
		GameWindow: DefaultGameWindow,
	}
}

func (c Config) GameTypeEnabled(t gameTypes.GameType) bool {
	return slices.Contains(c.GameTypes, t)
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if c.L1RPCKind == "" {
		return ErrMissingL1RPCKind
	}
	if c.L1Beacon == "" {
		return ErrMissingL1Beacon
	}
	if len(c.L2Rpcs) == 0 {
		return ErrMissingL2Rpc
	}
	if c.GameFactoryAddress == (common.Address{}) {
		return ErrMissingGameFactoryAddress
	}
	if len(c.GameTypes) == 0 {
		return ErrMissingGameType
	}
	if c.Datadir == "" {
		return ErrMissingDatadir
	}
	if c.MaxConcurrency == 0 {
		return ErrMaxConcurrencyZero
	}
	if c.GameTypeEnabled(gameTypes.SuperCannonGameType) || c.GameTypeEnabled(gameTypes.SuperPermissionedGameType) {
		if c.SuperRPC == "" {
			return ErrMissingSuperRpc
		}

		if len(c.Cannon.Networks) == 0 && c.Cannon.DepsetConfigPath == "" {
			return ErrMissingDepsetConfig
		}
		if err := c.validateBaseCannonOptions(); err != nil {
			return err
		}
	}
	if c.GameTypeEnabled(gameTypes.CannonGameType) || c.GameTypeEnabled(gameTypes.PermissionedGameType) {
		if c.RollupRpc == "" {
			return ErrMissingRollupRpc
		}
		if err := c.validateBaseCannonOptions(); err != nil {
			return err
		}
	}
	if c.GameTypeEnabled(gameTypes.SuperCannonKonaGameType) {
		if c.SuperRPC == "" {
			return ErrMissingSuperRpc
		}

		if len(c.CannonKona.Networks) == 0 && c.CannonKona.DepsetConfigPath == "" {
			return ErrMissingDepsetConfig
		}
		if err := c.validateBaseCannonKonaOptions(); err != nil {
			return err
		}
	}
	if c.GameTypeEnabled(gameTypes.CannonKonaGameType) {
		if c.RollupRpc == "" {
			return ErrMissingRollupRpc
		}
		if err := c.validateBaseCannonKonaOptions(); err != nil {
			return err
		}
	}
	if c.GameTypeEnabled(gameTypes.OptimisticZKGameType) {
		if c.RollupRpc == "" {
			return ErrMissingRollupRpc
		}
	}
	if c.GameTypeEnabled(gameTypes.AlphabetGameType) || c.GameTypeEnabled(gameTypes.FastGameType) {
		if c.RollupRpc == "" {
			return ErrMissingRollupRpc
		}
	}
	if err := c.TxMgrConfig.Check(); err != nil {
		return err
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	if err := c.PprofConfig.Check(); err != nil {
		return err
	}
	return nil
}

func (c Config) validateBaseCannonOptions() error {
	if err := c.Cannon.Check(); err != nil {
		return fmt.Errorf("cannon: %w", err)
	}
	if c.CannonAbsolutePreState == "" && c.CannonAbsolutePreStateBaseURL == nil {
		return ErrMissingCannonAbsolutePreState
	}
	if c.Cannon.SnapshotFreq == 0 {
		return ErrMissingCannonSnapshotFreq
	}
	if c.Cannon.InfoFreq == 0 {
		return ErrMissingCannonInfoFreq
	}
	return nil
}

func (c Config) validateBaseCannonKonaOptions() error {
	if err := c.CannonKona.Check(); err != nil {
		return fmt.Errorf("cannon kona: %w", err)
	}
	if c.CannonKonaAbsolutePreState == "" && c.CannonKonaAbsolutePreStateBaseURL == nil {
		return ErrMissingCannonKonaAbsolutePreState
	}
	if c.CannonKona.SnapshotFreq == 0 {
		return ErrMissingCannonKonaSnapshotFreq
	}
	if c.CannonKona.InfoFreq == 0 {
		return ErrMissingCannonKonaInfoFreq
	}
	return nil
}
