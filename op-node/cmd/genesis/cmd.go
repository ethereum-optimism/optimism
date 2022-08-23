package genesis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		Usage: "Initialized a new devnet genesis file",
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

			deployConfigDirectory := ctx.String("deploy-config")
			deployConfig := filepath.Join(deployConfigDirectory, network+".json")
			config, err := genesis.NewDeployConfig(deployConfig)
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
}
