package watch

import (
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
)

var Subcommands = cli.Commands{
	{
		Name:  "oracle",
		Usage: "Watches the L2OutputOracle for new output proposals",
		Action: func(ctx *cli.Context) error {
			logger, err := config.LoggerFromCLI(ctx)
			if err != nil {
				return err
			}
			logger.Info("Listening for new output proposals")

			cfg, err := config.NewConfigFromCLI(ctx)
			if err != nil {
				return err
			}

			return Oracle(logger, cfg)
		},
	},
}
