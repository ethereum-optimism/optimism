package derive

import (
	"context"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
)

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
	IngestData(data []byte) error
}

type L1Retrieval struct {
	log     log.Logger
	dataSrc DataAvailabilitySource
	next    L1SourceOutput

	progress Progress

	data  eth.Data
	datas DataIter
}

var _ Stage = (*L1Retrieval)(nil)

func NewL1Retrieval(log log.Logger, dataSrc DataAvailabilitySource, next L1SourceOutput) *L1Retrieval {
	return &L1Retrieval{
		log:     log,
		dataSrc: dataSrc,
		next:    next,
	}
}

func (l1r *L1Retrieval) Progress() Progress {
	return l1r.progress
}

func (l1r *L1Retrieval) Step(ctx context.Context, outer Progress) error {
	if changed, err := l1r.progress.Update(outer); err != nil || changed {
		return err
	}

	// specific to L1 source: if the L1 origin is closed, there is no more data to retrieve.
	if l1r.progress.Closed {
		return io.EOF
	}

	// create a source if we have none
	if l1r.datas == nil {
		datas, err := l1r.dataSrc.OpenData(ctx, l1r.progress.Origin.ID())
		if err != nil {
			l1r.log.Error("can't fetch L1 data", "origin", l1r.progress.Origin, "err", err)
			return nil
		}
		l1r.datas = datas
		return nil
	}

	// buffer data if we have none
	if l1r.data == nil {
		l1r.log.Debug("fetching next piece of data")
		data, err := l1r.datas.Next(ctx)
		if err != nil && err == ctx.Err() {
			l1r.log.Warn("context to retrieve next L1 data failed", "err", err)
			return nil
		} else if err == io.EOF {
			l1r.progress.Closed = true
			l1r.datas = nil
			return io.EOF
		} else if err != nil {
			return err
		} else {
			l1r.data = data
			return nil
		}
	}

	// try to flush the data to next stage
	if err := l1r.next.IngestData(l1r.data); err != nil {
		return err
	}
	l1r.data = nil
	return nil
}

func (l1r *L1Retrieval) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	l1r.progress = l1r.next.Progress()
	l1r.datas = nil
	l1r.data = nil
	return io.EOF
}
