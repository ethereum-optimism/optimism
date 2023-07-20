package main

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/geth"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

const version = "1.0.0"

type ServerAPI struct {
	c chan any
}

func (s *ServerAPI) Stop() {
	s.c <- struct{}{}
}

func NewServerAPI(c chan any) *ServerAPI {
	return &ServerAPI{
		c: c,
	}
}

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "geth",
		Usage: "Wrapper around a geth node",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "deploy-config",
				Usage: "Path to hardhat deploy config file",
			},
			&cli.StringFlag{
				Name:  "outfile.l1",
				Usage: "Path to L1 genesis output file",
			},
		},
		Action: func(ctx *cli.Context) error {
			deployConfig := ctx.String("deploy-config")
			if len(deployConfig) == 0 {
				return errors.New("Must specify a deploy-config")
			}

			config, err := genesis.NewDeployConfig(deployConfig)
			if err != nil {
				return err
			}

			// Temporary assertion on the clique signer address
			if config.CliqueSignerAddress != (common.Address{}) {
				return errors.New("Clique signer address must be empty")
			}

			if config.L1BlockTime == 0 {
				log.Warn("Sanitizing L1 blocktime to 1")
				config.L1BlockTime = 1
			}

			genesis, err := genesis.NewL1Genesis(config)
			if err != nil {
				return err
			}

			l1Node, l1Backend, err := geth.InitL1Geth(config, genesis, clock.SystemClock, []*ecdsa.PrivateKey{})
			if err != nil {
				return err
			}

			if err := l1Node.Start(); err != nil {
				return err
			}
			if err = l1Backend.Start(); err != nil {
				return err
			}

			c := make(chan any)

			rpcCfg := oprpc.ReadCLIConfig(ctx)
			server := oprpc.NewServer(
				rpcCfg.ListenAddr,
				rpcCfg.ListenPort,
				version,
				oprpc.WithLogger(log.New("rpc_server")),
			)
			server.AddAPI(gethrpc.API{
				Namespace: "admin",
				Service:   NewServerAPI(c),
			})

			if err := server.Start(); err != nil {
				return fmt.Errorf("error starting RPC server: %w", err)
			}

			interruptChannel := make(chan os.Signal, 1)
			signal.Notify(interruptChannel, opio.DefaultInterruptSignals...)

			for {
				select {
				case <-c:
				case <-interruptChannel:
					defer close(c)
					defer close(interruptChannel)

					// defer l1Backend.Stop()
					// defer l1Node.Close()

					/*
						if err := l1Backend.Stop(); err != nil {
							return err
						}
					*/
					/*
						if err := l1Node.Close(); err != nil {
							return err
						}
					*/
					return nil
				}
			}
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("problem starting geth", "err", err)
	}
}
