package processor

import (
	"time"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	defaultLoopInterval     = 5 * time.Second
	defaultHeaderBufferSize = 500
)

// ProcessFn is the the entrypoint for processing a batch of headers.
// In the event of failure, database operations are rolled back
type ProcessFn func(*database.DB, []*types.Header) error

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
		headers, err := p.headerTraversal.NextFinalizedHeaders(defaultHeaderBufferSize)
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

		err = p.db.Transaction(func(db *database.DB) error {
			err := p.processFn(db, headers)
			if err != nil {
				return err
			}
			return p.headerTraversal.Advance(headers[len(headers)-1])
		})

		if err != nil {
			batchLog.Warn("error processing batch. no operations committed", "err", err)
		} else {
			batchLog.Info("fully committed batch")
		}
	}
}
