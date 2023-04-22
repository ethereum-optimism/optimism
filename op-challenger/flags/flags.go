package flags

import (
	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const envVarPrefix = "OP_CHALLENGER"

var (
	// Required Flags
	L1EthRpcFlag = cli.StringFlag{
		Name:     "l1-eth-rpc",
		Usage:    "HTTP provider URL for L1.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "L1_ETH_RPC"),
	}
	RollupRpcFlag = cli.StringFlag{
		Name:     "rollup-rpc",
		Usage:    "HTTP provider URL for the rollup node.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "ROLLUP_RPC"),
	}
	L2OOAddressFlag = cli.StringFlag{
		Name:     "l2oo-address",
		Usage:    "Address of the L2OutputOracle contract.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "L2OO_ADDRESS"),
	}
	DGFAddressFlag = cli.StringFlag{
		Name:     "dgf-address",
		Usage:    "Address of the DisputeGameFactory contract.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "DGF_ADDRESS"),
	}
	PrivateKeyFlag = cli.StringFlag{
		Name:     "private-key",
		Usage:    "The private key to use with the service. Must not be used with mnemonic.",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "PRIVATE_KEY"),
	}
)

var cliFlags = []cli.Flag{
	L1EthRpcFlag,
	RollupRpcFlag,
	L2OOAddressFlag,
	DGFAddressFlag,
	PrivateKeyFlag,
}

func init() {
	cliFlags = append(cliFlags, oprpc.CLIFlags(envVarPrefix)...)

	cliFlags = append(cliFlags, oplog.CLIFlags(envVarPrefix)...)
	cliFlags = append(cliFlags, opmetrics.CLIFlags(envVarPrefix)...)
	cliFlags = append(cliFlags, oppprof.CLIFlags(envVarPrefix)...)
	cliFlags = append(cliFlags, TxManagerCLIFlags(envVarPrefix)...)

	Flags = cliFlags
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
