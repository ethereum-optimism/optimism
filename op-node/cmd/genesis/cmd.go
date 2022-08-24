package genesis

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/hardhat"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/ethclient"
)

var Subcommands = cli.Commands{
	{
		Name:  "devnet-l2",
		Usage: "Initialized a new L2 devnet genesis file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "artifacts",
				Usage: "Comma delimeted list of hardhat artifact directories",
			},
			cli.StringFlag{
				Name:  "network",
				Usage: "Name of hardhat deploy network",
			},
			cli.StringFlag{
				Name:  "deployments",
				Usage: "Comma delimated list of hardhat deploy artifact directories",
			},
			cli.StringFlag{
				Name:  "deploy-config",
				Usage: "Path to hardhat deploy config directory",
			},
			cli.StringFlag{
				Name:  "rpc-url",
				Usage: "L1 RPC URL",
			},
			cli.StringFlag{
				Name:  "outfile",
				Usage: "Path to file to write output to",
			},
		},
		Action: func(ctx *cli.Context) error {
			// Turn off logging for this command unless it is a critical
			// error so that the output can be piped to jq
			log.Root().SetHandler(
				log.LvlFilterHandler(
					log.LvlCrit,
					log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
				),
			)

			artifact := ctx.String("artifacts")
			artifacts := strings.Split(artifact, ",")
			deployment := ctx.String("deployments")
			deployments := strings.Split(deployment, ",")
			network := ctx.String("network")
			hh, err := hardhat.New(network, artifacts, deployments)
			if err != nil {
				return err
			}

			deployConfig := ctx.String("deploy-config")
			config, err := genesis.NewDeployConfigWithNetwork(network, deployConfig)
			if err != nil {
				return err
			}

			rpcUrl := ctx.String("rpc-url")
			client, err := ethclient.Dial(rpcUrl)
			if err != nil {
				return err
			}

			gen, err := genesis.BuildOptimismDeveloperGenesis(hh, config, client)
			if err != nil {
				return err
			}

			file, err := json.MarshalIndent(gen, "", " ")
			if err != nil {
				return err
			}

			outfile := ctx.String("outfile")
			if outfile == "" {
				fmt.Println(string(file))
			} else {
				if err := os.WriteFile(outfile, file, 0644); err != nil {
					return err
				}
			}
			return nil
		},
	},
	{
		Name:  "devnet-l1",
		Usage: "Initialized a new L1 devnet genesis file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "network",
				Usage: "Name of hardhat deploy network",
			},
			cli.StringFlag{
				Name:  "deploy-config",
				Usage: "Path to hardhat deploy config directory",
			},
			cli.StringFlag{
				Name:  "outfile",
				Usage: "Path to file to write output to",
			},
		},
		Action: func(ctx *cli.Context) error {
			network := ctx.String("network")
			deployConfig := ctx.String("deploy-config")

			config, err := genesis.NewDeployConfigWithNetwork(network, deployConfig)
			if err != nil {
				return err
			}

			gen, err := genesis.BuildL1DeveloperGenesis(config)
			if err != nil {
				return err
			}

			file, err := json.MarshalIndent(gen, "", " ")
			if err != nil {
				return err
			}

			outfile := ctx.String("outfile")
			if outfile == "" {
				fmt.Println(string(file))
			} else {
				if err := os.WriteFile(outfile, file, 0644); err != nil {
					return err
				}
			}
			return nil
		},
	},
}
