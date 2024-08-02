package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	oplog.SetGlobalLogHandler(log.NewTerminalHandler(os.Stderr, color))

	app := &cli.App{
		Name:  "check-deploy-config",
		Usage: "Check that a deploy config is valid",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "path",
				Required: true,
				Usage:    "File system path to the deploy config",
			},
		},
		Action: entrypoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error checking deploy config", "err", err)
	}
}

func entrypoint(ctx *cli.Context) error {
	path := ctx.String("path")

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	log.Info("Checking deploy config", "name", name, "path", path)

	config, err := genesis.NewDeployConfig(path)
	if err != nil {
		return err
	}

	cfg := oplog.DefaultCLIConfig()
	logger := oplog.NewLogger(ctx.App.Writer, cfg)

	// Check the config, no need to call `CheckAddresses()`
	if err := config.Check(logger); err != nil {
		return err
	}

	log.Info("Valid deploy config")
	return nil
}
