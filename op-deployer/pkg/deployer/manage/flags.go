package manage

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

var (
	ConfigFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "path to the config file",
	}
)

var Commands = cli.Commands{
	&cli.Command{
		Name:  "add-game-type",
		Usage: "adds a new game type to the chain",
		Flags: append([]cli.Flag{
			deployer.L1RPCURLFlag,
			deployer.ArtifactsLocatorFlag,
			ConfigFlag,
		}, oplog.CLIFlags(deployer.EnvVarPrefix)...),
		Action: AddGameTypeCLI,
	},
}
