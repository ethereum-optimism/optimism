package main

import (
	"context"
	"errors"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-node/cmd/doc"
	"github.com/ethereum-optimism/optimism/op-program/client"
	cldr "github.com/ethereum-optimism/optimism/op-program/client/driver"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

func main() {
	// Default to a machine parsable but relatively human friendly log format.
	// Don't do anything fancy to detect if color output is supported.
	logger := oplog.NewLogger(os.Stdout, oplog.CLIConfig{
		Level:  log.LvlInfo,
		Format: oplog.FormatLogFmt,
		Color:  false,
	})
	oplog.SetGlobalLogHandler(logger.GetHandler())

	// Invoke cancel when an interrupt is received.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		opio.BlockOnInterrupts()
		cancel()
	}()

	app := cli.NewApp()
	app.Name = "op-program"
	app.Action = curryMain(logger)
	app.Commands = []*cli.Command{
		{
			Name:        "doc",
			Subcommands: doc.Subcommands,
		},
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		if errors.Is(err, cldr.ErrClaimNotValid) {
			log.Error("Claim is invalid", "err", err)
			os.Exit(1)
		} else if err != nil {
			log.Error("Program failed", "err", err)
			os.Exit(2)
		}
	} else {
		log.Info("Claim successfully verified")
		os.Exit(0)
	}
}

func curryMain(logger log.Logger) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		return client.Main(ctx, logger)
	}
}
