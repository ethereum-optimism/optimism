package etl

import (
	"context"

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

func NewL2ETL(log log.Logger, db *database.DB, client node.EthClient) (*L2ETL, error) {
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
		log.Info("detected last indexed block", "number", latestHeader.Number.Int, "hash", latestHeader.Hash)
		fromHeader = latestHeader.RLPHeader.Header()
	} else {
		log.Info("no indexed state, starting from genesis")
	}

	etlBatches := make(chan ETLBatch)
	etl := ETL{
		log:             log,
		headerTraversal: node.NewHeaderTraversal(client, fromHeader),
		ethClient:       client.GethEthClient(),
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

		// Index incoming batches
		case batch := <-l2Etl.etlBatches:
			// We're indexing every L2 block.
			l2BlockHeaders := make([]database.L2BlockHeader, len(batch.Headers))
			for i := range batch.Headers {
				l2BlockHeaders[i] = database.L2BlockHeader{BlockHeader: database.BlockHeaderFromHeader(&batch.Headers[i])}
			}

			l2ContractEvents := make([]database.L2ContractEvent, len(batch.Logs))
			for i := range batch.Logs {
				timestamp := batch.HeaderMap[batch.Logs[i].BlockHash].Time
				l2ContractEvents[i] = database.L2ContractEvent{ContractEvent: database.ContractEventFromLog(&batch.Logs[i], timestamp)}
			}

			// Continually try to persist this batch. If it fails after 5 attempts, we simply error out
			retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
			_, err := retry.Do[interface{}](ctx, 10, retryStrategy, func() (interface{}, error) {
				err := l2Etl.db.Transaction(func(tx *database.DB) error {
					if err := tx.Blocks.StoreL2BlockHeaders(l2BlockHeaders); err != nil {
						return err
					}
					if len(l2ContractEvents) > 0 {
						if err := tx.ContractEvents.StoreL2ContractEvents(l2ContractEvents); err != nil {
							return err
						}
					}
					return nil
				})

				if err != nil {
					batch.Logger.Error("unable to persist batch", "err", err)
					return nil, err
				}

				// a-ok! Can merge with the above block but being explicit
				return nil, nil
			})

			if err != nil {
				return err
			}

			batch.Logger.Info("indexed batch")
		}
	}
}
