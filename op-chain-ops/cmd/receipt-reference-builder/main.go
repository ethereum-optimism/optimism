package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const EnvPrefix = "OP_CHAIN_OPS_RECEIPT_REFERENCE_BUILDER"

var (
	StartFlag = &cli.Uint64Flag{
		Name:  "start",
		Usage: "the first block to include in data collection. INCLUSIVE",
	}
	EndFlag = &cli.Uint64Flag{
		Name:  "end",
		Usage: "the last block of the collection range. EXCLUSIVE",
	}
	RPCURLFlag = &cli.StringFlag{
		Name:    "rpc-url",
		Usage:   "RPC URL to connect to",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "RPC_URL"),
	}
	BackoffFlag = &cli.DurationFlag{
		Name:  "backoff",
		Value: 30 * time.Second,
		Usage: "how long to wait when a worker errors before retrying",
	}
	WorkerFlag = &cli.Uint64Flag{
		Name:  "workers",
		Value: 1,
		Usage: "how many workers to use to fetch txs",
	}
	BatchSizeFlag = &cli.Uint64Flag{
		Name:  "batch-size",
		Value: 50,
		Usage: "how many blocks to batch together for each worker",
	}
	OutputFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "the file to write the results to",
	}
	FilesFlag = &cli.StringSliceFlag{
		Name:    "files",
		Aliases: []string{"f"},
		Usage:   "the set of files to merge",
	}
	InputFormatFlag = &cli.StringFlag{
		Name:    "input-format",
		Aliases: []string{"if"},
		Value:   "json",
		Usage:   "the format to read aggregate files: json, gob",
	}
	OutputFormatFlag = &cli.StringFlag{
		Name:    "output-format",
		Aliases: []string{"of"},
		Value:   "json",
		Usage:   "the format to write the results in. Options: json, gob",
	}
	formats = map[string]aggregateReaderWriter{
		"json": jsonAggregateReaderWriter{},
		"gob":  gobAggregateReaderWriter{},
	}
	systemAddress = common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001")
	depositType   = uint8(126)
)

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	oplog.SetGlobalLogHandler(log.NewTerminalHandlerWithLevel(os.Stdout, slog.LevelDebug, color))

	app := &cli.App{
		Name:   "receipt-reference-builder",
		Usage:  "Used to generate reference data for deposit receipts of pre-canyon blocks",
		Flags:  []cli.Flag{},
		Writer: os.Stdout,
	}

	app.Commands = []*cli.Command{
		pullCommand,
		mergeCommand,
		convertCommand,
		printCommand,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("critical error", "err", err)
	}
}

type result struct {
	BlockNumber uint64   `json:"blockNumber"`
	Nonces      []uint64 `json:"nonces"`
}

type aggregate struct {
	Results map[uint64][]uint64 `json:"results"`
	ChainID uint64              `json:"chainId"`
	First   uint64              `json:"start"`
	Last    uint64              `json:"end"`
}
