package flags

import (
	"fmt"

	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const envVarPrefix = "OP_CHALLENGER"

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
	AlphabetFlag = &cli.StringFlag{
		Name:    "alphabet",
		Usage:   "Alphabet Trace (temporary)",
		EnvVars: prefixEnvVars("ALPHABET"),
	}
	AgreeWithProposedOutputFlag = &cli.BoolFlag{
		Name:    "agree-with-proposed-output",
		Usage:   "Temporary hardcoded flag if we agree or disagree with the proposed output.",
		EnvVars: prefixEnvVars("AGREE_WITH_PROPOSED_OUTPUT"),
	}
	// Optional Flags
)

// requiredFlags are checked by [CheckRequired]
var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	DGFAddressFlag,
	AlphabetFlag,
	AgreeWithProposedOutputFlag,
}

// optionalFlags is a list of unchecked cli flags
var optionalFlags = []cli.Flag{}

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
	return nil
}
