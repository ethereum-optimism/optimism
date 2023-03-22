package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	legacy_bindings "github.com/ethereum-optimism/optimism/op-bindings/legacy-bindings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "rollover"
	app.Usage = "Commands for assisting in the rollover of the system"

	var flags []cli.Flag
	flags = append(flags, util.ClientsFlags...)
	flags = append(flags, util.AddressesFlags...)

	app.Commands = []*cli.Command{
		{
			Name:  "fetch",
			Usage: "Fetches batches in the specified range",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				clients, err := util.NewClients(cliCtx)
				if err != nil {
					return err
				}

				addresses, err := util.NewAddresses(cliCtx)
				if err != nil {
					return err
				}

				addressManager, err := bindings.NewAddressManager(addresses.AddressManager, clients.L1Client)
				if err != nil {
					return err
				}

				for {
					shutoffBlock, err := addressManager.GetAddress(&bind.CallOpts{}, "DTL_SHUTOFF_BLOCK")
					if err != nil {
						return err
					}
					if shutoffBlock.Big().Cmp(common.Big0) != 0 {
						break
					}
					time.Sleep(3 * time.Second)
				}

				ctc, err := legacy_bindings.NewCanonicalTransactionChain(addresses.CanonicalTransactionChain, clients.L1Client)
				if err != nil {
					return err
				}

				queueLength, err := ctc.GetQueueLength(&bind.CallOpts{})
				if err != nil {
					return err
				}

				totalElements, err := ctc.GetTotalElements(&bind.CallOpts{})
				if err != nil {
					return err
				}

				totalBatches, err := ctc.GetTotalBatches(&bind.CallOpts{})
				if err != nil {
					return err
				}

				pending, err := ctc.GetNumPendingQueueElements(&bind.CallOpts{})
				if err != nil {
					return err
				}

				log.Info(
					"CanonicalTransactionChain",
					"address", addresses.CanonicalTransactionChain,
					"queue-length", queueLength,
					"total-elements", totalElements,
					"total-batches", totalBatches,
					"pending", pending,
				)

				log.Info("Searching backwards for final deposit")
				blockNumber, err := clients.L2Client.BlockNumber(context.Background())
				if err != nil {
					return err
				}

				for {
					bn := new(big.Int).SetUint64(blockNumber)
					// TODO: Cannot use BlockByNumber, need to make a low level
					// call so that it can parse the legacy system's diff
					block, err := clients.L2Client.BlockByNumber(context.Background(), bn)
					if err != nil {
						return err
					}
					fmt.Println(block)
					blockNumber--

					// TODO: remove
					break
				}

				return nil
			},
		},
		{
			// TODO
			Name:  "",
			Usage: "",
			Flags: []cli.Flag{},
			Action: func(cliCtx *cli.Context) error {
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}
