package manage

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade"
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
	InitialBondFlag = &cli.StringFlag{
		Name:    "initial-bond",
		Usage:   "Initial bond amount required for the dispute game (value as string, in wei). Defaults to 1 ETH.",
		EnvVars: deployer.PrefixEnvVar("INITIAL_BOND"),
		Value:   "1000000000000000000",
	}
	StartingAnchorRootFlag = &cli.StringFlag{
		Name:    "starting-anchor-root",
		Usage:   "Starting anchor root.",
		EnvVars: deployer.PrefixEnvVar("STARTING_ANCHOR_ROOT"),
	}
	StartingAnchorL2SequenceNumberFlag = &cli.Uint64Flag{
		Name:    "starting-anchor-l2-sequence-number",
		Usage:   "Starting anchor L2 sequence number.",
		EnvVars: deployer.PrefixEnvVar("STARTING_ANCHOR_L2_SEQUENCE_NUMBER"),
	}
	// OPCM v2 flags
	MigrateStartingRespectedGameTypeFlag = &cli.Uint64Flag{
		Name:    "starting-respected-game-type",
		Usage:   "Starting respected game type for OPCM v2 migration. Defaults to 4 (Super Cannon).",
		EnvVars: deployer.PrefixEnvVar("STARTING_RESPECTED_GAME_TYPE"),
		Value:   4,
	}
	MigrateDisputeGameEnabledFlag = &cli.BoolFlag{
		Name:    "dispute-game-enabled",
		Usage:   "Whether the dispute game should be enabled. Used for OPCM v2 migration.",
		EnvVars: deployer.PrefixEnvVar("DISPUTE_GAME_ENABLED"),
		Value:   true,
	}
)

var Commands = cli.Commands{
	&cli.Command{
		Name:  "add-game-type-v2",
		Usage: "allows to add new game types to the chain using the OPContractsManager V2",
		Flags: append([]cli.Flag{
			deployer.L1RPCURLFlag,
			upgrade.ConfigFlag,
			upgrade.OverrideArtifactsURLFlag,
			upgrade.OutfileFlag,
			deployer.CacheDirFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: AddGameTypeOPCMV2CLI,
	},
	&cli.Command{
		Name:  "migrate",
		Usage: "migrates the chain to use superproofs using OPCM v2.",
		Flags: append([]cli.Flag{
			deployer.CacheDirFlag,
			deployer.L1RPCURLFlag,
			deployer.PrivateKeyFlag,
			deployer.ArtifactsLocatorFlag,
			L1ProxyAdminOwnerFlag,
			OPCMImplFlag,
			StartingAnchorRootFlag,
			StartingAnchorL2SequenceNumberFlag,
			InitialBondFlag,
			SystemConfigProxyFlag,
			DisputeAbsolutePrestateFlag,
			// OPCM v2 flags
			MigrateStartingRespectedGameTypeFlag,
			MigrateDisputeGameEnabledFlag,
			DisputeGameTypeFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: MigrateCLI,
	},
}
