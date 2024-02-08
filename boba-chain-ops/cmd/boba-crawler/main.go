package main

import (
	"os"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/ether"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/genesis"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node"
	"github.com/ledgerwatch/log/v3"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat()))

	app := &cli.App{
		Name:  "boba-crawler",
		Usage: "Crawl all addresses that have sent or received ETH from the L2 node",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "rpc-url",
				Usage:    "RPC URL for an Ethereum Node",
				Required: true,
				EnvVars:  []string{"RPC_URL"},
			},
			&cli.Int64Flag{
				Name:    "end-block",
				Usage:   "Block number to end crawling at",
				Value:   0,
				EnvVars: []string{"END_BLOCK"},
			},
			&cli.StringFlag{
				Name:    "output-path",
				Usage:   "File to write the output to",
				Value:   "eth-addresses.json",
				EnvVars: []string{"OUTPUT_PATH"},
			},
			&cli.StringFlag{
				Name:    "rpc-time-out",
				Usage:   "Time out for RPC requests",
				Value:   "30s",
				EnvVars: []string{"RPC_TIME_OUT"},
			},
			&cli.StringFlag{
				Name:    "polling-interval",
				Usage:   "Interval between sending a request to the L2 node to request a block",
				Value:   "100ms",
				EnvVars: []string{"POLLING_INTERVAL"},
			},
			&cli.StringFlag{
				Name:  "alloc-path",
				Usage: "Path to the file containing the genesis allocation",
			},
			&cli.BoolFlag{
				Name:     "post-check-only",
				Usage:    "Only perform sanity checks",
				Required: false,
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

			postCheckOnly := ctx.Bool("post-check-only")
			if postCheckOnly {
				alloc, err := genesis.NewAlloc(ctx.String("alloc-path"))
				if err != nil {
					return err
				}
				if err := ether.CheckEthSlots(*alloc, ctx.String("output-path")); err != nil {
					return err
				}
				log.Info("All checks passed")
				return nil
			}

			rpcURL := ctx.String("rpc-url")
			endBlock := ctx.Int64("end-block")
			rpcTimeout := ctx.Duration("rpc-time-out")
			rpcPollingInterval := ctx.Duration("polling-interval")
			outputPath := ctx.String("output-path")

			client, err := node.NewRPC(rpcURL, rpcTimeout, logger)
			if err != nil {
				return err
			}

			crawler := ether.NewCrawler(client, endBlock, rpcPollingInterval, outputPath)
			if err := crawler.Start(); err != nil {
				return err
			}
			crawler.Wait()

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("critical error exits", "err", err)
	}
}
