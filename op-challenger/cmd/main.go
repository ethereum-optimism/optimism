package main

import (
	"os"

	op_challenger "github.com/ethereum-optimism/optimism/op-challenger"
	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-service/app"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/version"
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

type ConfigAction func(log log.Logger, config *config.Config, metricsFactory opmetrics.Factory) error

func run(args []string, action ConfigAction) error {
	metricsSvc := opmetrics.NewService(func(service *opmetrics.Service) {
		// TODO: Ultimately this should create the op-challenger Metrics instance.
		// Kind of ugly to have to create the app specific Metrics wrapper here though
		// Maybe just embrace it and pass the Metrics instance into ConfigAction instead of just the factory?
		metrics.MakeTxMetrics("foo", service.Factory())
	})
	return app.Run(args, "OP_CHALLENGER", func(app *cli.App) {
		app.Name = "op-challenger"
		app.Version = VersionWithMeta
		app.Usage = "Challenge outputs"
		app.Description = "Ensures that on chain outputs are correct."
		app.Flags = flags.Flags
	}, func(l log.Logger, ctx *cli.Context) error {
		cfg, err := config.NewConfigFromCLI(ctx)
		if err != nil {
			return err
		}
		return action(l, cfg, metricsSvc.Factory())
	}, metricsSvc)
}
