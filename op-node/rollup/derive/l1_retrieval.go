package derive

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
)

// This is a generic wrapper around fetching all transactions in a block & then
// it feeds one L1 transaction at a time to the next stage

// DataIter is a minimal iteration interface to fetch rollup input data from an arbitrary data-availability source
type DataIter interface {
	// Next can be repeatedly called for more data, until it returns an io.EOF error.
	// It never returns io.EOF and data at the same time.
	Next(ctx context.Context) (eth.Data, error)
}

// DataAvailabilitySource provides rollup input data
type DataAvailabilitySource interface {
	// OpenData does any initial data-fetching work and returns an iterator to fetch data with.
	OpenData(ctx context.Context, id eth.BlockID) (DataIter, error)
}

type L1SourceOutput interface {
	StageProgress
	IngestData(data []byte)
}

type L1Retrieval struct {
	log     log.Logger
	dataSrc DataAvailabilitySource
	next    L1SourceOutput
	prev    *L1Traversal

	// We maintain a `done` flag separate from `progress.Closed`
	// This is because when this stage is done, it asks the traversal stage to advance
	// There becomes a weird dependency between progress.Closed & progress.Update that necessitates this.
	done bool

	progress Progress

	data  eth.Data
	datas DataIter
}

var _ Stage = (*L1Retrieval)(nil)

func NewL1Retrieval(log log.Logger, dataSrc DataAvailabilitySource, next L1SourceOutput, prev *L1Traversal) *L1Retrieval {
	return &L1Retrieval{
		log:     log,
		dataSrc: dataSrc,
		next:    next,
		prev:    prev,
	}
}

func (l1r *L1Retrieval) Progress() Progress {
	return l1r.progress
}

func (l1r *L1Retrieval) Step(ctx context.Context, _ Progress) error {
	// Because the previous stage is a pull stage, we manually get it's progress
	outer := l1r.prev.Progress()
	if changed, err := l1r.progress.Update(outer); err != nil || changed {
		if changed && !outer.Closed {
			l1r.done = false
		}
		return err
	}

	// specific to L1 source: if the L1 origin is closed, there is no more data to retrieve.
	if l1r.done {
		// Return the result from the underlying stage
		return l1r.prev.NextL1Block(ctx)
	}

	// create a source if we have none
	if l1r.datas == nil {
		datas, err := l1r.dataSrc.OpenData(ctx, l1r.progress.Origin.ID())
		if err != nil {
			return NewTemporaryError(fmt.Errorf("can't fetch L1 data: %v: %w", l1r.progress.Origin, err))
		}
		l1r.datas = datas
		return nil
	}

	// buffer data if we have none
	if l1r.data == nil {
		l1r.log.Debug("fetching next piece of data")
		data, err := l1r.datas.Next(ctx)
		if err == io.EOF {
			l1r.done = true
			l1r.datas = nil
			return io.EOF
		} else if err != nil {
			return NewTemporaryError(fmt.Errorf("context to retrieve next L1 data failed: %w", err))
		} else {
			l1r.data = data
			return nil
		}
	}

	// flush the data to next stage
	l1r.next.IngestData(l1r.data)
	// and nil the data, the next step will retrieve the next data
	l1r.data = nil
	return nil
}

func (l1r *L1Retrieval) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	l1r.progress = l1r.next.Progress()
	l1r.datas = nil
	l1r.done = false
	l1r.data = nil
	l1r.log.Info("Reset of L1Retrieval done", "origin", l1r.progress.Origin)
	return io.EOF
}
