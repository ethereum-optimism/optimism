package indexer

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/etl"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/indexer/processors"
)

// Indexer contains the necessary resources for
// indexing the configured L1 and L2 chains
type Indexer struct {
	db  *database.DB
	log log.Logger

	L1ETL *etl.L1ETL
	L2ETL *etl.L2ETL

	BridgeProcessor *processors.BridgeProcessor
}

// NewIndexer initializes an instance of the Indexer
func NewIndexer(logger log.Logger, chainConfig config.ChainConfig, rpcsConfig config.RPCsConfig, db *database.DB) (*Indexer, error) {
	l1EthClient, err := node.DialEthClient(rpcsConfig.L1RPC)
	if err != nil {
		return nil, err
	}

	l1Etl, err := etl.NewL1ETL(logger, db, l1EthClient, chainConfig.L1Contracts)
	if err != nil {
		return nil, err
	}

	l2EthClient, err := node.DialEthClient(rpcsConfig.L2RPC)
	if err != nil {
		return nil, err
	}

	// Currently defaults to the predeploys
	l2Etl, err := etl.NewL2ETL(logger, db, l2EthClient)
	if err != nil {
		return nil, err
	}

	bridgeProcessor, err := processors.NewBridgeProcessor(logger, db, chainConfig)
	if err != nil {
		return nil, err
	}

	indexer := &Indexer{
		db:  db,
		log: logger,

		L1ETL:           l1Etl,
		L2ETL:           l2Etl,
		BridgeProcessor: bridgeProcessor,
	}

	return indexer, nil
}

// Start starts the indexing service on L1 and L2 chains
func (i *Indexer) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	// If either processor errors out, we stop
	subCtx, cancel := context.WithCancel(ctx)
	run := func(start func(ctx context.Context) error) {
		wg.Add(1)
		defer func() {
			if err := recover(); err != nil {
				i.log.Error("halting indexer on panic", "err", err)
				debug.PrintStack()
				errCh <- fmt.Errorf("panic: %v", err)
			}

			cancel()
			wg.Done()
		}()

		err := start(subCtx)
		if err != nil {
			i.log.Error("halting indexer on error", "err", err)
		}

		// Send a value down regardless if we've received an error
		// or halted via cancellation where err == nil
		errCh <- err
	}

	// Kick off all the dependent routines
	go run(i.L1ETL.Start)
	go run(i.L2ETL.Start)
	go run(i.BridgeProcessor.Start)
	err := <-errCh

	wg.Wait()
	i.log.Info("indexer stopped")
	return err
}

// Cleanup releases any resources that might be currently held by the indexer
func (i *Indexer) Cleanup() {
	i.db.Close()
}
