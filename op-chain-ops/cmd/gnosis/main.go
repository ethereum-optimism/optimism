package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/version"
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
	app.Name = "gnosis"
	app.Usage = "tool to interact with pre-deployed gnosis safe contracts"
	app.Flags = cliapp.ProtectFlags(GlobalFlags)
	app.Commands = []*cli.Command{
		{
			Name:   "send-tx",
			Usage:  "send tx via Gnosis Safe using calldata",
			Flags:  cliapp.ProtectFlags(SendGnosisTxFlags),
			Action: SendGnosisTransactionCLI,
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
