package etl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
)

type L2ETL struct {
	ETL
	LatestHeader *types.Header

	// the batch handler may do work that we can interrupt on shutdown
	resourceCtx    context.Context
	resourceCancel context.CancelFunc

	tasks tasks.Group

	db *database.DB

	mu        sync.Mutex
	listeners []chan interface{}
}

func NewL2ETL(cfg Config, log log.Logger, db *database.DB, metrics Metricer, client node.EthClient,
	contracts config.L2Contracts, shutdown context.CancelCauseFunc) (*L2ETL, error) {
	log = log.New("etl", "l2")

	zeroAddr := common.Address{}
	l2Contracts := []common.Address{}
	if err := contracts.ForEach(func(name string, addr common.Address) error {
		// Since we dont have backfill support yet, we want to make sure all expected
		// contracts are specified to ensure consistent behavior. Once backfill support
		// is ready, we can relax this requirement.
		if addr == zeroAddr {
			log.Error("address not configured", "name", name)
			return errors.New("all L2Contracts must be configured")
		}

		log.Info("configured contract", "name", name, "addr", addr)
		l2Contracts = append(l2Contracts, addr)
		return nil
	}); err != nil {
		return nil, err
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

	etlBatches := make(chan *ETLBatch)
	etl := ETL{
		loopInterval:     time.Duration(cfg.LoopIntervalMsec) * time.Millisecond,
		headerBufferSize: uint64(cfg.HeaderBufferSize),

		log:             log,
		metrics:         metrics,
		headerTraversal: node.NewHeaderTraversal(client, fromHeader, cfg.ConfirmationDepth),
		contracts:       l2Contracts,
		etlBatches:      etlBatches,

		EthClient: client,
	}

	resCtx, resCancel := context.WithCancel(context.Background())
	return &L2ETL{
		ETL:          etl,
		LatestHeader: fromHeader,

		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		db:             db,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in L2 ETL: %w", err))
		}},
	}, nil
}

func (l2Etl *L2ETL) Close() error {
	var result error
	// close the producer
	if err := l2Etl.ETL.Close(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to close internal ETL: %w", err))
	}
	// tell the consumer it can stop what it's doing
	l2Etl.resourceCancel()
	// wait for consumer to pick up on closure of producer
	if err := l2Etl.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await batch handler completion: %w", err))
	}
	// close listeners
	for i := range l2Etl.listeners {
		close(l2Etl.listeners[i])
	}
	return result
}

func (l2Etl *L2ETL) Start() error {
	l2Etl.log.Info("starting etl...")

	// start ETL batch producer
	if err := l2Etl.ETL.Start(); err != nil {
		return fmt.Errorf("failed to start internal ETL: %w", err)
	}

	// start ETL batch consumer
	l2Etl.tasks.Go(func() error {
		for batch := range l2Etl.etlBatches {
			if err := l2Etl.handleBatch(batch); err != nil {
				return fmt.Errorf("failed to handle batch, stopping L2 ETL: %w", err)
			}
		}
		l2Etl.log.Info("no more batches, shutting down batch handler")
		return nil
	})
	return nil
}

func (l2Etl *L2ETL) handleBatch(batch *ETLBatch) error {
	l2BlockHeaders := make([]database.L2BlockHeader, len(batch.Headers))
	for i := range batch.Headers {
		l2BlockHeaders[i] = database.L2BlockHeader{BlockHeader: database.BlockHeaderFromHeader(&batch.Headers[i])}
	}

	l2ContractEvents := make([]database.L2ContractEvent, len(batch.Logs))
	for i := range batch.Logs {
		timestamp := batch.HeaderMap[batch.Logs[i].BlockHash].Time
		l2ContractEvents[i] = database.L2ContractEvent{ContractEvent: database.ContractEventFromLog(&batch.Logs[i], timestamp)}
		l2Etl.ETL.metrics.RecordIndexedLog(batch.Logs[i].Address)
	}

	// Continually try to persist this batch. If it fails after 10 attempts, we simply error out
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	if _, err := retry.Do[interface{}](l2Etl.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
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

		// a-ok!
		return nil, nil
	}); err != nil {
		return err
	}

	batch.Logger.Info("indexed batch")
	l2Etl.LatestHeader = &batch.Headers[len(batch.Headers)-1]

	// Notify Listeners
	l2Etl.mu.Lock()
	defer l2Etl.mu.Unlock()
	for i := range l2Etl.listeners {
		select {
		case l2Etl.listeners[i] <- struct{}{}:
		default:
			// do nothing if the listener hasn't picked
			// up the previous notif
		}
	}

	return nil
}

// Notify returns a channel that'll receive a value every time new data has
// been persisted by the L2ETL
func (l2Etl *L2ETL) Notify() <-chan interface{} {
	receiver := make(chan interface{})
	l2Etl.mu.Lock()
	defer l2Etl.mu.Unlock()

	l2Etl.listeners = append(l2Etl.listeners, receiver)
	return receiver
}
