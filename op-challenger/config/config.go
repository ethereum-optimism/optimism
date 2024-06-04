package config

import (
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	ErrMissingTraceType                 = errors.New("no supported trace types specified")
	ErrMissingDatadir                   = errors.New("missing datadir")
	ErrMaxConcurrencyZero               = errors.New("max concurrency must not be 0")
	ErrMissingL2Rpc                     = errors.New("missing L2 rpc url")
	ErrMissingCannonBin                 = errors.New("missing cannon bin")
	ErrMissingCannonServer              = errors.New("missing cannon server")
	ErrMissingCannonAbsolutePreState    = errors.New("missing cannon absolute pre-state")
	ErrCannonAbsolutePreStateAndBaseURL = errors.New("only specify one of cannon absolute pre-state and cannon absolute pre-state base URL")
	ErrMissingL1EthRPC                  = errors.New("missing l1 eth rpc url")
	ErrMissingL1Beacon                  = errors.New("missing l1 beacon url")
	ErrMissingGameFactoryAddress        = errors.New("missing game factory address")
	ErrMissingCannonSnapshotFreq        = errors.New("missing cannon snapshot freq")
	ErrMissingCannonInfoFreq            = errors.New("missing cannon info freq")
	ErrMissingCannonRollupConfig        = errors.New("missing cannon network or rollup config path")
	ErrMissingCannonL2Genesis           = errors.New("missing cannon network or l2 genesis path")
	ErrCannonNetworkAndRollupConfig     = errors.New("only specify one of network or rollup config path")
	ErrCannonNetworkAndL2Genesis        = errors.New("only specify one of network or l2 genesis path")
	ErrCannonNetworkUnknown             = errors.New("unknown cannon network")
	ErrMissingRollupRpc                 = errors.New("missing rollup rpc url")

	ErrMissingAsteriscBin                 = errors.New("missing asterisc bin")
	ErrMissingAsteriscServer              = errors.New("missing asterisc server")
	ErrMissingAsteriscAbsolutePreState    = errors.New("missing asterisc absolute pre-state")
	ErrAsteriscAbsolutePreStateAndBaseURL = errors.New("only specify one of asterisc absolute pre-state and asterisc absolute pre-state base URL")
	ErrMissingAsteriscSnapshotFreq        = errors.New("missing asterisc snapshot freq")
	ErrMissingAsteriscInfoFreq            = errors.New("missing asterisc info freq")
	ErrMissingAsteriscRollupConfig        = errors.New("missing asterisc network or rollup config path")
	ErrMissingAsteriscL2Genesis           = errors.New("missing asterisc network or l2 genesis path")
	ErrAsteriscNetworkAndRollupConfig     = errors.New("only specify one of network or rollup config path")
	ErrAsteriscNetworkAndL2Genesis        = errors.New("only specify one of network or l2 genesis path")
	ErrAsteriscNetworkUnknown             = errors.New("unknown asterisc network")
)

type TraceType string

const (
	TraceTypeAlphabet     TraceType = "alphabet"
	TraceTypeFast         TraceType = "fast"
	TraceTypeCannon       TraceType = "cannon"
	TraceTypeAsterisc     TraceType = "asterisc"
	TraceTypePermissioned TraceType = "permissioned"
)

var TraceTypes = []TraceType{TraceTypeAlphabet, TraceTypeCannon, TraceTypePermissioned, TraceTypeAsterisc, TraceTypeFast}

func (t TraceType) String() string {
	return string(t)
}

// Set implements the Set method required by the [cli.Generic] interface.
func (t *TraceType) Set(value string) error {
	if !ValidTraceType(TraceType(value)) {
		return fmt.Errorf("unknown trace type: %q", value)
	}
	*t = TraceType(value)
	return nil
}

func (t *TraceType) Clone() any {
	cpy := *t
	return &cpy
}

func ValidTraceType(value TraceType) bool {
	for _, t := range TraceTypes {
		if t == value {
			return true
		}
	}
	return false
}

const (
	DefaultPollInterval         = time.Second * 12
	DefaultCannonSnapshotFreq   = uint(1_000_000_000)
	DefaultCannonInfoFreq       = uint(10_000_000)
	DefaultAsteriscSnapshotFreq = uint(1_000_000_000)
	DefaultAsteriscInfoFreq     = uint(10_000_000)
	// DefaultGameWindow is the default maximum time duration in the past
	// that the challenger will look for games to progress.
	// The default value is 28 days. The worst case duration for a game is 16 days
	// (due to clock extension), plus 7 days WETH withdrawal delay leaving a 5 day
	// buffer to monitor games to ensure bonds are claimed.
	DefaultGameWindow   = time.Duration(28 * 24 * time.Hour)
	DefaultMaxPendingTx = 10
)

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	L1EthRpc             string           // L1 RPC Url
	L1Beacon             string           // L1 Beacon API Url
	GameFactoryAddress   common.Address   // Address of the dispute game factory
	GameAllowlist        []common.Address // Allowlist of fault game addresses
	GameWindow           time.Duration    // Maximum time duration to look for games to progress
	Datadir              string           // Data Directory
	MaxConcurrency       uint             // Maximum number of threads to use when progressing games
	PollInterval         time.Duration    // Polling interval for latest-block subscription when using an HTTP RPC provider
	AllowInvalidPrestate bool             // Whether to allow responding to games where the prestate does not match

	AdditionalBondClaimants []common.Address // List of addresses to claim bonds for in addition to the tx manager sender

	SelectiveClaimResolution bool // Whether to only resolve claims for the claimants in AdditionalBondClaimants union [TxSender.From()]

	TraceTypes []TraceType // Type of traces supported

	RollupRpc string // L2 Rollup RPC Url

	L2Rpc string // L2 RPC Url

	// Specific to the cannon trace provider
	CannonBin                     string   // Path to the cannon executable to run when generating trace data
	CannonServer                  string   // Path to the op-program executable that provides the pre-image oracle server
	CannonAbsolutePreState        string   // File to load the absolute pre-state for Cannon traces from
	CannonAbsolutePreStateBaseURL *url.URL // Base URL to retrieve absolute pre-states for Cannon traces from
	CannonNetwork                 string
	CannonRollupConfigPath        string
	CannonL2GenesisPath           string
	CannonSnapshotFreq            uint // Frequency of snapshots to create when executing cannon (in VM instructions)
	CannonInfoFreq                uint // Frequency of cannon progress log messages (in VM instructions)

	// Specific to the asterisc trace provider
	AsteriscBin                     string   // Path to the asterisc executable to run when generating trace data
	AsteriscServer                  string   // Path to the op-program executable that provides the pre-image oracle server
	AsteriscAbsolutePreState        string   // File to load the absolute pre-state for Asterisc traces from
	AsteriscAbsolutePreStateBaseURL *url.URL // Base URL to retrieve absolute pre-states for Asterisc traces from
	AsteriscNetwork                 string
	AsteriscRollupConfigPath        string
	AsteriscL2GenesisPath           string
	AsteriscSnapshotFreq            uint // Frequency of snapshots to create when executing asterisc (in VM instructions)
	AsteriscInfoFreq                uint // Frequency of asterisc progress log messages (in VM instructions)

	MaxPendingTx uint64 // Maximum number of pending transactions (0 == no limit)

	TxMgrConfig   txmgr.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func NewConfig(
	gameFactoryAddress common.Address,
	l1EthRpc string,
	l1BeaconApi string,
	l2RollupRpc string,
	l2EthRpc string,
	datadir string,
	supportedTraceTypes ...TraceType,
) Config {
	return Config{
		L1EthRpc:           l1EthRpc,
		L1Beacon:           l1BeaconApi,
		RollupRpc:          l2RollupRpc,
		L2Rpc:              l2EthRpc,
		GameFactoryAddress: gameFactoryAddress,
		MaxConcurrency:     uint(runtime.NumCPU()),
		PollInterval:       DefaultPollInterval,

		TraceTypes: supportedTraceTypes,

		MaxPendingTx: DefaultMaxPendingTx,

		TxMgrConfig:   txmgr.NewCLIConfig(l1EthRpc, txmgr.DefaultChallengerFlagValues),
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),

		Datadir: datadir,

		CannonSnapshotFreq:   DefaultCannonSnapshotFreq,
		CannonInfoFreq:       DefaultCannonInfoFreq,
		AsteriscSnapshotFreq: DefaultAsteriscSnapshotFreq,
		AsteriscInfoFreq:     DefaultAsteriscInfoFreq,
		GameWindow:           DefaultGameWindow,
	}
}

func (c Config) TraceTypeEnabled(t TraceType) bool {
	return slices.Contains(c.TraceTypes, t)
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if c.L1Beacon == "" {
		return ErrMissingL1Beacon
	}
	if c.RollupRpc == "" {
		return ErrMissingRollupRpc
	}
	if c.L2Rpc == "" {
		return ErrMissingL2Rpc
	}
	if c.GameFactoryAddress == (common.Address{}) {
		return ErrMissingGameFactoryAddress
	}
	if len(c.TraceTypes) == 0 {
		return ErrMissingTraceType
	}
	if c.Datadir == "" {
		return ErrMissingDatadir
	}
	if c.MaxConcurrency == 0 {
		return ErrMaxConcurrencyZero
	}
	if c.TraceTypeEnabled(TraceTypeCannon) || c.TraceTypeEnabled(TraceTypePermissioned) {
		if c.CannonBin == "" {
			return ErrMissingCannonBin
		}
		if c.CannonServer == "" {
			return ErrMissingCannonServer
		}
		if c.CannonNetwork == "" {
			if c.CannonRollupConfigPath == "" {
				return ErrMissingCannonRollupConfig
			}
			if c.CannonL2GenesisPath == "" {
				return ErrMissingCannonL2Genesis
			}
		} else {
			if c.CannonRollupConfigPath != "" {
				return ErrCannonNetworkAndRollupConfig
			}
			if c.CannonL2GenesisPath != "" {
				return ErrCannonNetworkAndL2Genesis
			}
			if ch := chaincfg.ChainByName(c.CannonNetwork); ch == nil {
				return fmt.Errorf("%w: %v", ErrCannonNetworkUnknown, c.CannonNetwork)
			}
		}
		if c.CannonAbsolutePreState == "" && c.CannonAbsolutePreStateBaseURL == nil {
			return ErrMissingCannonAbsolutePreState
		}
		if c.CannonAbsolutePreState != "" && c.CannonAbsolutePreStateBaseURL != nil {
			return ErrCannonAbsolutePreStateAndBaseURL
		}
		if c.CannonSnapshotFreq == 0 {
			return ErrMissingCannonSnapshotFreq
		}
		if c.CannonInfoFreq == 0 {
			return ErrMissingCannonInfoFreq
		}
	}
	if c.TraceTypeEnabled(TraceTypeAsterisc) {
		if c.AsteriscBin == "" {
			return ErrMissingAsteriscBin
		}
		if c.AsteriscServer == "" {
			return ErrMissingAsteriscServer
		}
		if c.AsteriscNetwork == "" {
			if c.AsteriscRollupConfigPath == "" {
				return ErrMissingAsteriscRollupConfig
			}
			if c.AsteriscL2GenesisPath == "" {
				return ErrMissingAsteriscL2Genesis
			}
		} else {
			if c.AsteriscRollupConfigPath != "" {
				return ErrAsteriscNetworkAndRollupConfig
			}
			if c.AsteriscL2GenesisPath != "" {
				return ErrAsteriscNetworkAndL2Genesis
			}
			if ch := chaincfg.ChainByName(c.AsteriscNetwork); ch == nil {
				return fmt.Errorf("%w: %v", ErrAsteriscNetworkUnknown, c.AsteriscNetwork)
			}
		}
		if c.AsteriscAbsolutePreState == "" && c.AsteriscAbsolutePreStateBaseURL == nil {
			return ErrMissingAsteriscAbsolutePreState
		}
		if c.AsteriscAbsolutePreState != "" && c.AsteriscAbsolutePreStateBaseURL != nil {
			return ErrAsteriscAbsolutePreStateAndBaseURL
		}
		if c.AsteriscSnapshotFreq == 0 {
			return ErrMissingAsteriscSnapshotFreq
		}
		if c.AsteriscInfoFreq == 0 {
			return ErrMissingAsteriscInfoFreq
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
