package main

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const EnvPrefix = "OP_CHAIN_OPS_PROTOCOL_VERSION"

var (
	FirstFlag = &cli.Uint64Flag{
		Name:    "first",
		Value:   0,
		Usage:   "the first block to include in data collection. INCLUSIVE",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "FIRST"),
	}
	LastFlag = &cli.Uint64Flag{
		Name:    "last",
		Value:   0,
		Usage:   "the last block to include in data collection. INCLUSIVE",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "LAST"),
	}
	RPCURLFlag = &cli.StringFlag{
		Name:    "rpc-url",
		Usage:   "RPC URL to connect to",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "RPC_URL"),
	}
	WorkerFlag = &cli.Uint64Flag{
		Name:    "workers",
		Value:   1,
		Usage:   "how many workers to use to fetch txs",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "WORKERS"),
	}
	OutputFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "the file to write the results to",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "OUTPUT"),
	}
	FilesFlag = &cli.StringSliceFlag{
		Name:    "files",
		Aliases: []string{"f"},
		Usage:   "the set of files to merge",
	}
	systemAddress = common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001")
	depositType   = uint8(126)
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:   "receipt-reference-builder",
		Usage:  "Used to generate reference data for deposit receipts of pre-canyon blocks",
		Flags:  []cli.Flag{},
		Writer: os.Stdout,
	}

	app.Commands = []*cli.Command{
		pullCommand,
		mergeCommand,
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
	First   uint64              `json:"first"`
	Last    uint64              `json:"last"`
}
