package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/cmd/batch_decoder/fetch"
	"github.com/ethereum-optimism/optimism/op-node/cmd/batch_decoder/reassemble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "batch-decoder"
	app.Usage = "Optimism Batch Decoding Utility"
	app.Commands = []cli.Command{
		{
			Name:  "fetch",
			Usage: "Fetches batches in the specified range",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:     "start",
					Required: true,
					Usage:    "First block (inclusive) to fetch",
				},
				cli.IntFlag{
					Name:     "end",
					Required: true,
					Usage:    "Last block (exclusive) to fetch",
				},
				cli.StringFlag{
					Name:     "inbox",
					Required: true,
					Usage:    "Batch Inbox Address",
				},
				cli.StringFlag{
					Name:     "sender",
					Required: true,
					Usage:    "Batch Sender Address",
				},
				cli.StringFlag{
					Name:  "out",
					Value: "/tmp/batch_decoder/transactions_cache",
					Usage: "Cache directory for the found transactions",
				},
				cli.StringFlag{
					Name:     "l1",
					Required: true,
					Usage:    "L1 RPC URL",
					EnvVar:   "L1_RPC",
				},
			},
			Action: func(cliCtx *cli.Context) error {
				client, err := ethclient.Dial(cliCtx.String("l1"))
				if err != nil {
					log.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				chainID, err := client.ChainID(ctx)
				if err != nil {
					log.Fatal(err)
				}
				config := fetch.Config{
					Start:   uint64(cliCtx.Int("start")),
					End:     uint64(cliCtx.Int("end")),
					ChainID: chainID,
					BatchSenders: map[common.Address]struct{}{
						common.HexToAddress(cliCtx.String("sender")): struct{}{},
					},
					BatchInbox:   common.HexToAddress(cliCtx.String("inbox")),
					OutDirectory: cliCtx.String("out"),
				}
				totalValid, totalInvalid := fetch.Batches(client, config)
				fmt.Printf("Fetched batches in range [%v,%v). Found %v valid & %v invalid batches\n", config.Start, config.End, totalValid, totalInvalid)
				fmt.Printf("Fetch Config: Chain ID: %v. Inbox Address: %v. Valid Senders: %v.\n", config.ChainID, config.BatchInbox, config.BatchSenders)
				fmt.Printf("Wrote transactions with batches to %v\n", config.OutDirectory)
				return nil
			},
		},
		{
			Name:  "reassemble",
			Usage: "Reassembles channels from fetched batches",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "inbox",
					Value: "0xff00000000000000000000000000000000000420",
					Usage: "Batch Inbox Address",
				},
				cli.StringFlag{
					Name:  "in",
					Value: "/tmp/batch_decoder/transactions_cache",
					Usage: "Cache directory for the found transactions",
				},
				cli.StringFlag{
					Name:  "out",
					Value: "/tmp/batch_decoder/channel_cache",
					Usage: "Cache directory for the found channels",
				},
			},
			Action: func(cliCtx *cli.Context) error {
				config := reassemble.Config{
					BatchInbox:   common.HexToAddress(cliCtx.String("inbox")),
					InDirectory:  cliCtx.String("in"),
					OutDirectory: cliCtx.String("out"),
				}
				reassemble.Channels(config)
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
