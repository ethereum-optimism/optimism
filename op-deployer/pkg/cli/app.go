package cli

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/clean"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/manage"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/verify"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/urfave/cli/v2"
)

// NewApp creates and configures a new CLI application
func NewApp(versionWithMeta string) *cli.App {
	app := cli.NewApp()
	app.Version = versionWithMeta
	app.Name = "op-deployer"
	app.Usage = "Tool to configure and deploy OP Chains."
	app.Flags = cliapp.ProtectFlags(deployer.GlobalFlags)
	app.Before = func(context *cli.Context) error {
		if err := deployer.CreateCacheDir(context.String(deployer.CacheDirFlagName)); err != nil {
			return err
		}
		return nil
	}
	app.Commands = []*cli.Command{
		{
			Name:   "init",
			Usage:  "initializes a chain intent and state file",
			Flags:  cliapp.ProtectFlags(deployer.InitFlags),
			Action: deployer.InitCLI(),
		},
		{
			Name:   "apply",
			Usage:  "applies a chain intent to the chain",
			Flags:  cliapp.ProtectFlags(deployer.ApplyFlags),
			Action: deployer.ApplyCLI(),
		},
		{
			Name:        "upgrade",
			Usage:       "upgrades contracts by sending tx to OPCM.upgrade function",
			Flags:       cliapp.ProtectFlags(deployer.UpgradeFlags),
			Subcommands: upgrade.Commands,
		},
		{
			Name:        "bootstrap",
			Usage:       "bootstraps global contract instances",
			Subcommands: bootstrap.Commands,
		},
		{
			Name:        "inspect",
			Usage:       "inspects the state of a deployment",
			Subcommands: inspect.Commands,
		},
		{
			Name:        "clean",
			Usage:       "cleans up various things",
			Subcommands: clean.Commands,
		},
		{
			Name:   "verify",
			Usage:  "verifies deployed contracts on Etherscan",
			Flags:  cliapp.ProtectFlags(deployer.VerifyFlags),
			Action: verify.VerifyCLI,
		},
		{
			Name:        "manage",
			Usage:       "manages the chain",
			Subcommands: manage.Commands,
		},
	}
	return app
}
