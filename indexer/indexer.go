package indexer

import (
	"context"
	"fmt"
	"math/big"
	"runtime/debug"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/etl"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/indexer/processors"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

// Indexer contains the necessary resources for
// indexing the configured L1 and L2 chains
type Indexer struct {
	log log.Logger
	db  *database.DB

	metricsConfig   config.MetricsConfig
	metricsRegistry *prometheus.Registry

	L1ETL           *etl.L1ETL
	L2ETL           *etl.L2ETL
	BridgeProcessor *processors.BridgeProcessor
}

// NewIndexer initializes an instance of the Indexer
func NewIndexer(logger log.Logger, db *database.DB, chainConfig config.ChainConfig, rpcsConfig config.RPCsConfig, metricsConfig config.MetricsConfig) (*Indexer, error) {
	metricsRegistry := metrics.NewRegistry()

	// L1
	l1EthClient, err := node.DialEthClient(rpcsConfig.L1RPC, node.NewMetrics(metricsRegistry, "l1"))
	if err != nil {
		return nil, err
	}
	l1Cfg := etl.Config{
		LoopIntervalMsec:  chainConfig.L1PollingInterval,
		HeaderBufferSize:  chainConfig.L1HeaderBufferSize,
		ConfirmationDepth: big.NewInt(int64(chainConfig.L1ConfirmationDepth)),
		StartHeight:       big.NewInt(int64(chainConfig.L1StartingHeight)),
	}
	l1Etl, err := etl.NewL1ETL(l1Cfg, logger, db, etl.NewMetrics(metricsRegistry, "l1"), l1EthClient, chainConfig.L1Contracts)
	if err != nil {
		return nil, err
	}

	// L2 (defaults to predeploy contracts)
	l2EthClient, err := node.DialEthClient(rpcsConfig.L2RPC, node.NewMetrics(metricsRegistry, "l2"))
	if err != nil {
		return nil, err
	}
	l2Cfg := etl.Config{
		LoopIntervalMsec:  chainConfig.L2PollingInterval,
		HeaderBufferSize:  chainConfig.L2HeaderBufferSize,
		ConfirmationDepth: big.NewInt(int64(chainConfig.L2ConfirmationDepth)),
	}
	l2Etl, err := etl.NewL2ETL(l2Cfg, logger, db, etl.NewMetrics(metricsRegistry, "l2"), l2EthClient)
	if err != nil {
		return nil, err
	}

	// Bridge
	bridgeProcessor, err := processors.NewBridgeProcessor(logger, db, l1Etl, chainConfig)
	if err != nil {
		return nil, err
	}

	indexer := &Indexer{
		log: logger,
		db:  db,

		metricsConfig:   metricsConfig,
		metricsRegistry: metricsRegistry,

		L1ETL:           l1Etl,
		L2ETL:           l2Etl,
		BridgeProcessor: bridgeProcessor,
	}

	return indexer, nil
}

func (i *Indexer) startMetricsServer(ctx context.Context) error {
	i.log.Info("starting metrics server...", "port", i.metricsConfig.Port)
	err := metrics.ListenAndServe(ctx, i.metricsRegistry, i.metricsConfig.Host, i.metricsConfig.Port)
	if err != nil {
		i.log.Error("metrics server stopped", "err", err)
	} else {
		i.log.Info("metrics server stopped")
	}

	return err
}

// Start starts the indexing service on L1 and L2 chains
func (i *Indexer) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	// if any goroutine halts, we stop the entire indexer
	processCtx, processCancel := context.WithCancel(ctx)
	runProcess := func(start func(ctx context.Context) error) {
		wg.Add(1)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					i.log.Error("halting indexer on panic", "err", err)
					debug.PrintStack()
					errCh <- fmt.Errorf("panic: %v", err)
				}

				processCancel()
				wg.Done()
			}()

			errCh <- start(processCtx)
		}()
	}

	// Kick off all the dependent routines
	runProcess(i.L1ETL.Start)
	runProcess(i.L2ETL.Start)
	runProcess(i.BridgeProcessor.Start)
	runProcess(i.startMetricsServer)
	wg.Wait()

	err := <-errCh
	if err != nil {
		i.log.Error("indexer stopped", "err", err)
	} else {
		i.log.Info("indexer stopped")
	}

	return err
}
