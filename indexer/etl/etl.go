package etl

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	defaultLoopInterval     = 5 * time.Second
	defaultHeaderBufferSize = 500
)

type ETL struct {
	log log.Logger

	headerTraversal *node.HeaderTraversal
	ethClient       *ethclient.Client
	contracts       []common.Address

	etlBatches chan ETLBatch
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
	pollTicker := time.NewTicker(defaultLoopInterval)
	defer pollTicker.Stop()

	etl.log.Info("starting etl...")
	var headers []types.Header
	for {
		select {
		case <-done:
			etl.log.Info("stopping etl")
			return nil

		case <-pollTicker.C:
			if len(headers) == 0 {
				newHeaders, err := etl.headerTraversal.NextFinalizedHeaders(defaultHeaderBufferSize)
				if err != nil {
					etl.log.Error("error querying for headers", "err", err)
					continue
				}
				if len(newHeaders) == 0 {
					// Logged as an error since this loop should be operating at a longer interval than the provider
					etl.log.Error("no new headers. processor unexpectedly at head...")
					continue
				}

				headers = newHeaders
			} else {
				etl.log.Info("retrying previous batch")
			}

			firstHeader := headers[0]
			lastHeader := headers[len(headers)-1]
			batchLog := etl.log.New("batch_start_block_number", firstHeader.Number, "batch_end_block_number", lastHeader.Number)
			batchLog.Info("extracting batch", "size", len(headers))

			headerMap := make(map[common.Hash]*types.Header, len(headers))
			for i := range headers {
				headerMap[headers[i].Hash()] = &headers[i]
			}

			headersWithLog := make(map[common.Hash]bool, len(headers))
			logFilter := ethereum.FilterQuery{FromBlock: firstHeader.Number, ToBlock: lastHeader.Number, Addresses: etl.contracts}
			logs, err := etl.ethClient.FilterLogs(context.Background(), logFilter)
			if err != nil {
				batchLog.Info("unable to extract logs within batch", "err", err)
				continue // spin and try again
			}

			for i := range logs {
				if _, ok := headerMap[logs[i].BlockHash]; !ok {
					// NOTE. Definitely an error state if the none of the headers were re-orged out in between
					// the blocks and logs retreival operations. However, we need to gracefully handle reorgs
					batchLog.Error("log found with block hash not in the batch", "block_hash", logs[i].BlockHash, "log_index", logs[i].Index)
					return errors.New("parsed log with a block hash not in the fetched batch")
				}
				headersWithLog[logs[i].BlockHash] = true
			}

			if len(logs) > 0 {
				batchLog.Info("detected logs", "size", len(logs))
			}

			// create a new reference such that subsequent changes to `headers` does not affect the reference
			headersRef := headers
			batch := ETLBatch{Logger: batchLog, Headers: headersRef, HeaderMap: headerMap, Logs: logs, HeadersWithLog: headersWithLog}

			headers = nil
			etl.etlBatches <- batch
		}
	}
}
