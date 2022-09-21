package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "storage",
		Usage: "fetches all storage slots for a given contract",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "rpc-url",
				Value: "http://127.0.0.1:8545",
				Usage: "RPC URL for the node to dump storage from",
			},
			&cli.StringFlag{
				Name:  "contract",
				Usage: "Address of the contract to dump storage from",
			},
			&cli.StringFlag{
				Name:  "blockhash",
				Value: "latest",
				Usage: "Block hash to dump storage from",
			},
			&cli.StringFlag{
				Name:  "outfile",
				Usage: "Path to output file",
				Value: "out.json",
			},
		},
		Action: func(ctx *cli.Context) error {
			rpcCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()

			rpcURL := ctx.String("rpc-url")

			eth, err := ethclient.Dial(rpcURL)
			if err != nil {
				return err
			}

			client, err := rpc.DialContext(rpcCtx, rpcURL)
			if err != nil {
				return err
			}

			contract := ctx.String("contract")
			if contract == "" {
				return errors.New("must pass in contract address")
			}

			blockhash := ctx.String("blockhash")
			if blockhash == "latest" {
				block, err := eth.BlockByNumber(rpcCtx, nil)
				if err != nil {
					return err
				}

				blockhash = block.Hash().Hex()
			}

			var storage []state.EncodedStorage
			var nextResult state.StorageRangeResult
			for ok := true; ok; ok = nextResult.NextKey != nil {
				var nextKey string
				if nextResult.NextKey != nil {
					nextKey = nextResult.NextKey.Hex()
				} else {
					nextKey = "0x"
				}

				err = client.Call(&nextResult, "debug_storageRangeAt", blockhash, 0, common.HexToAddress(contract), nextKey, 255)
				if err != nil {
					return err
				}

				for _, value := range nextResult.Storage {
					storage = append(storage, value)
				}
			}

			outfile := ctx.String("outfile")
			if err := writeJSONFile(outfile, storage); err != nil {
				return err
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}

func writeJSONFile(outfile string, input interface{}) error {
	f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(input)
}
