package deployer

import (
	"os"

	op_service "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

const (
	EnvVarPrefix       = "DEPLOYER"
	L1RPCURLFlagName   = "l1-rpc-url"
	L1ChainIDFlagName  = "l1-chain-id"
	WorkdirFlagName    = "workdir"
	OutdirFlagName     = "outdir"
	DevFlagName        = "dev"
	PrivateKeyFlagName = "private-key"
)

var (
	L1RPCURLFlag = &cli.StringFlag{
		Name: L1RPCURLFlagName,
		Usage: "RPC URL for the L1 chain. Can be set to 'genesis' for deployments " +
			"that will be deployed at the launch of the L1.",
		EnvVars: []string{
			"L1_RPC_URL",
		},
	}
	L1ChainIDFlag = &cli.Uint64Flag{
		Name:    L1ChainIDFlagName,
		Usage:   "Chain ID of the L1 chain.",
		EnvVars: prefixEnvVar("L1_CHAIN_ID"),
		Value:   900,
	}
	WorkdirFlag = &cli.StringFlag{
		Name:    WorkdirFlagName,
		Usage:   "Directory storing intent and stage. Defaults to the current directory.",
		EnvVars: prefixEnvVar("WORKDIR"),
		Value:   cwd(),
		Aliases: []string{
			OutdirFlagName,
		},
	}
	DevFlag = &cli.BoolFlag{
		Name: DevFlagName,
		Usage: "Use development mode. This will use the development mnemonic to own the chain" +
			" and fund dev accounts.",
		EnvVars: prefixEnvVar("DEV"),
	}

	PrivateKeyFlag = &cli.StringFlag{
		Name:    PrivateKeyFlagName,
		Usage:   "Private key of the deployer account.",
		EnvVars: prefixEnvVar("PRIVATE_KEY"),
	}
)

var GlobalFlags = append([]cli.Flag{}, oplog.CLIFlags(EnvVarPrefix)...)

var InitFlags = []cli.Flag{
	L1ChainIDFlag,
	WorkdirFlag,
	DevFlag,
}

var ApplyFlags = []cli.Flag{
	L1RPCURLFlag,
	WorkdirFlag,
	PrivateKeyFlag,
}

func prefixEnvVar(name string) []string {
	return op_service.PrefixEnvVar(EnvVarPrefix, name)
}

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}
