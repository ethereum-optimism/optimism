package driver

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1Metrics interface {
	RecordL1ReorgDepth(d uint64)
	RecordL1Ref(name string, ref eth.L1BlockRef)
}

// L1State tracks L1 head, safe and finalized blocks. It is not safe to write and read concurrently.
type L1State struct {
	log     log.Logger
	metrics L1Metrics

	// Latest recorded head, safe block and finalized block of the L1 Chain, independent of derivation work
	l1Head      eth.L1BlockRef
	l1Safe      eth.L1BlockRef
	l1Finalized eth.L1BlockRef
}

func NewL1State(log log.Logger, metrics L1Metrics) *L1State {
	return &L1State{
		log:     log,
		metrics: metrics,
	}
}

func (s *L1State) HandleNewL1HeadBlock(head eth.L1BlockRef) {
	// We don't need to do anything if the head hasn't changed.
	if s.l1Head == (eth.L1BlockRef{}) {
		s.log.Info("Received first L1 head signal", "l1_head", head)
	} else if s.l1Head.Hash == head.Hash {
		s.log.Trace("Received L1 head signal that is the same as the current head", "l1_head", head)
	} else if s.l1Head.Hash == head.ParentHash {
		// We got a new L1 block whose parent hash is the same as the current L1 head. Means we're
		// dealing with a linear extension (new block is the immediate child of the old one).
		s.log.Debug("L1 head moved forward", "l1_head", head)
	} else {
		if s.l1Head.Number >= head.Number {
			s.metrics.RecordL1ReorgDepth(s.l1Head.Number - head.Number)
		}
		// New L1 block is not the same as the current head or a single step linear extension.
		// This could either be a long L1 extension, or a reorg, or we simply missed a head update.
		s.log.Warn("L1 head signal indicates a possible L1 re-org", "old_l1_head", s.l1Head, "new_l1_head_parent", head.ParentHash, "new_l1_head", head)
	}
	s.metrics.RecordL1Ref("l1_head", head)
	s.l1Head = head
}

func (s *L1State) HandleNewL1SafeBlock(safe eth.L1BlockRef) {
	s.log.Info("New L1 safe block", "l1_safe", safe)
	s.metrics.RecordL1Ref("l1_safe", safe)
	s.l1Safe = safe
}

func (s *L1State) HandleNewL1FinalizedBlock(finalized eth.L1BlockRef) {
	s.log.Info("New L1 finalized block", "l1_finalized", finalized)
	s.metrics.RecordL1Ref("l1_finalized", finalized)
	s.l1Finalized = finalized
}

// L1Head returns either the stored L1 head or an empty block reference
// if the L1 Head has not been initialized yet.
func (s *L1State) L1Head() eth.L1BlockRef {
	return s.l1Head
}

// L1Safe returns either the stored L1 safe block or an empty block reference
// if the L1 safe block has not been initialized yet.
func (s *L1State) L1Safe() eth.L1BlockRef {
	return s.l1Safe
}

// L1Finalized returns either the stored L1 finalized block or an empty block reference
// if the L1 finalized block has not been initialized yet.
func (s *L1State) L1Finalized() eth.L1BlockRef {
	return s.l1Finalized
}
