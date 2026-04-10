package manage

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/urfave/cli/v2"
)

// AddGameTypeOPCMV2CLI is the CLI command for adding a new game type to the chain using the OPContractsManager V2.
func AddGameTypeOPCMV2CLI(cliCtx *cli.Context) error {
	return upgrade.UpgradeCLI(embedded.DefaultUpgrader)(cliCtx)
}
