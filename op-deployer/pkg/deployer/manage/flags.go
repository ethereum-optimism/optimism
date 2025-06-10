package manage

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

var (
	L1ProxyAdminOwnerFlag = &cli.StringFlag{
		Name:    "l1-proxy-admin-owner-address",
		Usage:   "Address to use for as the proxy admin owner. Not compatible with the --workdir flag.",
		EnvVars: deployer.PrefixEnvVar("PROXY_ADMIN_OWNER"),
	}
	OPCMImplFlag = &cli.StringFlag{
		Name:    "opcm-impl-address",
		Usage:   "Address of the OPCM implementation contract. Not compatible with the --workdir flag.",
		EnvVars: deployer.PrefixEnvVar("OPCM_IMPL_ADDRESS"),
	}
	SystemConfigProxyFlag = &cli.StringFlag{
		Name:    "system-config-proxy-address",
		Usage:   "Address of the SystemConfig proxy contract. Not compatible with the --workdir flag.",
		EnvVars: deployer.PrefixEnvVar("SYSTEM_CONFIG_PROXY_ADDRESS"),
	}
	OPChainProxyAdminFlag = &cli.StringFlag{
		Name:    "op-chain-proxy-admin-address",
		Usage:   "Address of the OP Chain's proxy admin on L1. Not compatible with the --workdir flag.",
		EnvVars: deployer.PrefixEnvVar("OP_CHAIN_PROXY_ADMIN_ADDRESS"),
	}
	DelayedWETHProxyFlag = &cli.StringFlag{
		Name: "delayed-weth-proxy-address",
		Usage: "Address of the DelayedWETHProxy contract to include as part of this game type. If not specified, " +
			"one will be deployed for you by the OPCM.",
		EnvVars: deployer.PrefixEnvVar("DELAYED_WETH_PROXY_ADDRESS"),
	}
	DisputeGameTypeFlag = &cli.Uint64Flag{
		Name:    "dispute-game-type",
		Usage:   "Numeric type identifier for the dispute game.",
		EnvVars: deployer.PrefixEnvVar("DISPUTE_GAME_TYPE"),
		Value:   uint64(standard.DisputeGameType),
	}
	DisputeAbsolutePrestateFlag = &cli.StringFlag{
		Name:    "dispute-absolute-prestate",
		Usage:   "The absolute prestate hash for the dispute game. Defaults to the standard value.",
		EnvVars: deployer.PrefixEnvVar("DISPUTE_ABSOLUTE_PRESTATE"),
		Value:   standard.DisputeAbsolutePrestate.Hex(),
	}
	DisputeMaxGameDepthFlag = &cli.Uint64Flag{
		Name:    "dispute-max-game-depth",
		Usage:   "Maximum depth of the dispute game tree (value as string). Defaults to the standard value.",
		EnvVars: deployer.PrefixEnvVar("DISPUTE_MAX_GAME_DEPTH"),
		Value:   standard.DisputeMaxGameDepth,
	}
	DisputeSplitDepthFlag = &cli.Uint64Flag{
		Name:    "dispute-split-depth",
		Usage:   "Depth at which the dispute game tree splits (value as string). Defaults to the standard value.",
		EnvVars: deployer.PrefixEnvVar("DISPUTE_SPLIT_DEPTH"),
		Value:   standard.DisputeSplitDepth,
	}
	DisputeClockExtensionFlag = &cli.Uint64Flag{
		Name:    "dispute-clock-extension",
		Usage:   "Clock extension in seconds for dispute game timing. Defaults to the standard value.",
		EnvVars: deployer.PrefixEnvVar("DISPUTE_CLOCK_EXTENSION"),
		Value:   standard.DisputeClockExtension,
	}
	DisputeMaxClockDurationFlag = &cli.Uint64Flag{
		Name:    "dispute-max-clock-duration",
		Usage:   "Maximum clock duration in seconds for dispute game timing. Defaults to the standard value.",
		EnvVars: deployer.PrefixEnvVar("DISPUTE_MAX_CLOCK_DURATION"),
		Value:   standard.DisputeMaxClockDuration,
	}
	InitialBondFlag = &cli.StringFlag{
		Name:    "initial-bond",
		Usage:   "Initial bond amount required for the dispute game (value as string, in wei). Defaults to 1 ETH.",
		EnvVars: deployer.PrefixEnvVar("INITIAL_BOND"),
		Value:   "1000000000000000000",
	}
	VMFlag = &cli.StringFlag{
		Name:    "vm-address",
		Usage:   "Address of the VM contract used by the dispute game.",
		EnvVars: deployer.PrefixEnvVar("VM_ADDRESS"),
	}
	PermissionlessFlag = &cli.BoolFlag{
		Name:    "permissionless",
		Usage:   "Boolean indicating if the dispute game should be deployed in permissionless mode.",
		EnvVars: deployer.PrefixEnvVar("PERMISSIONED"),
	}
	SaltMixerFlag = &cli.StringFlag{
		Name:    "salt-mixer",
		Usage:   "String value for the salt mixer, used in CREATE2 address calculation. Default to keccak256(\"op-stack-contract-impls-salt-v0\").",
		EnvVars: deployer.PrefixEnvVar("SALT_MIXER"),
		Value:   "89fca2352a158519d2daabf7e53686272e828ddbff9487204546d918490b2ecf",
	}
	WorkdirFlag = &cli.StringFlag{
		Name:    "workdir",
		Usage:   "Path to a working directory containing a state file. Addresses will be retrieved from the state file, and cannot be specified on the command line.",
		EnvVars: deployer.PrefixEnvVar("WORKDIR"),
	}
	L2ChainIDFlag = &cli.StringFlag{
		Name:    "l2-chain-id",
		Usage:   "Chain ID of the L2 network to retrieve from state. Must be specified when --workdir is set.",
		EnvVars: deployer.PrefixEnvVar("CHAIN_ID"),
	}
)

var Commands = cli.Commands{
	&cli.Command{
		Name:  "add-game-type",
		Usage: "adds a new game type to the chain",
		Flags: append([]cli.Flag{
			deployer.L1RPCURLFlag,
			deployer.ArtifactsLocatorFlag,
			L1ProxyAdminOwnerFlag,
			OPCMImplFlag,
			SystemConfigProxyFlag,
			OPChainProxyAdminFlag,
			DelayedWETHProxyFlag,
			DisputeGameTypeFlag,
			DisputeAbsolutePrestateFlag,
			DisputeMaxGameDepthFlag,
			DisputeSplitDepthFlag,
			DisputeClockExtensionFlag,
			DisputeMaxClockDurationFlag,
			InitialBondFlag,
			VMFlag,
			PermissionlessFlag,
			SaltMixerFlag,
			WorkdirFlag,
			L2ChainIDFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: AddGameTypeCLI,
	},
}
