package derive

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DataAvailabilitySource interface {
	OpenData(ctx context.Context, ref eth.L1BlockRef, batcherAddr common.Address) (DataIter, error)
}

type NextBlockProvider interface {
	NextL1Block(context.Context) (eth.L1BlockRef, error)
	Origin() eth.L1BlockRef
	SystemConfig() eth.SystemConfig
}

type L1Retrieval struct {
	log     log.Logger
	dataSrc DataAvailabilitySource
	prev    NextBlockProvider

	datas DataIter
}

var _ ResettableStage = (*L1Retrieval)(nil)

func NewL1Retrieval(log log.Logger, dataSrc DataAvailabilitySource, prev NextBlockProvider) *L1Retrieval {
	return &L1Retrieval{
		log:     log,
		dataSrc: dataSrc,
		prev:    prev,
	}
}

func (l1r *L1Retrieval) Origin() eth.L1BlockRef {
	return l1r.prev.Origin()
}

// NextData does an action in the L1 Retrieval stage
// If there is data, it pushes it to the next stage.
// If there is no more data open ourselves if we are closed or close ourselves if we are open
func (l1r *L1Retrieval) NextData(ctx context.Context) ([]byte, error) {
	if l1r.datas == nil {
		next, err := l1r.prev.NextL1Block(ctx)
		if err == io.EOF {
			return nil, io.EOF
		} else if err != nil {
			return nil, err
		}
		if l1r.datas, err = l1r.dataSrc.OpenData(ctx, next, l1r.prev.SystemConfig().BatcherAddr); err != nil {
			return nil, fmt.Errorf("failed to open data source: %w", err)
		}
	}

	l1r.log.Debug("fetching next piece of data")
	data, err := l1r.datas.Next(ctx)
	if err == io.EOF {
		l1r.datas = nil
		return nil, io.EOF
	} else if err != nil {
		// CalldataSource appropriately wraps the error so avoid double wrapping errors here.
		return nil, err
	} else {
		return data, nil
	}
}

// Reset re-initializes the L1 Retrieval stage to block of it's `next` progress.
// Note that we open up the `l1r.datas` here because it is required to maintain the
// internal invariants that later propagate up the derivation pipeline.
func (l1r *L1Retrieval) Reset(ctx context.Context, base eth.L1BlockRef, sysCfg eth.SystemConfig) error {
	var err error
	if l1r.datas, err = l1r.dataSrc.OpenData(ctx, base, sysCfg.BatcherAddr); err != nil {
		return fmt.Errorf("failed to open data source: %w", err)
	}
	l1r.log.Info("Reset of L1Retrieval done", "origin", base)
	return io.EOF
}
