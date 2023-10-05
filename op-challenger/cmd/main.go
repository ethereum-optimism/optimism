package main

import (
	"context"
	"fmt"
	"os"

	op_challenger "github.com/ethereum-optimism/optimism/op-challenger"
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
	args := os.Args
	if err := run(args, op_challenger.Main); err != nil {
		log.Crit("Application failed", "err", err)
	}
}

type ConfigAction func(ctx context.Context, log log.Logger, config *config.Config) error

func run(args []string, action ConfigAction) error {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Version = VersionWithMeta

	for _, f := range flags.Flags {
		if f.Names()[0] == "log.level" {
			if f.IsSet() {
				panic("flag is already set!")
			}
			v, ok := f.(*cli.GenericFlag)
			if !ok {
				panic("not a generic flag!")
			}
			fv, ok := v.Value.(*oplog.LvlFlagValue)
			if !ok {
				panic("not a lvl flag value!")
			}
			if fv == nil {
				panic("nil lvl flag value!")
			}
			fmt.Printf("lvl flag value: %q\n", fv.String())
		}
	}

	app.Flags = flags.Flags
	app.Name = "op-challenger"
	app.Usage = "Challenge outputs"
	app.Description = "Ensures that on chain outputs are correct."
	app.Action = func(ctx *cli.Context) error {
		logger, err := setupLogging(ctx)
		if err != nil {
			return err
		}
		logger.Info("Starting op-challenger", "version", VersionWithMeta)

		cfg, err := flags.NewConfigFromCLI(ctx)
		if err != nil {
			return err
		}
		return action(ctx.Context, logger, cfg)
	}
	return app.Run(args)
}

func setupLogging(ctx *cli.Context) (log.Logger, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	logger := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
	oplog.SetGlobalLogHandler(logger.GetHandler())
	return logger, nil
}
