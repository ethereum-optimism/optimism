package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "op-deployer"
	app.Usage = "Tool to configure and deploy OP Chains."
	app.Flags = cliapp.ProtectFlags(deployer.GlobalFlags)
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
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}
