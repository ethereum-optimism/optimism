package derive

import (
	"context"
	"fmt"
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
	OriginStage
	IngestData(data []byte) error
}

type L1Source struct {
	log     log.Logger
	dataSrc DataAvailabilitySource
	next    L1SourceOutput

	currentOrigin eth.L1BlockRef
	originOpen    bool
	data          eth.Data
	datas         DataIter
}

func (l1s *L1Source) OpenOrigin(ref eth.L1BlockRef) error {
	if l1s.originOpen {
		panic("double open")
	}
	if ref.ParentHash != l1s.currentOrigin.Hash {
		return fmt.Errorf("reorg detected, cannot start consuming this L1 block without using a new channel bank: new.parent: %s, expected: %s", ref.ParentID(), l1s.currentOrigin.ParentID())
	}
	l1s.currentOrigin = ref
	l1s.originOpen = true
	return nil
}

func (l1s *L1Source) CloseOrigin() {
	l1s.originOpen = false
}

func (l1s *L1Source) IsOriginOpen() bool {
	return l1s.originOpen
}

var _ OriginStage = (*L1Source)(nil)

func NewL1Source(log log.Logger, dataSrc DataAvailabilitySource, next L1SourceOutput) *L1Source {
	return &L1Source{
		log:     log,
		dataSrc: dataSrc,
		next:    next,
	}
}

func (l1s *L1Source) CurrentOrigin() eth.L1BlockRef {
	return l1s.currentOrigin
}

func (l1s *L1Source) Step(ctx context.Context) error {
	// open origin of next stage if we have not yet
	if l1s.next.CurrentOrigin() != l1s.currentOrigin {
		return l1s.next.OpenOrigin(l1s.currentOrigin)
	}

	// create a source if we have none
	if l1s.datas == nil {
		datas, err := l1s.dataSrc.OpenData(ctx, l1s.currentOrigin.ID())
		if err != nil {
			l1s.log.Error("can't fetch L1 data", "origin", l1s.currentOrigin)
			return nil
		}
		l1s.log.Warn("opened L1 data source")
		l1s.datas = datas
		return nil
	}

	// buffer data if we have none
	if l1s.data == nil {
		data, err := l1s.datas.Next(ctx)
		if err != nil && err == ctx.Err() {
			l1s.log.Warn("context to retrieve next L1 data failed", "err", err)
			return nil
		} else if err == io.EOF {
			// close previous data if we still need to
			if l1s.next.IsOriginOpen() {
				l1s.next.CloseOrigin()
			}
			return io.EOF
		} else if err != nil {
			return err
		} else {
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
	l1s.currentOrigin = l1s.next.CurrentOrigin()
	l1s.datas = nil
	l1s.data = nil
	return io.EOF
}
