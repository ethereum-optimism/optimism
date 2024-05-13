package etl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
)

type L1ETL struct {
	ETL
	latestHeader *types.Header

	cfg Config

	// the batch handler may do work that we can interrupt on shutdown
	resourceCtx    context.Context
	resourceCancel context.CancelFunc

	tasks tasks.Group

	db *database.DB

	mu        sync.Mutex
	listeners []chan *types.Header
}

// NewL1ETL creates a new L1ETL instance that will start indexing from different starting points
// depending on the state of the database and the supplied start height.
func NewL1ETL(cfg Config, log log.Logger, db *database.DB, metrics Metricer, client client.Client,
	contracts config.L1Contracts, shutdown context.CancelCauseFunc) (*L1ETL, error) {
	log = log.New("etl", "l1")

	zeroAddr := common.Address{}
	l1Contracts := []common.Address{}
	if err := contracts.ForEach(func(name string, addr common.Address) error {
		// Since we dont have backfill support yet, we want to make sure all expected
		// contracts are specified to ensure consistent behavior. Once backfill support
		// is ready, we can relax this requirement.
		if addr == zeroAddr && !strings.HasPrefix(name, "Legacy") {
			log.Error("address not configured", "name", name)
			return errors.New("all L1Contracts must be configured")
		}

		log.Info("configured contract", "name", name, "addr", addr)
		l1Contracts = append(l1Contracts, addr)
		return nil
	}); err != nil {
		return nil, err
	}

	latestHeader, err := db.Blocks.L1LatestBlockHeader()
	if err != nil {
		return nil, err
	}

	// Determine the starting height for traversal
	var fromHeader *types.Header
	if latestHeader != nil {
		log.Info("detected last indexed block", "number", latestHeader.Number, "hash", latestHeader.Hash)
		fromHeader = latestHeader.RLPHeader.Header()
	} else if cfg.StartHeight.BitLen() > 0 {
		log.Info("no indexed state starting from supplied L1 height", "height", cfg.StartHeight.String())

		ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
		defer cancel()

		header, err := client.HeaderByNumber(ctxwt, cfg.StartHeight)
		if err != nil {
			return nil, fmt.Errorf("could not fetch starting block header: %w", err)
		}

		fromHeader = header
	} else {
		log.Info("no indexed state, starting from genesis")
	}

	// NOTE - The use of un-buffered channel here assumes that downstream consumers
	// will be able to keep up with the rate of incoming batches.
	// When the producer closes the channel we stop consuming from it.
	etlBatches := make(chan *ETLBatch)

	etl := ETL{
		loopInterval:     time.Duration(cfg.LoopIntervalMsec) * time.Millisecond,
		headerBufferSize: uint64(cfg.HeaderBufferSize),

		log:             log,
		metrics:         metrics,
		headerTraversal: node.NewHeaderTraversal(client, fromHeader, cfg.ConfirmationDepth),
		contracts:       l1Contracts,
		etlBatches:      etlBatches,

		client: client,
	}

	resCtx, resCancel := context.WithCancel(context.Background())
	return &L1ETL{
		ETL:          etl,
		latestHeader: fromHeader,

		cfg: cfg,

		db:             db,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in L1 ETL: %w", err))
		}},
	}, nil
}

func (l1Etl *L1ETL) Close() error {
	var result error
	// close the producer
	if err := l1Etl.ETL.Close(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to close internal ETL: %w", err))
	}
	// tell the consumer it can stop what it's doing
	l1Etl.resourceCancel()
	// wait for consumer to pick up on closure of producer
	if err := l1Etl.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await batch handler completion: %w", err))
	}
	// close listeners
	for i := range l1Etl.listeners {
		close(l1Etl.listeners[i])
	}
	return result
}

func (l1Etl *L1ETL) Start() error {
	l1Etl.log.Info("starting etl...")

	// start ETL batch producer
	if err := l1Etl.ETL.Start(); err != nil {
		return fmt.Errorf("failed to start internal ETL: %w", err)
	}
	// start ETL batch consumer
	l1Etl.tasks.Go(func() error {
		for batch := range l1Etl.etlBatches {
			if err := l1Etl.handleBatch(batch); err != nil {
				return fmt.Errorf("failed to handle batch, stopping L2 ETL: %w", err)
			}
		}
		l1Etl.log.Info("no more batches, shutting down batch handler")
		return nil
	})
	return nil
}

func (l1Etl *L1ETL) handleBatch(batch *ETLBatch) error {
	// Index incoming batches (only L1 blocks that have an emitted log)
	l1BlockHeaders := make([]database.L1BlockHeader, 0, len(batch.Headers))
	for i := range batch.Headers {
		if _, ok := batch.HeadersWithLog[batch.Headers[i].Hash()]; ok {
			l1BlockHeaders = append(l1BlockHeaders, database.L1BlockHeader{BlockHeader: database.BlockHeaderFromHeader(&batch.Headers[i])})
		}
	}

	// If there has been no activity and the inactivity window has elapsed since the last header, index the latest
	// block to remediate any false-stall alerts on downstream processors that rely on indexed state.
	if l1Etl.cfg.AllowedInactivityWindowSeconds > 0 && len(l1BlockHeaders) == 0 {
		latestHeader := batch.Headers[len(batch.Headers)-1]
		if l1Etl.latestHeader == nil || latestHeader.Time-l1Etl.latestHeader.Time > uint64(l1Etl.cfg.AllowedInactivityWindowSeconds) {
			l1BlockHeaders = append(l1BlockHeaders, database.L1BlockHeader{BlockHeader: database.BlockHeaderFromHeader(&latestHeader)})
		}
	}

	l1ContractEvents := make([]database.L1ContractEvent, len(batch.Logs))
	for i := range batch.Logs {
		timestamp := batch.HeaderMap[batch.Logs[i].BlockHash].Time
		l1ContractEvents[i] = database.L1ContractEvent{ContractEvent: database.ContractEventFromLog(&batch.Logs[i], timestamp)}
		l1Etl.ETL.metrics.RecordIndexedLog(batch.Logs[i].Address)
	}

	// Continually try to persist this batch. If it fails after 10 attempts, we simply error out
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	if _, err := retry.Do[interface{}](l1Etl.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
		if len(l1BlockHeaders) == 0 {
			return nil, nil // skip
		}

		if err := l1Etl.db.Transaction(func(tx *database.DB) error {
			if err := tx.Blocks.StoreL1BlockHeaders(l1BlockHeaders); err != nil {
				return err
			}
			// we must have logs if we have l1 blocks
			if len(l1ContractEvents) > 0 {
				if err := tx.ContractEvents.StoreL1ContractEvents(l1ContractEvents); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			batch.Logger.Error("unable to persist batch", "err", err)
			return nil, fmt.Errorf("unable to persist batch: %w", err)
		}

		// a-ok!
		return nil, nil
	}); err != nil {
		return err
	}

	if len(l1BlockHeaders) == 0 {
		batch.Logger.Info("skipped batch. no logs found")
	} else {
		batch.Logger.Info("indexed batch")
	}

	// Since not every L1 block is indexed, we still want our metrics to cover L1 blocks
	// that have been observed so that a false stall alert isn't triggered on low activity
	l1Etl.latestHeader = &batch.Headers[len(batch.Headers)-1]
	l1Etl.ETL.metrics.RecordIndexedHeaders(len(l1BlockHeaders))
	l1Etl.ETL.metrics.RecordEtlLatestHeight(l1Etl.latestHeader.Number)

	// Notify Listeners
	l1Etl.mu.Lock()
	defer l1Etl.mu.Unlock()
	for i := range l1Etl.listeners {
		select {
		case l1Etl.listeners[i] <- l1Etl.latestHeader:
		default:
			// do nothing if the listener hasn't picked
			// up the previous notif
		}
	}

	return nil
}

// Notify returns a channel that'll receive the latest header when new data has been persisted
func (l1Etl *L1ETL) Notify() <-chan *types.Header {
	receiver := make(chan *types.Header)
	l1Etl.mu.Lock()
	defer l1Etl.mu.Unlock()

	l1Etl.listeners = append(l1Etl.listeners, receiver)
	return receiver
}
