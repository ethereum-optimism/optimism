package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/version"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/urfave/cli/v2"
)

var (
	GitCommit = ""
	GitDate   = ""
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = opservice.FormatVersion(version.Version, GitCommit, GitDate, version.Meta)

func main() {
	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Name = "op-fetcher"
	app.Usage = "Tool to fetch OP Chains info from onchain"
	app.Flags = cliapp.ProtectFlags(deployer.GlobalFlags)
	app.Commands = []*cli.Command{
		{
			Name:   "fetch",
			Usage:  "fetches onchain data for a given chain",
			Flags:  cliapp.ProtectFlags(fetcher.FetchChainInfoFlags),
			Action: fetcher.FetchChainInfoCLI(),
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
