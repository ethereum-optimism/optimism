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

type L1Source struct {
	log     log.Logger
	dataSrc DataAvailabilitySource
	next    L1SourceOutput

	Origin

	data  eth.Data
	datas DataIter
}

var _ Stage = (*L1Source)(nil)

func NewL1Source(log log.Logger, dataSrc DataAvailabilitySource, next L1SourceOutput) *L1Source {
	return &L1Source{
		log:     log,
		dataSrc: dataSrc,
		next:    next,
	}
}

func (l1s *L1Source) Step(ctx context.Context, outer Origin) error {
	if changed, err := l1s.UpdateOrigin(outer); err != nil || changed {
		return err
	}

	// specific to L1 source: if the L1 origin is closed, there is no more data to retrieve.
	if l1s.Origin.Closed {
		return io.EOF
	}

	// create a source if we have none
	if l1s.datas == nil {
		datas, err := l1s.dataSrc.OpenData(ctx, l1s.Origin.Current.ID())
		if err != nil {
			l1s.log.Error("can't fetch L1 data", "origin", l1s.Origin.Current)
			return nil
		}
		l1s.log.Warn("opened L1 data source")
		l1s.datas = datas
		return nil
	}

	// buffer data if we have none
	if l1s.data == nil {
		l1s.log.Warn("fetching next piece of data")
		data, err := l1s.datas.Next(ctx)
		if err != nil && err == ctx.Err() {
			l1s.log.Warn("context to retrieve next L1 data failed", "err", err)
			return nil
		} else if err == io.EOF {
			l1s.log.Warn("no more data")
			l1s.Origin.Closed = true
			l1s.datas = nil
			return io.EOF
		} else if err != nil {
			return err
		} else {
			l1s.log.Warn("read piece of data")
			l1s.data = data
			return nil
		}
	}

	// try to flush the data to next stage
	if err := l1s.next.IngestData(l1s.data); err != nil {
		return err
	}
	l1s.data = nil
	return nil
}

func (l1s *L1Source) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	l1s.Origin = l1s.next.Progress()
	l1s.datas = nil
	l1s.data = nil
	return io.EOF
}
