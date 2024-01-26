package main

import (
	"context"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
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
		{
			Name:   "pull",
			Usage:  "Pull a range of blocks and extract nonces from all user deposits",
			Flags:  []cli.Flag{FirstFlag, LastFlag, RPCURLFlag, WorkerFlag},
			Action: run,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("critical error", "err", err)
	}
}

type result struct {
	blockNumber uint64
	nonces      nonces
}
type nonces []uint64

// run starts the cli tool
// it will start a number of workers to process blocks
// and runs an aggregation to collect the results
func run(ctx *cli.Context) error {
	timeout := 1 * time.Minute
	log := log.New()
	// create a new client
	c, err := dial.DialEthClientWithTimeout(
		ctx.Context,
		timeout,
		log,
		ctx.String("rpc-url"),
	)
	if err != nil {
		log.Error("failed to dial rollup client", "err", err)
		return err
	}

	results := make(chan result)
	errors := make(chan error)

	first := ctx.Uint64("first")
	last := ctx.Uint64("last")
	workers := ctx.Uint64("workers")
	log.Info("Starting", "first", first, "last", last, "workers", workers)
	if workers > last-first {
		log.Info("more workers than required", "workers", workers, "blocks", last-first)
		workers = last - first
	}

	// start workers
	wg := &sync.WaitGroup{}
	for id := uint64(0); id < workers; id++ {
		wg.Add(1)
		go startWorker(id, workers, ctx, c, results, errors, log, wg)
	}

	// start a worker-waiter to end the aggregation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		log.Info("All workers finished")
		done <- struct{}{}
	}()

	// aggregate until the done signal is received
	aggregateResults := startAggregator(results, errors, done, log)

	log.Info("Aggregated Results", "results", aggregateResults)

	return nil
}

// startAggregator will aggregate the results of the workers and return the aggregation once done
func startAggregator(results chan result, errors chan error, done chan struct{}, log log.Logger) []result {
	aggregateResults := []result{}
	for {
		select {
		case r := <-results:
			if len(r.nonces) > 0 {
				log.Info("Block has deposit transactions", "block", r.blockNumber, "nonces", r.nonces)
				aggregateResults = append(aggregateResults, r)
			}
		case err := <-errors:
			log.Error("Got Error", "err", err)
		case <-done:
			return aggregateResults
		}
	}
}

// startWorker will start a worker to process blocks
// workers process blocks and return the results to a channel for aggregation
// workers use their ID to determine which blocks to process
func startWorker(id uint64,
	workers uint64,
	ctx *cli.Context,
	c *ethclient.Client,
	results chan result,
	errors chan error,
	log log.Logger,
	wg *sync.WaitGroup) {
	defer wg.Done()
	log.Info("Starting Worker", "id", id)
	for i := ctx.Uint64("first") + id; i <= ctx.Uint64("last")+1; i += workers {
		ns, err := processBlock(ctx.Context, c, i, log)
		// TODO: put a retry here
		if err != nil {
			log.Error("failed to process block", "err", err)
			errors <- err
			return
		}
		results <- result{
			blockNumber: i,
			nonces:      ns,
		}
	}
}

// processBlock will investigate the transaction of a block and return the nonces of all user deposits which pass checks
func processBlock(ctx context.Context, c *ethclient.Client, blockNumber uint64, log log.Logger) (nonces, error) {
	ns := make(nonces, 0)

	block, err := c.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		log.Error("failed to get block", "err", err)
		return nonces{}, err
	}

	txCount := 0
	matches := 0
	for _, tx := range block.Transactions() {
		ok, err := checkTransaction(ctx, c, tx, log)
		if err != nil {
			log.Error("failed to check tx", "err", err)
			return nonces{}, err
		}
		txCount += 1
		if ok {
			matches += 1
			ns = append(ns, tx.Nonce())
		}
	}

	log.Info("Processed Block", "block", blockNumber, "txCount", txCount, "userDeposits", matches)
	return ns, nil
}

// checkTransaction will check if a transaction is a user deposit, and not initiated by the system address
func checkTransaction(ctx context.Context, c *ethclient.Client, tx types.Transaction, log log.Logger) (bool, error) {
	from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), &tx)
	if err != nil {
		log.Error("failed to get sender from tx", "err", err)
		return false, err
	}

	if from == systemAddress ||
		tx.Type() != depositType {
		return false, nil
	}
	log.Info("Got Transaction", "from", from, "nonce", tx.Nonce(), "type", tx.Type())
	return true, nil
}
