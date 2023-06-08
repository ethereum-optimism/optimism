package processor

import (
	"time"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const defaultLoopInterval = 5 * time.Second

// processFn is the the function used to process unindexed headers. In
// the event of a failure, all database operations are not committed
type processFn func(*database.DB, []*types.Header) error

type processor struct {
	fetcher *node.Fetcher

	db         *database.DB
	processFn  processFn
	processLog log.Logger
}

// Start kicks off the processing loop
func (p processor) Start() {
	pollTicker := time.NewTicker(defaultLoopInterval)
	p.processLog.Info("starting processor...")

	// Make this loop stoppable
	for range pollTicker.C {
		p.processLog.Info("checking for new headers...")

		headers, err := p.fetcher.NextFinalizedHeaders()
		if err != nil {
			p.processLog.Error("unable to query for headers", "err", err)
			continue
		}

		if len(headers) == 0 {
			p.processLog.Info("no new headers. indexer must be at head...")
			continue
		}

		batchLog := p.processLog.New("startHeight", headers[0].Number, "endHeight", headers[len(headers)-1].Number)
		batchLog.Info("indexing batch of headers")

		// wrap operations within a single transaction
		err = p.db.Transaction(func(db *database.DB) error {
			return p.processFn(db, headers)
		})

		// TODO(DX-79) if processFn failed, the next poll should retry starting from this same batch of headers

		if err != nil {
			batchLog.Info("unable to index batch", "err", err)
			panic(err)
		} else {
			batchLog.Info("done indexing batch")
		}
	}
}
