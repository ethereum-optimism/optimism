package etl

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Config struct {
	LoopIntervalMsec uint
	HeaderBufferSize uint

	StartHeight       *big.Int
	ConfirmationDepth *big.Int
}

type ETL struct {
	log     log.Logger
	metrics Metricer

	loopInterval     time.Duration
	headerBufferSize uint64
	headerTraversal  *node.HeaderTraversal

	contracts  []common.Address
	etlBatches chan ETLBatch

	EthClient node.EthClient
}

type ETLBatch struct {
	Logger log.Logger

	Headers   []types.Header
	HeaderMap map[common.Hash]*types.Header

	Logs           []types.Log
	HeadersWithLog map[common.Hash]bool
}

func (etl *ETL) Start(ctx context.Context) error {
	done := ctx.Done()
	pollTicker := time.NewTicker(etl.loopInterval)
	defer pollTicker.Stop()

	// A reference that'll stay populated between intervals
	// in the event of failures in order to retry.
	var headers []types.Header

	etl.log.Info("starting etl...")
	for {
		select {
		case <-done:
			etl.log.Info("stopping etl")
			return nil

		case <-pollTicker.C:
			done := etl.metrics.RecordInterval()
			if len(headers) > 0 {
				etl.log.Info("retrying previous batch")
			} else {
				newHeaders, err := etl.headerTraversal.NextFinalizedHeaders(etl.headerBufferSize)
				if err != nil {
					etl.log.Error("error querying for headers", "err", err)
				} else if len(newHeaders) == 0 {
					etl.log.Warn("no new headers. processor unexpectedly at head...")
				}

				headers = newHeaders
				etl.metrics.RecordBatchHeaders(len(newHeaders))
			}

			// only clear the reference if we were able to process this batch
			err := etl.processBatch(headers)
			if err == nil {
				headers = nil
			}

			done(err)
		}
	}
}

func (etl *ETL) processBatch(headers []types.Header) error {
	if len(headers) == 0 {
		return nil
	}

	firstHeader, lastHeader := headers[0], headers[len(headers)-1]
	batchLog := etl.log.New("batch_start_block_number", firstHeader.Number, "batch_end_block_number", lastHeader.Number)
	batchLog.Info("extracting batch", "size", len(headers))

	etl.metrics.RecordBatchLatestHeight(lastHeader.Number)
	headerMap := make(map[common.Hash]*types.Header, len(headers))
	for i := range headers {
		header := headers[i]
		headerMap[header.Hash()] = &header
	}

	headersWithLog := make(map[common.Hash]bool, len(headers))
	logs, err := etl.EthClient.FilterLogs(ethereum.FilterQuery{FromBlock: firstHeader.Number, ToBlock: lastHeader.Number, Addresses: etl.contracts})
	if err != nil {
		batchLog.Info("unable to extract logs", "err", err)
		return err
	}
	if len(logs) > 0 {
		batchLog.Info("detected logs", "size", len(logs))
	}

	for i := range logs {
		log := logs[i]
		if _, ok := headerMap[log.BlockHash]; !ok {
			// NOTE. Definitely an error state if the none of the headers were re-orged out in between
			// the blocks and logs retrieval operations. However, we need to gracefully handle reorgs
			batchLog.Error("log found with block hash not in the batch", "block_hash", logs[i].BlockHash, "log_index", logs[i].Index)
			return errors.New("parsed log with a block hash not in the batch")
		}

		etl.metrics.RecordBatchLog(log.Address)
		headersWithLog[log.BlockHash] = true
	}

	// ensure we use unique downstream references for the etl batch
	headersRef := headers
	etl.etlBatches <- ETLBatch{Logger: batchLog, Headers: headersRef, HeaderMap: headerMap, Logs: logs, HeadersWithLog: headersWithLog}
	return nil
}
