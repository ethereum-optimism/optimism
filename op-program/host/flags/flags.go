package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	nodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	service "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const envVarPrefix = "OP_PROGRAM"

var (
	RollupConfig = cli.StringFlag{
		Name:   "rollup.config",
		Usage:  "Rollup chain parameters",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "ROLLUP_CONFIG"),
	}
	Network = cli.StringFlag{
		Name:   "network",
		Usage:  fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVar: service.PrefixEnvVar(envVarPrefix, "NETWORK"),
	}
	L2NodeAddr = cli.StringFlag{
		Name:   "l2",
		Usage:  "Address of L2 JSON-RPC endpoint to use (eth and debug namespace required)",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "L2_RPC"),
	}
	L1Head = cli.StringFlag{
		Name:   "l1.head",
		Usage:  "Hash of the L1 head block. Derivation stops after this block is processed.",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "L1_HEAD"),
	}
	L2Head = cli.StringFlag{
		Name:   "l2.head",
		Usage:  "Hash of the agreed L2 block to start derivation from",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "L2_HEAD"),
	}
	L2GenesisPath = cli.StringFlag{
		Name:   "l2.genesis",
		Usage:  "Path to the op-geth genesis file",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "L2_GENESIS"),
	}
	L1NodeAddr = cli.StringFlag{
		Name:   "l1",
		Usage:  "Address of L1 JSON-RPC endpoint to use (eth namespace required)",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "L1_RPC"),
	}
	L1TrustRPC = cli.BoolFlag{
		Name:   "l1.trustrpc",
		Usage:  "Trust the L1 RPC, sync faster at risk of malicious/buggy RPC providing bad or inconsistent L1 data",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "L1_TRUST_RPC"),
	}
	L1RPCProviderKind = cli.GenericFlag{
		Name: "l1.rpckind",
		Usage: "The kind of RPC provider, used to inform optimal transactions receipts fetching, and thus reduce costs. Valid options: " +
			nodeflags.EnumString[sources.RPCProviderKind](sources.RPCProviderKinds),
		EnvVar: service.PrefixEnvVar(envVarPrefix, "L1_RPC_KIND"),
		Value: func() *sources.RPCProviderKind {
			out := sources.RPCKindBasic
			return &out
		}(),
	}
)

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

var programFlags = []cli.Flag{
	RollupConfig,
	Network,
	L2NodeAddr,
	L1Head,
	L2Head,
	L2GenesisPath,
	L1NodeAddr,
	L1TrustRPC,
	L1RPCProviderKind,
}

func init() {
	Flags = append(Flags, oplog.CLIFlags(envVarPrefix)...)
	Flags = append(Flags, programFlags...)
}

func CheckRequired(ctx *cli.Context) error {
	rollupConfig := ctx.GlobalString(RollupConfig.Name)
	network := ctx.GlobalString(Network.Name)
	if rollupConfig == "" && network == "" {
		return fmt.Errorf("flag %s or %s is required", RollupConfig.Name, Network.Name)
	}
	if rollupConfig != "" && network != "" {
		return fmt.Errorf("cannot specify both %s and %s", RollupConfig.Name, Network.Name)
	}
	if ctx.GlobalString(L2GenesisPath.Name) == "" {
		return fmt.Errorf("flag %s is required", L2GenesisPath.Name)
	}
	if ctx.GlobalString(L2Head.Name) == "" {
		return fmt.Errorf("flag %s is required", L2Head.Name)
	}
	if ctx.GlobalString(L1Head.Name) == "" {
		return fmt.Errorf("flag %s is required", L1Head.Name)
	}
	return nil
}
