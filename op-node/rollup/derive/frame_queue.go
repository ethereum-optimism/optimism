package derive

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var _ NextFrameProvider = &FrameQueue{}

type NextDataProvider interface {
	NextData(context.Context) ([]byte, error)
	Origin() eth.L1BlockRef
}

type FrameQueue struct {
	log    log.Logger
	frames []Frame
	prev   NextDataProvider
}

func NewFrameQueue(log log.Logger, prev NextDataProvider) *FrameQueue {
	return &FrameQueue{
		log:  log,
		prev: prev,
	}
}

func (fq *FrameQueue) Origin() eth.L1BlockRef {
	return fq.prev.Origin()
}

func (fq *FrameQueue) NextFrame(ctx context.Context) (Frame, error) {
	// Find more frames if we need to
	if len(fq.frames) == 0 {
		if data, err := fq.prev.NextData(ctx); err != nil {
			return Frame{}, err
		} else {
			if new, err := ParseFrames(data); err == nil {
				fq.frames = append(fq.frames, new...)
			} else {
				fq.log.Warn("Failed to parse frames", "origin", fq.prev.Origin(), "err", err)
			}
		}
	}
	// If we did not add more frames but still have more data, retry this function.
	if len(fq.frames) == 0 {
		return Frame{}, NotEnoughData
	}

	ret := fq.frames[0]
	fq.frames = fq.frames[1:]
	return ret, nil
}

func (fq *FrameQueue) Reset(_ context.Context, _ eth.L1BlockRef, _ eth.SystemConfig) error {
	fq.frames = fq.frames[:0]
	return io.EOF
}
