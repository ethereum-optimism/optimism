package main

import (
	"errors"
	"os"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	wheel "github.com/ethereum-optimism/optimism/op-wheel"
)

var (
	Version   = ""
	GitCommit = ""
	GitDate   = ""
)

func main() {
	app := cli.NewApp()
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "op-wheel"
	app.Usage = "Optimism Wheel is a CLI tool for the execution engine"
	app.Description = "Optimism Wheel is a CLI tool to direct the engine one way or the other with DB cheats and Engine API routines."
	app.Flags = []cli.Flag{wheel.GlobalGethLogLvlFlag}
	app.Before = func(c *cli.Context) error {
		log.Root().SetHandler(
			log.LvlFilterHandler(
				c.Generic(wheel.GlobalGethLogLvlFlag.Name).(*oplog.LvlFlagValue).LogLvl(),
				log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
			),
		)
		return nil
	}
	app.Action = cli.ActionFunc(func(c *cli.Context) error {
		return errors.New("see 'cheat' and 'engine' subcommands and --help")
	})
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		wheel.CheatCmd,
		wheel.EngineCmd,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
