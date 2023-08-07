package indexer

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/indexer/processor"
)

// Indexer contains the necessary resources for
// indexing the configured L1 and L2 chains
type Indexer struct {
	db  *database.DB
	log log.Logger

	L1Processor *processor.L1Processor
	L2Processor *processor.L2Processor
}

// NewIndexer initializes an instance of the Indexer
func NewIndexer(cfg config.Config) (*Indexer, error) {
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=disable", cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)
	if cfg.DB.User != "" {
		dsn += fmt.Sprintf(" user=%s", cfg.DB.User)
	}
	if cfg.DB.Password != "" {
		dsn += fmt.Sprintf(" password=%s", cfg.DB.Password)
	}

	db, err := database.NewDB(dsn)
	if err != nil {
		return nil, err
	}

	l1Contracts := cfg.Chain.L1Contracts
	l1EthClient, err := node.DialEthClient(cfg.RPCs.L1RPC)
	if err != nil {
		return nil, err
	}
	l1Processor, err := processor.NewL1Processor(cfg.Logger, l1EthClient, db, l1Contracts)
	if err != nil {
		return nil, err
	}

	// L2Processor (predeploys). Although most likely the right setting, make this configurable?
	l2Contracts := processor.L2ContractPredeploys()
	l2EthClient, err := node.DialEthClient(cfg.RPCs.L2RPC)
	if err != nil {
		return nil, err
	}
	l2Processor, err := processor.NewL2Processor(cfg.Logger, l2EthClient, db, l2Contracts)
	if err != nil {
		return nil, err
	}

	indexer := &Indexer{
		db:          db,
		log:         cfg.Logger,
		L1Processor: l1Processor,
		L2Processor: l2Processor,
	}

	return indexer, nil
}

// Start starts the indexing service on L1 and L2 chains
func (i *Indexer) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error)

	// If either processor errors out, we stop
	processorCtx, cancel := context.WithCancel(ctx)
	run := func(start func(ctx context.Context) error) {
		wg.Add(1)
		defer wg.Done()

		err := start(processorCtx)
		if err != nil {
			i.log.Error("halting indexer on error", "err", err)

			cancel()
			errCh <- err
		}
	}

	// Kick off the processors
	go run(i.L1Processor.Start)
	go run(i.L2Processor.Start)
	err := <-errCh

	// ensure both processors have halted before returning
	wg.Wait()
	return err
}

// Cleanup releases any resources that might be currently held by the indexer
func (i *Indexer) Cleanup() {
	i.db.Close()
}
