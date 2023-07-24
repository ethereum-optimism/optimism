package config

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrMissingTraceType     = errors.New("missing trace type")
	ErrMissingCannonDatadir = errors.New("missing cannon datadir")
	ErrMissingAlphabetTrace = errors.New("missing alphabet trace")
	ErrMissingL1EthRPC      = errors.New("missing l1 eth rpc url")
	ErrMissingGameAddress   = errors.New("missing game address")
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

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	L1EthRpc                string         // L1 RPC Url
	GameAddress             common.Address // Address of the fault game
	AgreeWithProposedOutput bool           // Temporary config if we agree or disagree with the posted output
	GameDepth               int            // Depth of the game tree

	TraceType     TraceType // Type of trace
	AlphabetTrace string    // String for the AlphabetTraceProvider
	CannonDatadir string    // Cannon Data Directory for the CannonTraceProvider

	TxMgrConfig txmgr.CLIConfig
}

func NewConfig(
	l1EthRpc string,
	gameAddress common.Address,
	traceType TraceType,
	alphabetTrace string,
	cannonDatadir string,
	agreeWithProposedOutput bool,
	gameDepth int,
) Config {
	return Config{
		L1EthRpc:    l1EthRpc,
		GameAddress: gameAddress,

		AgreeWithProposedOutput: agreeWithProposedOutput,
		GameDepth:               gameDepth,

		TraceType:     traceType,
		AlphabetTrace: alphabetTrace,
		CannonDatadir: cannonDatadir,

		TxMgrConfig: txmgr.NewCLIConfig(l1EthRpc),
	}
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if c.GameAddress == (common.Address{}) {
		return ErrMissingGameAddress
	}
	if c.TraceType == "" {
		return ErrMissingTraceType
	}
	if c.TraceType == TraceTypeCannon && c.CannonDatadir == "" {
		return ErrMissingCannonDatadir
	}
	if c.TraceType == TraceTypeAlphabet && c.AlphabetTrace == "" {
		return ErrMissingAlphabetTrace
	}
	if err := c.TxMgrConfig.Check(); err != nil {
		return err
	}
	return nil
}
