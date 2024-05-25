package config

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	ErrMissingTraceType              = errors.New("no supported trace types specified")
	ErrMissingDatadir                = errors.New("missing datadir")
	ErrMaxConcurrencyZero            = errors.New("max concurrency must not be 0")
	ErrMissingCannonL2               = errors.New("missing cannon L2")
	ErrMissingCannonBin              = errors.New("missing cannon bin")
	ErrMissingCannonServer           = errors.New("missing cannon server")
	ErrMissingCannonAbsolutePreState = errors.New("missing cannon absolute pre-state")
	ErrMissingAlphabetTrace          = errors.New("missing alphabet trace")
	ErrMissingL1EthRPC               = errors.New("missing l1 eth rpc url")
	ErrMissingGameFactoryAddress     = errors.New("missing game factory address")
	ErrMissingCannonSnapshotFreq     = errors.New("missing cannon snapshot freq")
	ErrMissingCannonInfoFreq         = errors.New("missing cannon info freq")
	ErrMissingCannonRollupConfig     = errors.New("missing cannon network or rollup config path")
	ErrMissingCannonL2Genesis        = errors.New("missing cannon network or l2 genesis path")
	ErrCannonNetworkAndRollupConfig  = errors.New("only specify one of network or rollup config path")
	ErrCannonNetworkAndL2Genesis     = errors.New("only specify one of network or l2 genesis path")
	ErrCannonNetworkUnknown          = errors.New("unknown cannon network")
	ErrMissingRollupRpc              = errors.New("missing rollup rpc url")
)

type TraceType string

const (
	TraceTypeAlphabet       TraceType = "alphabet"
	TraceTypeCannon         TraceType = "cannon"
	TraceTypeOutputCannon   TraceType = "output_cannon"
	TraceTypeOutputAlphabet TraceType = "output_alphabet"

	// Mainnet games
	CannonFaultGameID = 0

	// Devnet games
	AlphabetFaultGameID = 255
)

var TraceTypes = []TraceType{TraceTypeAlphabet, TraceTypeCannon, TraceTypeOutputCannon, TraceTypeOutputAlphabet}

// GameIdToString maps game IDs to their string representation.
var GameIdToString = map[uint8]string{
	CannonFaultGameID:   "Cannon",
	AlphabetFaultGameID: "Alphabet",
}

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
	DefaultPollInterval       = time.Second * 12
	DefaultCannonSnapshotFreq = uint(1_000_000_000)
	DefaultCannonInfoFreq     = uint(10_000_000)
	// DefaultGameWindow is the default maximum time duration in the past
	// that the challenger will look for games to progress.
	// The default value is 11 days, which is a 4 day resolution buffer
	// plus the 7 day game finalization window.
	DefaultGameWindow = time.Duration(11 * 24 * time.Hour)
)

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	L1EthRpc           string           // L1 RPC Url
	GameFactoryAddress common.Address   // Address of the dispute game factory
	GameAllowlist      []common.Address // Allowlist of fault game addresses
	GameWindow         time.Duration    // Maximum time duration to look for games to progress
	Datadir            string           // Data Directory
	MaxConcurrency     uint             // Maximum number of threads to use when progressing games
	PollInterval       time.Duration    // Polling interval for latest-block subscription when using an HTTP RPC provider

	TraceTypes []TraceType // Type of traces supported

	// Specific to the alphabet trace provider
	AlphabetTrace string // String for the AlphabetTraceProvider

	// Specific to the output cannon trace type
	RollupRpc string

	// Specific to the cannon trace provider
	CannonBin              string // Path to the cannon executable to run when generating trace data
	CannonServer           string // Path to the op-program executable that provides the pre-image oracle server
	CannonAbsolutePreState string // File to load the absolute pre-state for Cannon traces from
	CannonNetwork          string
	CannonRollupConfigPath string
	CannonL2GenesisPath    string
	CannonL2               string // L2 RPC Url
	CannonSnapshotFreq     uint   // Frequency of snapshots to create when executing cannon (in VM instructions)
	CannonInfoFreq         uint   // Frequency of cannon progress log messages (in VM instructions)

	TxMgrConfig   txmgr.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func NewConfig(
	gameFactoryAddress common.Address,
	l1EthRpc string,
	datadir string,
	supportedTraceTypes ...TraceType,
) Config {
	return Config{
		L1EthRpc:           l1EthRpc,
		GameFactoryAddress: gameFactoryAddress,
		MaxConcurrency:     uint(runtime.NumCPU()),
		PollInterval:       DefaultPollInterval,

		TraceTypes: supportedTraceTypes,

		TxMgrConfig:   txmgr.NewCLIConfig(l1EthRpc, txmgr.DefaultChallengerFlagValues),
		MetricsConfig: opmetrics.DefaultCLIConfig(),
		PprofConfig:   oppprof.DefaultCLIConfig(),

		Datadir: datadir,

		CannonSnapshotFreq: DefaultCannonSnapshotFreq,
		CannonInfoFreq:     DefaultCannonInfoFreq,
		GameWindow:         DefaultGameWindow,
	}
}

func (c Config) TraceTypeEnabled(t TraceType) bool {
	return slices.Contains(c.TraceTypes, t)
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
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
	if c.TraceTypeEnabled(TraceTypeOutputCannon) || c.TraceTypeEnabled(TraceTypeOutputAlphabet) {
		if c.RollupRpc == "" {
			return ErrMissingRollupRpc
		}
	}
	if c.TraceTypeEnabled(TraceTypeCannon) || c.TraceTypeEnabled(TraceTypeOutputCannon) {
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
		if c.CannonAbsolutePreState == "" {
			return ErrMissingCannonAbsolutePreState
		}
		if c.CannonL2 == "" {
			return ErrMissingCannonL2
		}
		if c.CannonSnapshotFreq == 0 {
			return ErrMissingCannonSnapshotFreq
		}
		if c.CannonInfoFreq == 0 {
			return ErrMissingCannonInfoFreq
		}
	}
	if c.TraceTypeEnabled(TraceTypeAlphabet) && c.AlphabetTrace == "" {
		return ErrMissingAlphabetTrace
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
