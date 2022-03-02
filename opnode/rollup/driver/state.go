package driver

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/log"
)

type inputInterface interface {
	L1Head(ctx context.Context) (eth.L1Node, error)
	L2Head(ctx context.Context) (eth.L2Node, error)
	L1ChainWindow(ctx context.Context, base eth.BlockID) ([]eth.BlockID, error)
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
	l1Heads <-chan eth.L1Node
	input   inputInterface
	output  outputInterface

	log  log.Logger
	done chan struct{}
}

// l1WindowEnd returns the last block that should be used as `base` to L1ChainWindow
// This is either the last block of the window, or the L1 base block if the window is not populated
func (s *state) l1WindowEnd() eth.BlockID {
	if len(s.l1Window) == 0 {
		return s.l1Base
	}
	return s.l1Window[len(s.l1Window)-1]
}

// TODO: Split this function into populate window & SequencingWindow
func (s *state) getNextWindow(ctx context.Context) ([]eth.BlockID, error) {

	if uint64(len(s.l1Window)) < s.Config.SeqWindowSize {
		nexts, err := s.input.L1ChainWindow(ctx, s.l1WindowEnd())
		if err != nil {
			return nil, err
		}
		s.l1Window = append(s.l1Window, nexts...)
	}
	l := uint64(len(s.l1Window))
	if l > s.Config.SeqWindowSize {
		l = s.Config.SeqWindowSize
	}
	return s.l1Window[:l], nil
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

func (s *state) Start(ctx context.Context, l1Heads <-chan eth.L1Node) error {
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
	s.l1Base = l2Head.L1Parent
	s.l1Heads = l1Heads

	go s.loop()
	return nil
}

func (s *state) Close() error {
	close(s.done)
	return nil
}

func (s *state) handleReorg(ctx context.Context, head eth.L1Node) error {
	log.Warn("L1 Head signal indicates an L1 re-org", "old_l1_head", s.l1Head, "new_l1_head_parent", head.Parent, "new_l1_head", head.Self)
	nextL2Head, err := s.input.L2Head(ctx)
	if err != nil {
		log.Error("Could not get new L2 head when trying to handle a re-org", "err", err)
		// TODO: How do you handle this error - it seems to break everything
		return err
	}
	s.l1Head = head.Parent
	s.l1Window = nil
	s.l1Base = nextL2Head.L1Parent
	s.l2Head = nextL2Head.Self
	return nil
}

// newL1Head takes the new head and updates the internal state to reflect the new chain state.
// Returns true if it is a re-org, false otherwise (linear extension, no-op, or other simple case).
// If there is a re-org, this updates the internal state to handle the re-org.
// Note that `ctx` is only used in a re-org and that handling a re-org may take a long period of time.
// The L2 engine is not modified in this function.
func (s *state) newL1Head(ctx context.Context, head eth.L1Node) (bool, error) {
	// Already have head
	if s.l1Head == head.Self {
		log.Trace("Received L1 head signal that is the same as the current head", "l1_head", head.Self)
		return false, nil
	}
	// Re-org (maybe also a skip)
	if s.l1Head != head.Parent {
		err := s.handleReorg(ctx, head)
		if err != nil {
			return false, err
		}
	}
	// Linear extension
	s.l1Head = head.Self
	if len(s.l1Window) > 0 && s.l1Window[len(s.l1Window)-1] == head.Parent {
		// // don't buffer more than 20 sequencing windows  (TBD, sanity limit)
		// if uint64(len(e.l1Next)) < e.Config.SeqWindowSize*20 {
		// 	e.l1Next = append(e.l1Next, l1HeadSig.Self)
		// }
		s.l1Window = append(s.l1Window, head.Self)
	}

	return false, nil

}

func (s *state) newSafeL2Head(head eth.BlockID) {
	s.log.Trace("New L2 head", "head", head)
	s.l2Head = head
	// TODO: Update L1 base here.

	// Remove the processed L1 block from the window
	if len(s.l1Window) > 0 {
		s.l1Window = s.l1Window[:1]
	}
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
		case l1HeadSig := <-s.l1Heads:
			s.log.Trace("L1 Head Update", "new_head", l1HeadSig.Self)
			// Set a long timeout because the timeout is for running sync.L2Head
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, err := s.newL1Head(ctx, l1HeadSig)
			if err != nil {
				panic(err)
			}
			requestStep()
			cancel()

		case <-stepRequest:
			s.log.Trace("Step request")
			window, err := s.getNextWindow(ctx)
			if err != nil {
				panic(err)
			}
			if len(window) == int(s.Config.SeqWindowSize) {
				s.log.Trace("Running step")
				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				newL2Head, err := s.output.step(ctx, s.l2Head, s.l2Finalized, window)
				cancel()
				s.log.Trace("step output", "head", newL2Head, "err", err)
				if err != nil {
					panic(err)
				}
				s.newSafeL2Head(newL2Head)
			} else {
				s.log.Trace("Not enough saved blocks to run step")
			}

		}
	}

}
