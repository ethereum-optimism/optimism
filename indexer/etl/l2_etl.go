package etl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L2ETL struct {
	ETL

	db *database.DB
}

func NewL2ETL(cfg Config, log log.Logger, db *database.DB, metrics Metricer, client node.EthClient) (*L2ETL, error) {
	log = log.New("etl", "l2")

	// allow predeploys to be overridable
	l2Contracts := []common.Address{}
	for name, addr := range predeploys.Predeploys {
		log.Info("configured contract", "name", name, "addr", addr)
		l2Contracts = append(l2Contracts, *addr)
	}

	latestHeader, err := db.Blocks.L2LatestBlockHeader()
	if err != nil {
		return nil, err
	}

	var fromHeader *types.Header
	if latestHeader != nil {
		log.Info("detected last indexed block", "number", latestHeader.Number, "hash", latestHeader.Hash)
		fromHeader = latestHeader.RLPHeader.Header()
	} else {
		log.Info("no indexed state, starting from genesis")
	}

	etlBatches := make(chan ETLBatch)
	etl := ETL{
		loopInterval:     time.Duration(cfg.LoopIntervalMsec) * time.Millisecond,
		headerBufferSize: uint64(cfg.HeaderBufferSize),

		log:             log,
		metrics:         metrics,
		headerTraversal: node.NewHeaderTraversal(client, fromHeader, cfg.ConfirmationDepth),
		ethClient:       client,
		contracts:       l2Contracts,
		etlBatches:      etlBatches,
	}

	return &L2ETL{ETL: etl, db: db}, nil
}

func (l2Etl *L2ETL) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- l2Etl.ETL.Start(ctx)
	}()

	for {
		select {
		case err := <-errCh:
			return err

		// Index incoming batches (all L2 Blocks)
		case batch := <-l2Etl.etlBatches:
			l2BlockHeaders := make([]database.L2BlockHeader, len(batch.Headers))
			for i := range batch.Headers {
				l2BlockHeaders[i] = database.L2BlockHeader{BlockHeader: database.BlockHeaderFromHeader(&batch.Headers[i])}
			}

			l2ContractEvents := make([]database.L2ContractEvent, len(batch.Logs))
			for i := range batch.Logs {
				timestamp := batch.HeaderMap[batch.Logs[i].BlockHash].Time
				l2ContractEvents[i] = database.L2ContractEvent{ContractEvent: database.ContractEventFromLog(&batch.Logs[i], timestamp)}
			}

			// Continually try to persist this batch. If it fails after 10 attempts, we simply error out
			retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
			if _, err := retry.Do[interface{}](ctx, 10, retryStrategy, func() (interface{}, error) {
				if err := l2Etl.db.Transaction(func(tx *database.DB) error {
					if err := tx.Blocks.StoreL2BlockHeaders(l2BlockHeaders); err != nil {
						return err
					}
					if len(l2ContractEvents) > 0 {
						if err := tx.ContractEvents.StoreL2ContractEvents(l2ContractEvents); err != nil {
							return err
						}
					}
					return nil
				}); err != nil {
					batch.Logger.Error("unable to persist batch", "err", err)
					return nil, err
				}

				l2Etl.ETL.metrics.RecordIndexedHeaders(len(l2BlockHeaders))
				l2Etl.ETL.metrics.RecordIndexedLatestHeight(l2BlockHeaders[len(l2BlockHeaders)-1].Number)
				if len(l2ContractEvents) > 0 {
					l2Etl.ETL.metrics.RecordIndexedLogs(len(l2ContractEvents))
				}

				// a-ok!
				return nil, nil
			}); err != nil {
				return err
			}

			batch.Logger.Info("indexed batch")
		}
	}
}
