package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/log"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

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

	// Check the config, no need to call `CheckAddresses()`
	if err := config.Check(); err != nil {
		return err
	}

	log.Info("Valid deploy config")
	return nil
}
