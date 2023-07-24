package config

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	ErrMissingTraceType     = errors.New("missing trace type")
	ErrMissingCannonDatadir = errors.New("missing cannon datadir")
	ErrMissingAlphabetTrace = errors.New("missing alphabet trace")
	ErrMissingL1EthRPC      = errors.New("missing l1 eth rpc url")
	ErrMissingGameAddress   = errors.New("missing game address")
)

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	L1EthRpc                string         // L1 RPC Url
	GameAddress             common.Address // Address of the fault game
	AgreeWithProposedOutput bool           // Temporary config if we agree or disagree with the posted output
	GameDepth               int            // Depth of the game tree

	TraceType     flags.TraceType // Type of trace
	AlphabetTrace string          // String for the AlphabetTraceProvider
	CannonDatadir string          // Cannon Data Directory for the CannonTraceProvider

	TxMgrConfig txmgr.CLIConfig
}

func NewConfig(
	l1EthRpc string,
	gameAddress common.Address,
	traceType flags.TraceType,
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
	if c.TraceType == flags.TraceTypeCannon && c.CannonDatadir == "" {
		return ErrMissingCannonDatadir
	}
	if c.TraceType == flags.TraceTypeAlphabet && c.AlphabetTrace == "" {
		return ErrMissingAlphabetTrace
	}
	if err := c.TxMgrConfig.Check(); err != nil {
		return err
	}
	return nil
}

// NewConfigFromCLI parses the Config from the provided flags or environment variables.
func NewConfigFromCLI(ctx *cli.Context) (*Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}
	dgfAddress, err := opservice.ParseAddress(ctx.String(flags.DGFAddressFlag.Name))
	if err != nil {
		return nil, err
	}

	txMgrConfig := txmgr.ReadCLIConfig(ctx)

	traceTypeFlag := flags.TraceType(strings.ToLower(ctx.String(flags.TraceTypeFlag.Name)))

	return &Config{
		// Required Flags
		L1EthRpc:                ctx.String(flags.L1EthRpcFlag.Name),
		TraceType:               traceTypeFlag,
		GameAddress:             dgfAddress,
		AlphabetTrace:           ctx.String(flags.AlphabetFlag.Name),
		CannonDatadir:           ctx.String(flags.CannonDatadirFlag.Name),
		AgreeWithProposedOutput: ctx.Bool(flags.AgreeWithProposedOutputFlag.Name),
		GameDepth:               ctx.Int(flags.GameDepthFlag.Name),
		TxMgrConfig:             txMgrConfig,
	}, nil
}
