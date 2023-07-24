package flags

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	openum "github.com/ethereum-optimism/optimism/op-service/enum"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const (
	envVarPrefix = "OP_CHALLENGER"
)

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(envVarPrefix, name)
}

var (
	// Required Flags
	L1EthRpcFlag = &cli.StringFlag{
		Name:    "l1-eth-rpc",
		Usage:   "HTTP provider URL for L1.",
		EnvVars: prefixEnvVars("L1_ETH_RPC"),
	}
	DGFAddressFlag = &cli.StringFlag{
		Name:    "game-address",
		Usage:   "Address of the Fault Game contract.",
		EnvVars: prefixEnvVars("GAME_ADDRESS"),
	}
	TraceTypeFlag = &cli.GenericFlag{
		Name:    "trace-type",
		Usage:   "The trace type. Valid options: " + openum.EnumString(config.TraceTypes),
		EnvVars: prefixEnvVars("TRACE_TYPE"),
		Value: func() *config.TraceType {
			out := config.TraceType("") // No default value
			return &out
		}(),
	}
	AgreeWithProposedOutputFlag = &cli.BoolFlag{
		Name:    "agree-with-proposed-output",
		Usage:   "Temporary hardcoded flag if we agree or disagree with the proposed output.",
		EnvVars: prefixEnvVars("AGREE_WITH_PROPOSED_OUTPUT"),
	}
	GameDepthFlag = &cli.IntFlag{
		Name:    "game-depth",
		Usage:   "Depth of the game tree.",
		EnvVars: prefixEnvVars("GAME_DEPTH"),
	}
	// Optional Flags
	AlphabetFlag = &cli.StringFlag{
		Name:    "alphabet",
		Usage:   "Alphabet Trace (temporary)",
		EnvVars: prefixEnvVars("ALPHABET"),
	}
	CannonDatadirFlag = &cli.StringFlag{
		Name:    "cannon-datadir",
		Usage:   "Cannon Data Directory",
		EnvVars: prefixEnvVars("CANNON_DATADIR"),
	}
)

// requiredFlags are checked by [CheckRequired]
var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	DGFAddressFlag,
	TraceTypeFlag,
	AgreeWithProposedOutputFlag,
	GameDepthFlag,
}

// optionalFlags is a list of unchecked cli flags
var optionalFlags = []cli.Flag{
	AlphabetFlag,
	CannonDatadirFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oplog.CLIFlags(envVarPrefix)...)
	optionalFlags = append(optionalFlags, txmgr.CLIFlags(envVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	gameType := config.TraceType(strings.ToLower(ctx.String(TraceTypeFlag.Name)))
	switch gameType {
	case config.TraceTypeCannon:
		if !ctx.IsSet(CannonDatadirFlag.Name) {
			return fmt.Errorf("flag %s is required", "cannon-datadir")
		}
	case config.TraceTypeAlphabet:
		if !ctx.IsSet(AlphabetFlag.Name) {
			return fmt.Errorf("flag %s is required", "alphabet")
		}
	default:
		return fmt.Errorf("invalid trace type. must be one of %v", config.TraceTypes)
	}
	return nil
}

// NewConfigFromCLI parses the Config from the provided flags or environment variables.
func NewConfigFromCLI(ctx *cli.Context) (*config.Config, error) {
	if err := CheckRequired(ctx); err != nil {
		return nil, err
	}
	dgfAddress, err := opservice.ParseAddress(ctx.String(DGFAddressFlag.Name))
	if err != nil {
		return nil, err
	}

	txMgrConfig := txmgr.ReadCLIConfig(ctx)

	traceTypeFlag := config.TraceType(strings.ToLower(ctx.String(TraceTypeFlag.Name)))

	return &config.Config{
		// Required Flags
		L1EthRpc:                ctx.String(L1EthRpcFlag.Name),
		TraceType:               traceTypeFlag,
		GameAddress:             dgfAddress,
		AlphabetTrace:           ctx.String(AlphabetFlag.Name),
		CannonDatadir:           ctx.String(CannonDatadirFlag.Name),
		AgreeWithProposedOutput: ctx.Bool(AgreeWithProposedOutputFlag.Name),
		GameDepth:               ctx.Int(GameDepthFlag.Name),
		TxMgrConfig:             txMgrConfig,
	}, nil
}
