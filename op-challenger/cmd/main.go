package main

import (
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/version"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

var (
	GitCommit = ""
	GitDate   = ""
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := version.Version
	if GitCommit != "" {
		v += "-" + GitCommit[:8]
	}
	if GitDate != "" {
		v += "-" + GitDate
	}
	if version.Meta != "" {
		v += "-" + version.Meta
	}
	return v
}()

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Flags = flags.Flags
	app.Name = "op-challenger"
	app.Usage = "Challenge outputs"
	app.Description = "Ensures that on chain outputs are correct."
	app.Action = func(ctx *cli.Context) error {
		return FaultGame(VersionWithMeta, ctx)
	}
	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}

type ConfigAction func(log log.Logger, version string, config *config.Config) error

func FaultGame(version string, cliCtx *cli.Context) error {
	cfg, err := config.NewConfigFromCLI(cliCtx)
	if err != nil {
		return err
	}
	if err := cfg.Check(); err != nil {
		return err
	}
	log := oplog.NewLogger(cfg.LogConfig)
	log.Info("Fault game started")
	return nil
}
