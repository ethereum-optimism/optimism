package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
)

// TODO(tynes): handle connecting directly to a LevelDB based StateDB
func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "withdrawals",
		Usage: "fetches all pending withdrawals",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "l1-rpc-url",
				Value: "http://127.0.0.1:8545",
				Usage: "RPC URL for an L1 Node",
			},
			&cli.StringFlag{
				Name:  "l2-rpc-url",
				Value: "http://127.0.0.1:9545",
				Usage: "RPC URL for an L2 Node",
			},
			&cli.StringFlag{
				Name:  "l1-cross-domain-messenger-address",
				Usage: "Address of the L1CrossDomainMessenger",
			},
			&cli.Uint64Flag{
				Name:  "start",
				Usage: "Start height to search for events",
			},
			&cli.Uint64Flag{
				Name:  "end",
				Usage: "End height to search for events",
			},
			&cli.StringFlag{
				Name:  "outfile",
				Usage: "Path to output file",
				Value: "out.json",
			},
		},
		Action: func(ctx *cli.Context) error {
			l1RpcURL := ctx.String("l1-rpc-url")
			l2RpcURL := ctx.String("l2-rpc-url")

			l1Client, err := ethclient.Dial(l1RpcURL)
			if err != nil {
				return err
			}
			l2Client, err := ethclient.Dial(l2RpcURL)
			if err != nil {
				return err
			}

			backends := crossdomain.NewBackends(l1Client, l2Client)

			l1xDomainMessenger := ctx.String("l1-cross-domain-messenger-address")
			if l1xDomainMessenger == "" {
				return errors.New("Must pass in L1CrossDomainMessenger address")
			}

			l1xDomainMessengerAddr := common.HexToAddress(l1xDomainMessenger)
			messengers, err := crossdomain.NewMessengers(backends, l1xDomainMessengerAddr)
			if err != nil {
				return err
			}

			start := ctx.Uint64("start")
			end := ctx.Uint64("end")

			// All messages are expected to be version 0 messages
			withdrawals, err := crossdomain.GetPendingWithdrawals(messengers, common.Big0, start, end)
			if err != nil {
				return err
			}

			outfile := ctx.String("outfile")
			if err := writeJSONFile(outfile, withdrawals); err != nil {
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
