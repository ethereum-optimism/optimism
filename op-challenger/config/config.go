package config

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrMissingTraceType              = errors.New("missing trace type")
	ErrMissingCannonDatadir          = errors.New("missing cannon datadir")
	ErrMissingCannonL2               = errors.New("missing cannon L2")
	ErrMissingCannonBin              = errors.New("missing cannon bin")
	ErrMissingCannonServer           = errors.New("missing cannon server")
	ErrMissingCannonAbsolutePreState = errors.New("missing cannon absolute pre-state")
	ErrMissingAlphabetTrace          = errors.New("missing alphabet trace")
	ErrMissingL1EthRPC               = errors.New("missing l1 eth rpc url")
	ErrMissingGameFactoryAddress     = errors.New("missing game factory address")
	ErrMissingCannonSnapshotFreq     = errors.New("missing cannon snapshot freq")
	ErrMissingCannonRollupConfig     = errors.New("missing cannon network or rollup config path")
	ErrMissingCannonL2Genesis        = errors.New("missing cannon network or l2 genesis path")
	ErrCannonNetworkAndRollupConfig  = errors.New("only specify one of network or rollup config path")
	ErrCannonNetworkAndL2Genesis     = errors.New("only specify one of network or l2 genesis path")
	ErrCannonNetworkUnknown          = errors.New("unknown cannon network")
)

type TraceType string

const (
	TraceTypeAlphabet TraceType = "alphabet"
	TraceTypeCannon   TraceType = "cannon"
)

var TraceTypes = []TraceType{TraceTypeAlphabet, TraceTypeCannon}

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

func ValidTraceType(value TraceType) bool {
	for _, t := range TraceTypes {
		if t == value {
			return true
		}
	}
	return false
}

const DefaultCannonSnapshotFreq = uint(1_000_000_000)

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	L1EthRpc                string         // L1 RPC Url
	GameFactoryAddress      common.Address // Address of the dispute game factory
	GameAddress             common.Address // Address of the fault game
	AgreeWithProposedOutput bool           // Temporary config if we agree or disagree with the posted output

	TraceType TraceType // Type of trace

	// Specific to the alphabet trace provider
	AlphabetTrace string // String for the AlphabetTraceProvider

	// Specific to the cannon trace provider
	CannonBin              string // Path to the cannon executable to run when generating trace data
	CannonServer           string // Path to the op-program executable that provides the pre-image oracle server
	CannonAbsolutePreState string // File to load the absolute pre-state for Cannon traces from
	CannonNetwork          string
	CannonRollupConfigPath string
	CannonL2GenesisPath    string
	CannonDatadir          string // Cannon Data Directory
	CannonL2               string // L2 RPC Url
	CannonSnapshotFreq     uint   // Frequency of snapshots to create when executing cannon (in VM instructions)

	TxMgrConfig txmgr.CLIConfig
}

func NewConfig(
	gameFactoryAddress common.Address,
	l1EthRpc string,
	traceType TraceType,
	agreeWithProposedOutput bool,
) Config {
	return Config{
		L1EthRpc:           l1EthRpc,
		GameFactoryAddress: gameFactoryAddress,

		AgreeWithProposedOutput: agreeWithProposedOutput,

		TraceType: traceType,

		TxMgrConfig: txmgr.NewCLIConfig(l1EthRpc),

		CannonSnapshotFreq: DefaultCannonSnapshotFreq,
	}
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if c.GameFactoryAddress == (common.Address{}) {
		return ErrMissingGameFactoryAddress
	}
	if c.TraceType == "" {
		return ErrMissingTraceType
	}
	if c.TraceType == TraceTypeCannon {
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
			if _, ok := chaincfg.NetworksByName[c.CannonNetwork]; !ok {
				return fmt.Errorf("%w: %v", ErrCannonNetworkUnknown, c.CannonNetwork)
			}
		}
		if c.CannonAbsolutePreState == "" {
			return ErrMissingCannonAbsolutePreState
		}
		if c.CannonDatadir == "" {
			return ErrMissingCannonDatadir
		}
		if c.CannonL2 == "" {
			return ErrMissingCannonL2
		}
		if c.CannonSnapshotFreq == 0 {
			return ErrMissingCannonSnapshotFreq
		}
	}
	if c.TraceType == TraceTypeAlphabet && c.AlphabetTrace == "" {
		return ErrMissingAlphabetTrace
	}
	if err := c.TxMgrConfig.Check(); err != nil {
		return err
	}
	return nil
}
