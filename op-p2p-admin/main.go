package main

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-p2p-admin/cmd"
	"github.com/ethereum-optimism/optimism/op-p2p-admin/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
	"os"
)

var (
	Version   = "v0.1.0"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "op-p2p-admin"
	app.Usage = "P2P Admin monitoring and control"
	app.Description = "Service for monitoring and controlling p2p nodes"
	app.Action = curryMain(Version)
	app.Commands = []cli.Command{
		// TODO: should de-dup this metrics doc command with proposer/batcher/etc.
		//{
		//	Name:        "doc",
		//	Subcommands: doc.Subcommands,
		//},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func curryMain(version string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		return cmd.Main(version, ctx)
	}
}
