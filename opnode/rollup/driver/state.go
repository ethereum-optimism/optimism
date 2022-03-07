package driver

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/log"
)

type inputInterface interface {
	L1Head(ctx context.Context) (eth.L1BlockRef, error)
	L2Head(ctx context.Context) (eth.L2BlockRef, error)
	L1ChainWindow(ctx context.Context, base eth.BlockID) ([]eth.BlockID, error)
	// SafeL2Head is the L2 Head found via the sync algorithm
	SafeL2Head(ctx context.Context) (eth.L2BlockRef, error)
}

type outputInterface interface {
	step(ctx context.Context, l2Head eth.BlockID, l2Finalized eth.BlockID, l1Window []eth.BlockID) (eth.BlockID, error)
}

type state struct {
	// Chain State
	l1Head      eth.BlockID   // Latest recorded head of the L1 Chain
	l1Base      eth.BlockID   // L1 Parent of L2 Head block
	l2Head      eth.BlockID   // L2 Safe Head - this is the head of the L2 chain as derived from L1 (thus it is Sequencer window blocks behind)
	l2Finalized eth.BlockID   // L2 Block that will never be reversed
	l1Window    []eth.BlockID // l1Window buffers the next L1 block IDs to derive new L2 blocks from, with increasing block height.

	// Rollup config
	Config rollup.Config

	// Connections (in/out)
	l1Heads <-chan eth.L1BlockRef
	input   inputInterface
	output  outputInterface

	log  log.Logger
	done chan struct{}
}

func NewState(log log.Logger, config rollup.Config, input inputInterface, output outputInterface) *state {
	return &state{
		Config: config,
		done:   make(chan struct{}),
		log:    log,
		input:  input,
		output: output,
	}
}

func (s *state) Start(ctx context.Context, l1Heads <-chan eth.L1BlockRef) error {
	l1Head, err := s.input.L1Head(ctx)
	if err != nil {
		return err
	}
	l2Head, err := s.input.L2Head(ctx)
	if err != nil {
		return err
	}

	s.l1Head = l1Head.Self
	s.l2Head = l2Head.Self
	s.l1Base = l2Head.L1Origin
	s.l1Heads = l1Heads

	go s.loop()
	return nil
}

func (s *state) Close() error {
	close(s.done)
	return nil
}

// l1WindowEnd returns the last block that should be used as `base` to L1ChainWindow.
// This is either the last block of the window, or the L1 base block if the window is not populated.
func (s *state) l1WindowEnd() eth.BlockID {
	if len(s.l1Window) == 0 {
		return s.l1Base
	}
	return s.l1Window[len(s.l1Window)-1]
}

// extendL1Window extends the cached L1 window by pulling blocks from L1.
// It starts just after `s.l1WindowEnd()`.
func (s *state) extendL1Window(ctx context.Context) error {
	s.log.Trace("Extending the cached window from L1", "cached_size", len(s.l1Window), "window_end", s.l1WindowEnd())
	nexts, err := s.input.L1ChainWindow(ctx, s.l1WindowEnd())
	if err != nil {
		return err
	}
	s.l1Window = append(s.l1Window, nexts...)
	return nil
}

// sequencingWindow returns the next sequencing window and true if it exists, (nil, false) if
// there are not enough saved blocks.
func (s *state) sequencingWindow() ([]eth.BlockID, bool) {
	if len(s.l1Window) < int(s.Config.SeqWindowSize) {
		return nil, false
	}
	return s.l1Window[:int(s.Config.SeqWindowSize)], true
}

func (s *state) loop() {
	s.log.Info("State loop started")
	ctx := context.Background()
	// l1Poll := time.NewTicker(1 * time.Second)
	// l2Poll := time.NewTicker(1 * time.Second)
	stepRequest := make(chan struct{}, 1)
	// defer l1Poll.Stop()
	// defer l2Poll.Stop()

	requestStep := func() {
		select {
		case stepRequest <- struct{}{}:
		default:
		}
	}

	requestStep()

	for {
		select {
		// TODO: Poll cases (and move to bottom)
		// case <-l1Poll.C:
		// case <-l2Poll.C:
		case <-s.done:
			return
		case newL1Head := <-s.l1Heads:
			s.log.Trace("Received new L1 Head", "new_head", newL1Head.Self, "old_head", s.l1Head)
			// Check if we have a stutter step. May be due to a L1 Poll operation.
			if s.l1Head == newL1Head.Self {
				log.Trace("Received L1 head signal that is the same as the current head", "l1_head", newL1Head.Self)
				continue
			}

			// Typically get linear extension, but if not, handle a re-org
			if s.l1Head == newL1Head.Parent {
				s.log.Trace("Linear extension")
				s.l1Head = newL1Head.Self
				if s.l1WindowEnd() == newL1Head.Parent {
					s.l1Window = append(s.l1Window, newL1Head.Self)
				}
			} else {
				s.log.Warn("L1 Head signal indicates an L1 re-org", "old_l1_head", s.l1Head, "new_l1_head_parent", newL1Head.Parent, "new_l1_head", newL1Head.Self)
				nextL2Head, err := s.input.SafeL2Head(ctx)
				if err != nil {
					s.log.Error("Could not get new safe L2 head when trying to handle a re-org", "err", err)
					continue
				}
				s.l1Head = newL1Head.Self
				s.l1Window = nil
				s.l1Base = nextL2Head.L1Origin
				s.l2Head = nextL2Head.Self
			}
			// Run step if we are able to
			if s.l1Head.Number-s.l1Base.Number >= s.Config.SeqWindowSize {
				requestStep()
			}
		case <-stepRequest:
			s.log.Trace("Got step request")
			// Extend cached window if we do not have enough saved blocks
			if len(s.l1Window) < int(s.Config.SeqWindowSize) {
				err := s.extendL1Window(context.Background())
				if err != nil {
					s.log.Error("Could not extend the cached L1 window", "err", err, "l1Head", s.l1Head, "l1Base", s.l1Base, "window_end", s.l1WindowEnd())
					continue
				}
			}

			// Get next window (& ensure that it exists)
			if window, ok := s.sequencingWindow(); ok {
				s.log.Trace("Have enough cached blocks to run step.")
				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				newL2Head, err := s.output.step(ctx, s.l2Head, s.l2Finalized, window)
				cancel()
				if err != nil {
					s.log.Error("Error in running the output step.", "err", err, "l2Head", s.l2Head, "l2Finalized", s.l2Finalized, "window", window)
					continue
				}
				s.l2Head = newL2Head
				s.l1Base = s.l1Window[0]
				s.l1Window = s.l1Window[1:]
				// TODO: l2Finalized
			} else {
				s.log.Trace("Not enough cached blocks to run step", "cached_window_len", len(s.l1Window))
			}

			// Immediately run next step if we have enough blocks.
			if s.l1Head.Number-s.l1Base.Number >= s.Config.SeqWindowSize {
				requestStep()
			}

		}
	}

}
