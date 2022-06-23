package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
)

type L1BlockRefByNumberFetcher interface {
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
}

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
	log      log.Logger
	l1Blocks L1BlockRefByNumberFetcher
	dataSrc  DataAvailabilitySource
	next     L1SourceOutput

	origin eth.L1BlockRef
	data   eth.Data
	datas  DataIter
}

var _ Stage = (*L1Source)(nil)

func NewL1Source(log log.Logger, l1Blocks L1BlockRefByNumberFetcher,
	dataSrc DataAvailabilitySource, next L1SourceOutput) *L1Source {
	return &L1Source{
		log:      log,
		l1Blocks: l1Blocks,
		dataSrc:  dataSrc,
		next:     next,
	}
}

func (l1s *L1Source) CurrentOrigin() eth.L1BlockRef {
	return l1s.origin
}

func (l1s *L1Source) Step(ctx context.Context) error {
	// open origin if we have not yet
	if l1s.next.CurrentOrigin() != l1s.origin {
		return l1s.next.OpenOrigin(l1s.origin)
	}
	// write data if we have any
	if l1s.data != nil {
		if err := l1s.next.IngestData(l1s.data); err != nil {
			return err
		}
		l1s.data = nil
		return nil
	}

	// buffer data if we have a source
	if l1s.datas != nil {
		data, err := l1s.datas.Next(ctx)
		if err != nil && err == ctx.Err() {
			l1s.log.Warn("context to retrieve next L1 data failed", "err", err)
			return nil
		} else if err == io.EOF {
			l1s.datas = nil
			return nil
		} else if err != nil {
			return err
		} else {
			l1s.data = data
			return nil
		}
	}

	// close previous data if we need to
	if l1s.next.IsOriginOpen() {
		l1s.next.CloseOrigin()
		return nil
	}

	// TODO: we need to add confirmation depth in the source here, and return ethereum.NotFound when the data is not ready to be read.

	nextL1Origin, err := l1s.l1Blocks.L1BlockRefByNumber(ctx, l1s.origin.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		l1s.log.Debug("can't find next L1 block info", "number", l1s.origin.Number+1)
		return io.EOF
	} else if err != nil {
		l1s.log.Warn("failed to find L1 block info by number", "number", l1s.origin.Number+1, "err", err)
		return nil
	}
	if nextL1Origin.ParentHash != l1s.origin.Hash {
		// reorg, time to reset the pipeline
		return fmt.Errorf("reorg on L1, found %s with parent %s but expected parent to be %s",
			nextL1Origin.ID(), nextL1Origin.ParentID(), l1s.origin.ID())
	}
	datas, err := l1s.dataSrc.OpenData(ctx, nextL1Origin.ID())
	if err != nil {
		l1s.log.Debug("can't fetch L1 data", "origin", nextL1Origin)
		return nil
	}
	l1s.origin = nextL1Origin
	l1s.datas = datas
	return nil
}

func (l1s *L1Source) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	l1s.origin = l1s.next.CurrentOrigin()
	l1s.log.Info("completed reset of derivation pipeline", "origin", l1s.origin)
	return io.EOF
}
