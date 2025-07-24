package main

import (
	"fmt"
	"os"

	"github.com/HashKeyChain/verse/op-deployer/pkg/deployer/version"
	"github.com/HashKeyChain/verse/op-fetcher/pkg/fetcher/fetch"
	opservice "github.com/HashKeyChain/verse/op-service"
	"github.com/HashKeyChain/verse/op-service/cliapp"
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
	app.Usage = "tool to fetch OP Chain config info from onchain"
	app.Flags = cliapp.ProtectFlags(fetch.GlobalFlags)
	app.Commands = []*cli.Command{
		{
			Name:   "fetch",
			Usage:  "fetches onchain data for a given chain",
			Flags:  cliapp.ProtectFlags(fetch.FetchChainInfoFlags),
			Action: fetch.FetchChainInfoCLI(),
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
