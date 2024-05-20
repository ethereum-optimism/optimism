package main

import (
	"fmt"
	"os"

	"github.com/ledgerwatch/log/v3"
	"github.com/urfave/cli/v2"

	"github.com/bobanetwork/boba/boba-chain-ops/genesis"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat()))

	app := &cli.App{
		Name:  "boba-regenerate",
		Usage: "Use engine api to regenerate the chain",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "l2-private-endpoint",
				Value:   "http://localhost:8551",
				Usage:   "Private endpoint for the L2 node",
				EnvVars: []string{"L2_PRIVATE_ENDPOINT"},
			},
			&cli.StringFlag{
				Name:    "l2-public-endpoint",
				Value:   "http://localhost:9545",
				Usage:   "Public endpoint for the L2 node",
				EnvVars: []string{"L2_PUBLIC_ENDPOINT"},
			},
			&cli.StringFlag{
				Name:    "l2-legacy-endpoint",
				Value:   "http://localhost:8545",
				Usage:   "Legacy endpoint for the L2 node",
				EnvVars: []string{"L2_LEGACY_ENDPOINT"},
			},
			&cli.StringFlag{
				Name:     "jwt-secret-path",
				Usage:    "Path to the file containing the JWT secret",
				EnvVars:  []string{"JWT_SECRET_PATH"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "rpc-time-out",
				Usage:   "Timeout for the RPC requests",
				Value:   "50s",
				EnvVars: []string{"RPC_TIME_OUT"},
			},
			&cli.StringFlag{
				Name:    "polling-interval",
				Usage:   "Interval between sending a request to the L2 node to build a new block",
				Value:   "100ms",
				EnvVars: []string{"POLLING_INTERVAL"},
			},
			&cli.Int64Flag{
				Name:    "hard-fork-block-number",
				Usage:   "Block number at which the hard fork will happen",
				Value:   0,
				EnvVars: []string{"HARD_FORK_BLOCK_NUMBER"},
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level",
				Value: "info",
			},
		},
		Action: func(ctx *cli.Context) error {
			logger := log.New()
			logLevel, err := log.LvlFromString(ctx.String("log-level"))
			if err != nil {
				logLevel = log.LvlInfo
				if ctx.String("log-level") != "" {
					log.Warn("invalid server.log_level set: " + ctx.String("log-level"))
				}
			}
			log.Root().SetHandler(
				log.LvlFilterHandler(
					logLevel,
					log.StreamHandler(os.Stdout, log.TerminalFormat()),
				),
			)

			l2PrivateEndpoint := ctx.String("l2-private-endpoint")
			l2PublicEndpoint := ctx.String("l2-public-endpoint")
			l2LegacyEndpoint := ctx.String("l2-legacy-endpoint")
			jwtSecretPath := ctx.String("jwt-secret-path")
			rpcTimeout := ctx.Duration("rpc-time-out")
			pollingInterval := ctx.Duration("polling-interval")
			hardforkBlockNumber := ctx.Int64("hard-fork-block-number")

			builderEngine, err := genesis.NewEngineConfig(l2PrivateEndpoint, l2PublicEndpoint, l2LegacyEndpoint, jwtSecretPath, rpcTimeout, pollingInterval, hardforkBlockNumber, logger)
			if err != nil {
				return fmt.Errorf("failed to create engine config: %w", err)
			}

			builderEngine.Start()
			builderEngine.Wait()

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("critical error exits", "err", err)
	}
}
