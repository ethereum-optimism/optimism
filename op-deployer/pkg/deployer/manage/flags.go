package manage

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
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
}
