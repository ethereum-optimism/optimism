package derive

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var _ NextFrameProvider = &FrameQueue{}

//go:generate mockery --name NextDataProvider --case snake
type NextDataProvider interface {
	NextData(context.Context) ([]byte, error)
	Origin() eth.L1BlockRef
}

type FrameQueue struct {
	log    log.Logger
	frames []Frame
	prev   NextDataProvider
	cfg    *rollup.Config
}

func NewFrameQueue(log log.Logger, cfg *rollup.Config, prev NextDataProvider) *FrameQueue {
	return &FrameQueue{
		log:  log,
		prev: prev,
		cfg:  cfg,
	}
}

func (fq *FrameQueue) Origin() eth.L1BlockRef {
	return fq.prev.Origin()
}

func (fq *FrameQueue) NextFrame(ctx context.Context) (Frame, error) {
	if err := fq.loadNextFrames(ctx); err != nil {
		return Frame{}, err
	}

	// Note: this implementation first parses all frames from the next L1 transaction and only then
	// prunes all frames that were parsed. An even more memory-efficient implementation could prune
	// the frame queue each time after pulling out only a single frame.

	if fq.cfg.IsHolocene(fq.Origin().Time) {
		// TODO(12157): reset frame queue once at Holocene L1 origin block
		fq.prune()
	}

	// If we did not add more frames but still have more data, retry this function.
	if len(fq.frames) == 0 {
		return Frame{}, NotEnoughData
	}

	ret := fq.frames[0]
	fq.frames = fq.frames[1:]
	return ret, nil
}

func (fq *FrameQueue) loadNextFrames(ctx context.Context) error {
	// Find more frames if we need to
	if len(fq.frames) == 0 {
		data, err := fq.prev.NextData(ctx)
		if err != nil {
			return err
		}

		if frame, err := ParseFrames(data); err == nil {
			fq.frames = append(fq.frames, frame...)
		} else {
			fq.log.Warn("Failed to parse frames", "origin", fq.prev.Origin(), "err", err)
		}
	}
	return nil
}

func (fq *FrameQueue) prune() {
	fq.frames = pruneFrameQueue(fq.frames)
}

// pruneFrameQueue prunes the frame queue to only hold contiguous and ordered
// frames, conforming to Holocene frame queue rules.
func pruneFrameQueue(frames []Frame) []Frame {
	for i := 0; i < len(frames)-1; {
		current, next := frames[i], frames[i+1]
		discard := func(shift int) {
			frames = append(frames[0:i+shift], frames[i+1+shift:]...)
		}
		// frames for the same channel ID must arrive in order
		if current.ID == next.ID {
			if current.IsLast {
				discard(1) // discard next
				continue
			}
			if next.FrameNumber != current.FrameNumber+1 {
				discard(1) // discard next
				continue
			}
		} else {
			// first frames discard previously unclosed channels
			if next.FrameNumber == 0 && !current.IsLast {
				discard(0) // discard current
				// make sure we backwards invalidate more frames of unclosed channel
				if i > 0 {
					i--
				}
				continue
			}
			// non-first frames of new channels are dropped
			if next.FrameNumber != 0 {
				discard(1) // discard next
				continue
			}
		}
		// We only update the cursor if we didn't remove any frame, so if any frame got removed, the
		// checks are applied to the new pair in the queue at the same position.
		i++
	}
	return frames
}

func (fq *FrameQueue) Reset(_ context.Context, _ eth.L1BlockRef, _ eth.SystemConfig) error {
	fq.frames = fq.frames[:0]
	return io.EOF
}
