package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/cmd/batch_decoder/fetch"
	"github.com/ethereum-optimism/optimism/op-node/cmd/batch_decoder/reassemble"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "batch-decoder"
	app.Usage = "Optimism Batch Decoding Utility"
	app.Commands = []*cli.Command{
		{
			Name:  "fetch",
			Usage: "Fetches batches in the specified range",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:     "start",
					Required: true,
					Usage:    "First block (inclusive) to fetch",
				},
				&cli.IntFlag{
					Name:     "end",
					Required: true,
					Usage:    "Last block (exclusive) to fetch",
				},
				&cli.Uint64Flag{
					Name:  "l2-chain-id",
					Usage: "L2 chain ID to load inbox & sender from superchain-registry",
				},
				&cli.StringFlag{
					Name:  "inbox",
					Usage: "Batch Inbox Address",
				},
				&cli.StringFlag{
					Name:  "sender",
					Usage: "Batch Sender Address",
				},
				&cli.StringFlag{
					Name:  "out",
					Value: "/tmp/batch_decoder/transactions_cache",
					Usage: "Cache directory for the found transactions",
				},
				&cli.StringFlag{
					Name:     "l1",
					Required: true,
					Usage:    "L1 RPC URL",
					EnvVars:  []string{"L1_RPC"},
				},
				&cli.StringFlag{
					Name:     "l1.beacon",
					Required: false,
					Usage:    "Address of L1 Beacon-node HTTP endpoint to use",
					EnvVars:  []string{"L1_BEACON"},
				},
				&cli.IntFlag{
					Name:  "concurrent-requests",
					Value: 10,
					Usage: "Concurrency level when fetching L1",
				},
			},
			Action: func(cliCtx *cli.Context) error {
				l1Client, err := ethclient.Dial(cliCtx.String("l1"))
				if err != nil {
					log.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				chainID, err := l1Client.ChainID(ctx)
				if err != nil {
					log.Fatal(err)
				}

				beaconAddr := cliCtx.String("l1.beacon")
				var beacon *sources.L1BeaconClient
				if beaconAddr != "" {
					beaconClient := sources.NewBeaconHTTPClient(client.NewBasicHTTPClient(beaconAddr, nil))
					beaconCfg := sources.L1BeaconClientConfig{FetchAllSidecars: false}
					beacon = sources.NewL1BeaconClient(beaconClient, beaconCfg)
					_, err := beacon.GetVersion(ctx)
					if err != nil {
						log.Fatal(fmt.Errorf("failed to check L1 Beacon API version: %w", err))
					}
				} else {
					fmt.Println("L1 Beacon endpoint not set. Unable to fetch post-ecotone channel frames")
				}

				var inbox, sender common.Address
				if cliCtx.IsSet("l2-chain-id") {
					l2ChainID := cliCtx.Uint64("l2-chain-id")
					rcfg, err := rollup.LoadOPStackRollupConfig(l2ChainID)
					if err != nil {
						return err
					}
					inbox = rcfg.BatchInboxAddress
					sender = rcfg.Genesis.SystemConfig.BatcherAddr
				} else if cliCtx.IsSet("inbox") && cliCtx.IsSet("sender") {
					inbox = common.HexToAddress(cliCtx.String("inbox"))
					sender = common.HexToAddress(cliCtx.String("sender"))
				} else {
					return fmt.Errorf("either --l2-chain-id or both --inbox and --sender must be set")
				}

				config := fetch.Config{
					Start:              uint64(cliCtx.Int("start")),
					End:                uint64(cliCtx.Int("end")),
					ChainID:            chainID,
					BatchSenders:       map[common.Address]struct{}{sender: {}},
					BatchInbox:         inbox,
					OutDirectory:       cliCtx.String("out"),
					ConcurrentRequests: uint64(cliCtx.Int("concurrent-requests")),
				}
				fmt.Printf("Fetch Config: L1 Chain ID: %v. Inbox Address: %v. Valid Senders: %v.\n", config.ChainID, config.BatchInbox, config.BatchSenders)

				totalValid, totalInvalid := fetch.Batches(l1Client, beacon, config)
				fmt.Printf("Fetched batches in range [%v,%v). Found %v valid & %v invalid batches\n", config.Start, config.End, totalValid, totalInvalid)
				fmt.Printf("Wrote transactions with batches to %v\n", config.OutDirectory)
				return nil
			},
		},

		{
			Name:  "reassemble",
			Usage: "Reassembles channels from fetched batch transactions and decode batches",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "in",
					Value: "/tmp/batch_decoder/transactions_cache",
					Usage: "Cache directory for the found transactions",
				},
				&cli.StringFlag{
					Name:  "out",
					Value: "/tmp/batch_decoder/channel_cache",
					Usage: "Cache directory for the found channels",
				},
				&cli.Uint64Flag{
					Name:  "l2-chain-id",
					Usage: "L2 chain ID to load rollup config from superchain-registry",
				},
				&cli.PathFlag{
					Name:  "rollup-config",
					Value: "rollup.json",
					Usage: "Path to rollup config JSON file. Must only be set if not using l2-chain-id flag.",
				},
			},
			Action: func(cliCtx *cli.Context) error {
				var rollupCfg *rollup.Config
				if cliCtx.IsSet("l2-chain-id") {
					l2ChainID := new(big.Int).SetUint64(cliCtx.Uint64("l2-chain-id"))
					cfg, err := rollup.LoadOPStackRollupConfig(l2ChainID.Uint64())
					if err != nil {
						return err
					}
					rollupCfg = cfg
				} else if cliCtx.IsSet("rollup-config") {
					f, err := os.Open(cliCtx.String("rollup-config"))
					if err != nil {
						return err
					}
					defer f.Close()
					if err := json.NewDecoder(f).Decode(&rollupCfg); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("either --l2-chain-id or --rollup-config must be set")
				}

				config := reassemble.Config{
					BatchInbox:    rollupCfg.BatchInboxAddress,
					InDirectory:   cliCtx.String("in"),
					OutDirectory:  cliCtx.String("out"),
					L2ChainID:     rollupCfg.L2ChainID,
					L2GenesisTime: rollupCfg.Genesis.L2Time,
					L2BlockTime:   rollupCfg.BlockTime,
				}
				reassemble.Channels(config, rollupCfg)
				return nil
			},
		},

		{
			Name:  "force-close",
			Usage: "Create the tx data which will force close a channel",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "id",
					Required: true,
					Usage:    "ID of the channel to close",
				},
				&cli.StringFlag{
					Name:  "inbox",
					Value: "0x0000000000000000000000000000000000000000",
					Usage: "(Optional) Batch Inbox Address",
				},
				&cli.StringFlag{
					Name:  "in",
					Value: "/tmp/batch_decoder/transactions_cache",
					Usage: "Cache directory for the found transactions",
				},
			},
			Action: func(cliCtx *cli.Context) error {
				var id derive.ChannelID
				if err := (&id).UnmarshalText([]byte(cliCtx.String("id"))); err != nil {
					log.Fatal(err)
				}
				frames := reassemble.LoadFrames(cliCtx.String("in"), common.HexToAddress(cliCtx.String("inbox")))
				var filteredFrames []derive.Frame
				for _, frame := range frames {
					if frame.Frame.ID == id {
						filteredFrames = append(filteredFrames, frame.Frame)
					}
				}
				data, err := derive.ForceCloseTxData(filteredFrames)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("%x\n", data)
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
