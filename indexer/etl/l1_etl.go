package etl

import (
	"context"
	"errors"
	"reflect"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L1ETL struct {
	ETL

	db *database.DB
}

func NewL1ETL(log log.Logger, db *database.DB, client node.EthClient, contracts config.L1Contracts) (*L1ETL, error) {
	log = log.New("etl", "l1")

	contractValue := reflect.ValueOf(contracts)
	fields := reflect.VisibleFields(reflect.TypeOf(contracts))
	l1Contracts := make([]common.Address, len(fields))
	for i, field := range fields {
		// ruleid: unsafe-reflect-by-name
		addr, ok := (contractValue.FieldByName(field.Name).Interface()).(common.Address)
		if !ok {
			log.Error("non-address found in L1Contracts", "name", field.Name)
			return nil, errors.New("non-address found in L1Contracts")
		}

		log.Info("configured contract", "name", field.Name, "addr", addr)
		l1Contracts[i] = addr
	}

	latestHeader, err := db.Blocks.L1LatestBlockHeader()
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
		contracts:       l1Contracts,
		etlBatches:      etlBatches,
	}

	return &L1ETL{ETL: etl, db: db}, nil
}

func (l1Etl *L1ETL) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- l1Etl.ETL.Start(ctx)
	}()

	for {
		select {
		case err := <-errCh:
			return err

		// Index incoming batches
		case batch := <-l1Etl.etlBatches:
			// Pull out only L1 blocks that have emitted a log ( <= batch.Headers )
			l1BlockHeaders := make([]database.L1BlockHeader, 0, len(batch.Headers))
			for i := range batch.Headers {
				if _, ok := batch.HeadersWithLog[batch.Headers[i].Hash()]; ok {
					l1BlockHeaders = append(l1BlockHeaders, database.L1BlockHeader{BlockHeader: database.BlockHeaderFromHeader(&batch.Headers[i])})
				}
			}

			if len(l1BlockHeaders) == 0 {
				batch.Logger.Info("no l1 blocks with logs in batch")
				continue
			}

			l1ContractEvents := make([]database.L1ContractEvent, len(batch.Logs))
			for i := range batch.Logs {
				timestamp := batch.HeaderMap[batch.Logs[i].BlockHash].Time
				l1ContractEvents[i] = database.L1ContractEvent{ContractEvent: database.ContractEventFromLog(&batch.Logs[i], timestamp)}
			}

			// Continually try to persist this batch. If it fails after 10 attempts, we simply error out
			retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
			_, err := retry.Do[interface{}](ctx, 10, retryStrategy, func() (interface{}, error) {
				err := l1Etl.db.Transaction(func(tx *database.DB) error {
					if err := tx.Blocks.StoreL1BlockHeaders(l1BlockHeaders); err != nil {
						return err
					}

					// we must have logs if we have l1 blocks
					if err := tx.ContractEvents.StoreL1ContractEvents(l1ContractEvents); err != nil {
						return err
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
