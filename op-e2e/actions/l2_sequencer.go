package actions

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
)

// L2Sequencer is an actor that functions like a rollup node,
// without the full P2P/API/Node stack, but just the derivation state, and simplified driver with sequencing ability.
type L2Sequencer struct {
	L2Verifier

	sequencer        *driver.Sequencer
	l1OriginSelector *driver.L1OriginSelector

	seqOldOrigin bool // stay on current L1 origin when sequencing a block, unless forced to adopt the next origin

	failL2GossipUnsafeBlock error // mock error
}

func NewL2Sequencer(t Testing, log log.Logger, l1 derive.L1Fetcher, eng L2API, cfg *rollup.Config, seqConfDepth uint64) *L2Sequencer {
	ver := NewL2Verifier(t, log, l1, eng, cfg)
	return &L2Sequencer{
		L2Verifier:              *ver,
		sequencer:               driver.NewSequencer(log, cfg, l1, eng),
		l1OriginSelector:        driver.NewL1OriginSelector(log, cfg, l1, seqConfDepth),
		seqOldOrigin:            false,
		failL2GossipUnsafeBlock: nil,
	}
}

// ActL2StartBlock starts building of a new L2 block on top of the head
func (s *L2Sequencer) ActL2StartBlock(t Testing) {
	if !s.l2PipelineIdle {
		t.InvalidAction("cannot start L2 build when derivation is not idle")
		return
	}
	if s.l2Building {
		t.InvalidAction("already started building L2 block")
		return
	}

	parent := s.derivation.UnsafeL2Head()
	var origin eth.L1BlockRef
	if s.seqOldOrigin {
		// force old origin, for testing purposes
		oldOrigin, err := s.l1.L1BlockRefByHash(t.Ctx(), parent.L1Origin.Hash)
		require.NoError(t, err, "failed to get current origin: %s", parent.L1Origin)
		origin = oldOrigin
		s.seqOldOrigin = false // don't repeat this
	} else {
		// select origin the real way
		l1Origin, err := s.l1OriginSelector.FindL1Origin(t.Ctx(), s.l1State.L1Head(), parent)
		require.NoError(t, err)
		origin = l1Origin
	}

	err := s.sequencer.StartBuildingBlock(t.Ctx(), parent, s.derivation.SafeL2Head().ID(), s.derivation.Finalized().ID(), origin)
	require.NoError(t, err, "failed to start block building")

	s.l2Building = true
}

// ActL2EndBlock completes a new L2 block and applies it to the L2 chain as new canonical unsafe head
func (s *L2Sequencer) ActL2EndBlock(t Testing) {
	if !s.l2Building {
		t.InvalidAction("cannot end L2 block building when no block is being built")
		return
	}
	s.l2Building = false

	payload, err := s.sequencer.CompleteBuildingBlock(t.Ctx())
	// TODO: there may be legitimate temporary errors here, if we mock engine API RPC-failure.
	// For advanced tests we can catch those and print a warning instead.
	require.NoError(t, err)

	ref, err := derive.PayloadToBlockRef(payload, &s.rollupCfg.Genesis)
	require.NoError(t, err, "payload must convert to block ref")
	s.derivation.SetUnsafeHead(ref)
	// TODO: action-test publishing of payload on p2p
}

// ActL2KeepL1Origin makes the sequencer use the current L1 origin, even if the next origin is available.
func (s *L2Sequencer) ActL2KeepL1Origin(t Testing) {
	if s.seqOldOrigin { // don't do this twice
		t.InvalidAction("already decided to keep old L1 origin")
		return
	}
	s.seqOldOrigin = true
}

// ActBuildToL1Head builds empty blocks until (incl.) the L1 head becomes the L2 origin
func (s *L2Sequencer) ActBuildToL1Head(t Testing) {
	for s.derivation.UnsafeL2Head().L1Origin.Number < s.l1State.L1Head().Number {
		s.ActL2PipelineFull(t)
		s.ActL2StartBlock(t)
		s.ActL2EndBlock(t)
	}
}

// ActBuildToL1HeadExcl builds empty blocks until (excl.) the L1 head becomes the L2 origin
func (s *L2Sequencer) ActBuildToL1HeadExcl(t Testing) {
	for {
		s.ActL2PipelineFull(t)
		nextOrigin, err := s.l1OriginSelector.FindL1Origin(t.Ctx(), s.l1State.L1Head(), s.derivation.UnsafeL2Head())
		require.NoError(t, err)
		if nextOrigin.Number >= s.l1State.L1Head().Number {
			break
		}
		s.ActL2StartBlock(t)
		s.ActL2EndBlock(t)
	}
}
