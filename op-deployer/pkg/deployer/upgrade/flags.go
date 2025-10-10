package upgrade

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	embedded "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	v200 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v2_0_0"
	v300 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v3_0_0"
	v400 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v4_0_0"
	v410 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v4_1_0"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

var (
	ConfigFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "path to the config file",
	}
	OverrideArtifactsURLFlag = &cli.StringFlag{
		Name:  "override-artifacts-url",
		Usage: "override the artifacts URL",
	}
	OutfileFlag = &cli.StringFlag{
		Name:  "outfile",
		Usage: "path to write the output to, or - for stdout",
		Value: "-",
	}
)

var Commands = cli.Commands{
	&cli.Command{
		Name:  "v2.0.0",
		Usage: "upgrades a chain to version v2.0.0",
		Flags: append([]cli.Flag{
			deployer.L1RPCURLFlag,
			ConfigFlag,
			OverrideArtifactsURLFlag,
			OutfileFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: UpgradeCLI(v200.DefaultUpgrader),
	},
	&cli.Command{
		Name:  "v3.0.0",
		Usage: "upgrades a chain to version v3.0.0",
		Flags: append([]cli.Flag{
			deployer.L1RPCURLFlag,
			ConfigFlag,
			OverrideArtifactsURLFlag,
			OutfileFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: UpgradeCLI(v300.DefaultUpgrader),
	},
	&cli.Command{
		Name:  "v4.0.0",
		Usage: "upgrades a chain to version v4.0.0 (U16)",
		Flags: append([]cli.Flag{
			deployer.L1RPCURLFlag,
			ConfigFlag,
			OverrideArtifactsURLFlag,
			OutfileFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: UpgradeCLI(v400.DefaultUpgrader),
	},
	&cli.Command{
		Name:  "v4.1.0",
		Usage: "upgrades a chain to version v4.1.0 (U16a)",
		Flags: append([]cli.Flag{
			deployer.L1RPCURLFlag,
			ConfigFlag,
			OverrideArtifactsURLFlag,
			OutfileFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: UpgradeCLI(v410.DefaultUpgrader),
	},
	&cli.Command{
		Name:  "embedded",
		Usage: "upgrades a chain to version of contracts embedded in op-deployer",
		Flags: append([]cli.Flag{
			deployer.L1RPCURLFlag,
			ConfigFlag,
			OverrideArtifactsURLFlag,
			OutfileFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: UpgradeCLI(embedded.DefaultUpgrader),
	},
}
