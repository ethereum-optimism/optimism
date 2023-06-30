package processor

import (
	"time"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const defaultLoopInterval = 5 * time.Second

// ProcessFn is the the entrypoint for processing a batch of headers. To support
// partial batch processing, the function must return the last processed header
// in the batch. In the event of failure, database operations are rolled back
type ProcessFn func(*database.DB, []*types.Header) (*types.Header, error)

type processor struct {
	headerTraversal *node.BufferedHeaderTraversal

	db         *database.DB
	processFn  ProcessFn
	processLog log.Logger
}

// Start kicks off the processing loop
func (p processor) Start() {
	pollTicker := time.NewTicker(defaultLoopInterval)
	defer pollTicker.Stop()

	p.processLog.Info("starting processor...")
	for range pollTicker.C {
		headers, err := p.headerTraversal.NextFinalizedHeaders(500)
		if err != nil {
			p.processLog.Error("error querying for headers", "err", err)
			continue
		} else if len(headers) == 0 {
			// Logged as an error since this loop should be operating at a longer interval than the provider
			p.processLog.Error("no new headers. processor unexpectadly at head...")
			continue
		}

		batchLog := p.processLog.New("batch_start_block_number", headers[0].Number, "batch_end_block_number", headers[len(headers)-1].Number)
		batchLog.Info("processing batch")

		var lastProcessedHeader *types.Header
		err = p.db.Transaction(func(db *database.DB) error {
			lastProcessedHeader, err = p.processFn(db, headers)
			if err != nil {
				return err
			}

			err = p.headerTraversal.Advance(lastProcessedHeader)
			if err != nil {
				batchLog.Error("unable to advance processor", "last_processed_block_number", lastProcessedHeader.Number)
				return err
			}

			return nil
		})

		if err != nil {
			batchLog.Warn("error processing batch. no operations committed", "err", err)
		} else {
			if lastProcessedHeader.Number.Cmp(headers[len(headers)-1].Number) == 0 {
				batchLog.Info("fully committed batch")
			} else {
				batchLog.Info("partially committed batch", "last_processed_block_number", lastProcessedHeader.Number)
			}
		}
	}
}
