package derive

import (
	"context"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
)

type L1SourceOutput interface {
	StageProgress
	IngestData(data []byte)
}

type DataAvailabilitySource interface {
	OpenData(ctx context.Context, id eth.BlockID) DataIter
}

type NextBlockProvider interface {
	NextL1Block(context.Context) (eth.L1BlockRef, error)
}

type L1Retrieval struct {
	log     log.Logger
	dataSrc DataAvailabilitySource
	next    L1SourceOutput
	prev    NextBlockProvider

	progress Progress

	datas DataIter
}

var _ Stage = (*L1Retrieval)(nil)

func NewL1Retrieval(log log.Logger, dataSrc DataAvailabilitySource, next L1SourceOutput, prev NextBlockProvider) *L1Retrieval {
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

// Step does an action in the L1 Retrieval stage
// If there is data, it pushes it to the next stage.
// If there is no more data open ourselves if we are closed or close ourselves if we are open
func (l1r *L1Retrieval) Step(ctx context.Context, _ Progress) error {
	if l1r.datas != nil {
		l1r.log.Debug("fetching next piece of data")
		data, err := l1r.datas.Next(ctx)
		if err == io.EOF {
			l1r.datas = nil
			return io.EOF
		} else if err != nil {
			return err
		} else {
			l1r.next.IngestData(data)
			return nil
		}
	} else {
		if l1r.progress.Closed {
			next, err := l1r.prev.NextL1Block(ctx)
			if err == io.EOF {
				return io.EOF
			} else if err != nil {
				return err
			}
			l1r.datas = l1r.dataSrc.OpenData(ctx, next.ID())
			l1r.progress.Origin = next
			l1r.progress.Closed = false
		} else {
			l1r.progress.Closed = true
		}
		return nil
	}
}

// ResetStep re-initializes the L1 Retrieval stage to block of it's `next` progress.
// Note that we open up the `l1r.datas` here because it is requires to maintain the
// internal invariants that later propagate up the derivation pipeline.
func (l1r *L1Retrieval) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	l1r.progress = l1r.next.Progress()
	l1r.datas = l1r.dataSrc.OpenData(ctx, l1r.progress.Origin.ID())
	l1r.log.Info("Reset of L1Retrieval done", "origin", l1r.progress.Origin)
	return io.EOF
}
