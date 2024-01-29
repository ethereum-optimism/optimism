package main

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var pullCommand = &cli.Command{
	Name:   "pull",
	Usage:  "Pull a range of blocks and extract nonces from all user deposits",
	Flags:  []cli.Flag{FirstFlag, LastFlag, RPCURLFlag, WorkerFlag, OutputFlag},
	Action: pull,
}

// run starts the cli "pull"
// it will start a number of workers to process blocks
// and runs an aggregation to collect the results
func pull(ctx *cli.Context) error {
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

	// record start time
	start := time.Now()

	resultChan := make(chan result)
	errorChan := make(chan error)

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
		go startWorker(id, workers, ctx, c, resultChan, errorChan, log, wg)
	}

	// start a worker-waiter to end the aggregation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		log.Info("All workers finished")
		done <- struct{}{}
	}()

	// aggregate until the done signal is received
	aggregateResults, err := startAggregator(resultChan, errorChan, done, log)
	if err != nil {
		log.Error("failed to build aggregate results", "err", err)
		return err
	}
	aggregateResults.First = first
	aggregateResults.Last = last

	err = writeJSON(aggregateResults, ctx.String("output"))
	if err != nil {
		log.Error("failed to write aggregate results", "err", err)
		return err
	}

	log.Info("Aggregated Results", "results", aggregateResults)

	log.Info("Finished", "numBlocks", last-first, "duration", time.Since(start))

	return nil
}

// startAggregator will aggregate the results of the workers and return the aggregation once done
func startAggregator(results chan result, errorChan chan error, done chan struct{}, log log.Logger) (aggregate, error) {
	aggregateResults := aggregate{
		Results: make(map[uint64][]uint64),
	}
	var errs error
	for {
		select {
		case r := <-results:
			if len(r.Nonces) > 0 {
				log.Info("Block has deposit transactions", "block", r.BlockNumber, "nonces", r.Nonces)
				aggregateResults.Results[r.BlockNumber] = r.Nonces
			}
		case err := <-errorChan:
			log.Error("Got Error", "err", err)
			errs = errors.Join(errs, err)
		case <-done:
			return aggregateResults, errs
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
			BlockNumber: i,
			Nonces:      ns,
		}
	}
}

type ByHash []*types.Transaction

func (a ByHash) Len() int           { return len(a) }
func (a ByHash) Less(i, j int) bool { return a[i].Hash().Cmp(a[j].Hash()) < 0 }
func (a ByHash) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// processBlock will investigate the transaction of a block and return the nonces of all user deposits which pass checks
func processBlock(ctx context.Context, c *ethclient.Client, blockNumber uint64, log log.Logger) ([]uint64, error) {
	ns := []uint64{}

	block, err := c.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		log.Error("failed to get block", "err", err)
		return []uint64{}, err
	}

	txCount := 0
	matches := 0
	// sort the transactions by hash
	// readers of the data will need to sort to match
	sort.Sort(ByHash(block.Transactions()))
	for _, tx := range block.Transactions() {
		ok, err := checkTransaction(ctx, c, *tx, log)
		if err != nil {
			log.Error("failed to check tx", "err", err)
			return []uint64{}, err
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
