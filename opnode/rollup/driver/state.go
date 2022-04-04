package driver

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/sync"
	"github.com/ethereum/go-ethereum/log"
)

type state struct {
	// Chain State
	l1Head      eth.L1BlockRef // Latest recorded head of the L1 Chain
	l2Head      eth.L2BlockRef // L2 Unsafe Head
	l2SafeHead  eth.L2BlockRef // L2 Safe Head - this is the head of the L2 chain as derived from L1 (thus it is Sequencer window blocks behind)
	l2Finalized eth.BlockID    // L2 Block that will never be reversed
	l1WindowBuf []eth.BlockID  // l1WindowBuf buffers the next L1 block IDs to derive new L2 blocks from, with increasing block height.

	// Rollup config
	Config    rollup.Config
	sequencer bool

	// Connections (in/out)
	l1Heads <-chan eth.L1BlockRef
	l1      L1Chain
	l2      L2Chain
	output  outputInterface
	bss     BatchSubmitter

	log  log.Logger
	done chan struct{}

	closed uint32 // non-zero when closed
}

func NewState(log log.Logger, config rollup.Config, l1 L1Chain, l2 L2Chain, output outputInterface, submitter BatchSubmitter, sequencer bool) *state {
	return &state{
		Config:    config,
		done:      make(chan struct{}),
		log:       log,
		l1:        l1,
		l2:        l2,
		output:    output,
		bss:       submitter,
		sequencer: sequencer,
	}
}

func (s *state) Start(ctx context.Context, l1Heads <-chan eth.L1BlockRef) error {
	l1Head, err := s.l1.L1HeadBlockRef(ctx)
	if err != nil {
		return err
	}
	l2Head, err := s.l2.L2BlockRefByNumber(ctx, nil)
	if err != nil {
		return err
	}

	// TODO:
	// 1. Pull safehead from sync-start algorithm
	// 2. Check if heads are below genesis & if so, bump to genesis.
	s.l1Head = l1Head
	s.l2Head = l2Head
	s.l2SafeHead = l2Head
	s.l1Heads = l1Heads

	go s.loop()
	return nil
}

func (s *state) Close() error {
	close(s.done)
	return nil
}

// l1WindowBufEnd returns the last block that should be used as `base` to L1ChainWindow.
// This is either the last block of the window, or the L1 base block if the window is not populated.
func (s *state) l1WindowBufEnd() eth.BlockID {
	if len(s.l1WindowBuf) == 0 {
		return s.l2Head.L1Origin
	}
	return s.l1WindowBuf[len(s.l1WindowBuf)-1]
}

// extendL1Window extends the cached L1 window by pulling blocks from L1.
// It starts just after `s.l1WindowBufEnd()`.
func (s *state) extendL1Window(ctx context.Context) error {
	s.log.Trace("Extending the cached window from L1", "cached_size", len(s.l1WindowBuf), "window_buf_end", s.l1WindowBufEnd())
	// fetch enough ids for 2 sequencing windows (we'll shift from one into the other before we run out again)
	nexts, err := s.l1.L1Range(ctx, s.l1WindowBufEnd(), s.Config.SeqWindowSize*2)
	if err != nil {
		return err
	}
	s.l1WindowBuf = append(s.l1WindowBuf, nexts...)
	return nil
}

// sequencingWindow returns the next sequencing window and true if it exists, (nil, false) if
// there are not enough saved blocks.
func (s *state) sequencingWindow() ([]eth.BlockID, bool) {
	if len(s.l1WindowBuf) < int(s.Config.SeqWindowSize) {
		return nil, false
	}
	return s.l1WindowBuf[:int(s.Config.SeqWindowSize)], true
}

func (s *state) findNextL1Origin(ctx context.Context) (eth.L1BlockRef, error) {
	// [prev L2 + blocktime, L1 Bock)
	if s.l2Head.L1Origin.Hash == s.l1Head.Hash {
		return s.l1Head, nil
	}
	curr, err := s.l1.L1BlockRefByHash(ctx, s.l2Head.L1Origin.Hash)
	if err != nil {
		return eth.L1BlockRef{}, err
	}

	s.log.Info("Find next l1Origin", "l2Head", s.l2Head, "l1Origin", curr)
	// TODO: There is an interaction with not using the L1 Genesis as an L1 Origin and
	// the fact that the L2 Genesis time needs to be set around the L1 Genesis such
	// that this check will return true.
	if s.l2Head.Time+s.Config.BlockTime >= curr.Time {
		// TODO: Need to walk more?
		ref, err := s.l1.L1BlockRefByNumber(ctx, curr.Number+1)
		s.log.Info("Looking up new L1 Origin", "nextL1Origin", ref)
		return ref, err
	}
	return curr, nil
}

func (s *state) loop() {
	s.log.Info("State loop started")
	ctx := context.Background()
	var l2BlockCreation <-chan time.Time
	if s.sequencer {
		l2BlockCreationTicker := time.NewTicker(time.Duration(s.Config.BlockTime) * time.Second)
		defer l2BlockCreationTicker.Stop()
		l2BlockCreation = l2BlockCreationTicker.C
	}

	stepRequest := make(chan struct{}, 1)
	l2BlockCreationReq := make(chan struct{}, 1)

	createBlock := func() {
		select {
		case l2BlockCreationReq <- struct{}{}:
		// Don't deadlock if the channel is already full
		default:
		}
	}

	requestStep := func() {
		select {
		case stepRequest <- struct{}{}:
		// Don't deadlock if the channel is already full
		default:
		}
	}

	requestStep()

	for {
		select {
		case <-s.done:
			atomic.AddUint32(&s.closed, 1)
			return
		case <-l2BlockCreation:
			s.log.Trace("L2 Creation Ticker")
			createBlock()
		case <-l2BlockCreationReq:
			nextOrigin, err := s.findNextL1Origin(context.Background())
			if err != nil {
				s.log.Error("Error finding next L1 Origin")
				continue
			}
			if nextOrigin.Time <= s.Config.BlockTime+s.l2Head.Time {
				s.log.Trace("Skipping block production", "l2Head", s.l2Head)
				continue
			}
			// Don't produce blocks until past the L1 genesis
			if nextOrigin.Number <= s.Config.Genesis.L1.Number {
				s.log.Trace("Skipping block production b/c origin behind genesis")
				continue
			}
			// 2. Ask output to create new block
			newUnsafeL2Head, batch, err := s.output.newBlock(context.Background(), s.l2Finalized, s.l2Head, s.l2SafeHead.ID(), nextOrigin.ID())
			if err != nil {
				s.log.Error("Could not extend chain as sequencer", "err", err, "l2UnsafeHead", s.l2Head, "l1Origin", nextOrigin)
				continue
			}
			// 3. Update unsafe l2 head
			s.l2Head = newUnsafeL2Head
			s.log.Trace("Created new l2 block", "l2UnsafeHead", s.l2Head)
			// 4. Ask for batch submission
			go func() {
				_, err := s.bss.Submit(&s.Config, []*derive.BatchData{batch}) // TODO: submit multiple batches
				if atomic.LoadUint32(&s.closed) > 0 {
					return // closed, don't log (go-routine may be running after logger closed)
				}
				if err != nil {
					s.log.Error("Error submitting batch", "err", err)
				}
			}()
			if nextOrigin.Time > s.l2Head.Time+s.Config.BlockTime {
				s.log.Trace("Asking for a second L2 block asap", "l2Head", s.l2Head)
				createBlock()
			}

		case newL1Head := <-s.l1Heads:
			s.log.Trace("Received new L1 Head", "new_head", newL1Head, "old_head", s.l1Head)
			if s.l1Head.Hash == newL1Head.Hash {
				log.Trace("Received L1 head signal that is the same as the current head", "l1_head", newL1Head)
			} else if s.l1Head.Hash == newL1Head.ParentHash {
				s.log.Trace("Linear extension")
				s.l1Head = newL1Head
				if s.l1WindowBufEnd().Hash == newL1Head.ParentHash {
					s.l1WindowBuf = append(s.l1WindowBuf, newL1Head.ID())
				}
			} else {
				// Not strictly always a reorg, but that is the most likely case
				s.log.Warn("L1 Head signal indicates an L1 re-org", "old_l1_head", s.l1Head, "new_l1_head_parent", newL1Head.ParentHash, "new_l1_head", newL1Head)
				unsafeL2Head, safeL2Head, err := sync.FindL2Heads(ctx, s.l2Head, int(s.Config.SeqWindowSize), s.l1, s.l2, &s.Config.Genesis)
				if err != nil {
					s.log.Error("Could not get new unsafe L2 head when trying to handle a re-org", "err", err)
					continue
				}
				// TODO: Fork choice update
				s.l1Head = newL1Head
				s.l1WindowBuf = nil
				s.l2Head = unsafeL2Head // Note that verify only nodes can get an unsafe head because of a reorg. May want to remove that.
				s.l2SafeHead = safeL2Head
				s.log.Trace("State update", "l1Head", s.l1Head, "l2Head", s.l2Head, "l2SafeHead", s.l2SafeHead)
			}

			// Run step if we are able to
			if s.l1Head.Number-s.l2Head.L1Origin.Number >= s.Config.SeqWindowSize {
				s.log.Trace("Requesting next step", "l1Head", s.l1Head, "l2Head", s.l2Head, "l1Origin", s.l2Head.L1Origin)
				requestStep()
			}
		case <-stepRequest:
			if s.sequencer {
				s.log.Trace("Skipping extension based on L1 chain as sequencer")
				continue
			}
			s.log.Trace("Got step request")
			// Extend cached window if we do not have enough saved blocks
			if len(s.l1WindowBuf) < int(s.Config.SeqWindowSize) {
				err := s.extendL1Window(context.Background())
				if err != nil {
					s.log.Error("Could not extend the cached L1 window", "err", err, "l1Head", s.l1Head, "window_buf_end", s.l1WindowBufEnd())
					continue
				}
			}

			// Get next window (& ensure that it exists)
			if window, ok := s.sequencingWindow(); ok {
				s.log.Trace("Have enough cached blocks to run step.", "window", window)
				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				newL2Head, err := s.output.step(ctx, s.l2SafeHead, s.l2Finalized, s.l2Head.ID(), window)
				cancel()
				if err != nil {
					s.log.Error("Error in running the output step.", "err", err, "l2SafeHead", s.l2SafeHead, "l2Finalized", s.l2Finalized, "window", window)
					continue
				}
				s.l2Head = newL2Head
				s.l2SafeHead = newL2Head
				s.l1WindowBuf = s.l1WindowBuf[1:]
				// TODO: l2Finalized
			} else {
				s.log.Trace("Not enough cached blocks to run step", "cached_window_len", len(s.l1WindowBuf))
			}

			// Immediately run next step if we have enough blocks.
			if s.l1Head.Number-s.l2Head.L1Origin.Number >= s.Config.SeqWindowSize {
				s.log.Trace("Requesting next step", "l1Head", s.l1Head, "l2Head", s.l2Head, "l1Origin", s.l2Head.L1Origin)
				requestStep()
			}

		}
	}

}
