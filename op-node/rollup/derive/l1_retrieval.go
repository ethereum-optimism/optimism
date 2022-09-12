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

type L1Retrieval struct {
	log     log.Logger
	dataSrc DataAvailabilitySource
	prev    *L1Traversal

	// We maintain a `done` flag separate from `progress.Closed`
	// This is because when this stage is done, it asks the traversal stage to advance
	// There becomes a weird dependency between progress.Closed & progress.Update that necessitates this.
	progress Progress

	datas DataIter
}

var _ PullStage = (*L1Retrieval)(nil)

func NewL1Retrieval(log log.Logger, dataSrc DataAvailabilitySource, prev *L1Traversal) *L1Retrieval {
	return &L1Retrieval{
		log:     log,
		dataSrc: dataSrc,
		prev:    prev,
	}
}

func (l1r *L1Retrieval) Progress() Progress {
	return l1r.progress
}

// NextData returns the next piece of data if it has it or io.EOF if it + the
// underlying stage doesn't have it.
func (l1r *L1Retrieval) NextData(ctx context.Context) (eth.Data, error) {
	// If we are closed, try to advance & open ourselves
	if l1r.progress.Closed {
		if block, err := l1r.prev.NextL1Block(ctx); err != nil {
			l1r.progress.Origin = block
			l1r.progress.Closed = false
		} else {
			return nil, err
		}
	}

	// If we have just opened ourselves, attempt to create `l1r.datas`
	if l1r.datas == nil {
		datas, err := l1r.dataSrc.OpenData(ctx, l1r.progress.Origin.ID())
		if err != nil {
			return nil, NewTemporaryError(fmt.Errorf("can't fetch L1 data: %v: %w", l1r.progress.Origin, err))
		}
		l1r.datas = datas
	}

	// Return the piece of data
	l1r.log.Debug("fetching next piece of data")
	if data, err := l1r.datas.Next(ctx); err == io.EOF {
		l1r.datas = nil
		l1r.progress.Closed = true
		return nil, io.EOF
	} else if err != nil {
		return nil, NewTemporaryError(fmt.Errorf("context to retrieve next L1 data failed: %w", err))
	} else {
		return data, nil
	}
}

func (l1r *L1Retrieval) Reset(ctx context.Context, base eth.L1BlockRef) (eth.L1BlockRef, error) {
	l1r.progress = Progress{
		Origin: base,
		Closed: false,
	}
	datas, err := l1r.dataSrc.OpenData(ctx, l1r.progress.Origin.ID())
	if err != nil {
		return base, NewTemporaryError(fmt.Errorf("can't fetch L1 data: %v: %w", l1r.progress.Origin, err))
	}
	l1r.datas = datas
	l1r.log.Info("Reset of L1Retrieval done", "origin", l1r.progress.Origin)
	return base, io.EOF
}
