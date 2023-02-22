package actions

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
)

// MockL1OriginSelector is a shim to override the origin as sequencer, so we can force it to stay on an older origin.
type MockL1OriginSelector struct {
	actual         *driver.L1OriginSelector
	originOverride eth.L1BlockRef // override which origin gets picked
}

func (m *MockL1OriginSelector) FindL1Origin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
	if m.originOverride != (eth.L1BlockRef{}) {
		return m.originOverride, nil
	}
	return m.actual.FindL1Origin(ctx, l2Head)
}

// L2Sequencer is an actor that functions like a rollup node,
// without the full P2P/API/Node stack, but just the derivation state, and simplified driver with sequencing ability.
type L2Sequencer struct {
	L2Verifier

	sequencer *driver.Sequencer

	failL2GossipUnsafeBlock error // mock error

	mockL1OriginSelector *MockL1OriginSelector
}

func NewL2Sequencer(t Testing, log log.Logger, l1 derive.L1Fetcher, eng L2API, cfg *rollup.Config, seqConfDepth uint64) *L2Sequencer {
	ver := NewL2Verifier(t, log, l1, eng, cfg)
	attrBuilder := derive.NewFetchingAttributesBuilder(cfg, l1, eng)
	seqConfDepthL1 := driver.NewConfDepth(seqConfDepth, ver.l1State.L1Head, l1)
	l1OriginSelector := &MockL1OriginSelector{
		actual: driver.NewL1OriginSelector(log, cfg, seqConfDepthL1),
	}
	return &L2Sequencer{
		L2Verifier:              *ver,
		sequencer:               driver.NewSequencer(log, cfg, ver.derivation, attrBuilder, l1OriginSelector, metrics.NoopMetrics),
		mockL1OriginSelector:    l1OriginSelector,
		failL2GossipUnsafeBlock: nil,
	}
}

// ActL2StartBlock starts building of a new L2 block on top of the head
func (s *L2Sequencer) ActL2StartBlock(t Testing) {
	s.ActL2StartBlockCheckErr(t, nil)
}

func (s *L2Sequencer) ActL2StartBlockCheckErr(t Testing, checkErr error) {
	if !s.l2PipelineIdle {
		t.InvalidAction("cannot start L2 build when derivation is not idle")
		return
	}
	if s.l2Building {
		t.InvalidAction("already started building L2 block")
		return
	}

	err := s.sequencer.StartBuildingBlock(t.Ctx())
	if checkErr == nil {
		require.NoError(t, err, "failed to start block building")
	} else {
		require.ErrorIs(t, err, checkErr, "expected typed error")
	}

	if errors.Is(err, derive.ErrReset) {
		s.derivation.Reset()
	}

	if err == nil {
		s.l2Building = true
	}
}

// ActL2EndBlock completes a new L2 block and applies it to the L2 chain as new canonical unsafe head
func (s *L2Sequencer) ActL2EndBlock(t Testing) {
	if !s.l2Building {
		t.InvalidAction("cannot end L2 block building when no block is being built")
		return
	}
	s.l2Building = false

	_, err := s.sequencer.CompleteBuildingBlock(t.Ctx())
	// TODO: there may be legitimate temporary errors here, if we mock engine API RPC-failure.
	// For advanced tests we can catch those and print a warning instead.
	require.NoError(t, err)

	// TODO: action-test publishing of payload on p2p
}

// ActL2KeepL1Origin makes the sequencer use the current L1 origin, even if the next origin is available.
func (s *L2Sequencer) ActL2KeepL1Origin(t Testing) {
	parent := s.derivation.UnsafeL2Head()
	// force old origin, for testing purposes
	oldOrigin, err := s.l1.L1BlockRefByHash(t.Ctx(), parent.L1Origin.Hash)
	require.NoError(t, err, "failed to get current origin: %s", parent.L1Origin)
	s.mockL1OriginSelector.originOverride = oldOrigin
}

// ActBuildToL1Head builds empty blocks until (incl.) the L1 head becomes the L2 origin
func (s *L2Sequencer) ActBuildToL1Head(t Testing) {
	for s.derivation.UnsafeL2Head().L1Origin.Number < s.l1State.L1Head().Number {
		s.ActL2PipelineFull(t)
		s.ActL2StartBlock(t)
		s.ActL2EndBlock(t)
	}
}

// ActBuildToL1HeadUnsafe builds empty blocks until (incl.) the L1 head becomes the L1 origin of the L2 head
func (s *L2Sequencer) ActBuildToL1HeadUnsafe(t Testing) {
	for s.derivation.UnsafeL2Head().L1Origin.Number < s.l1State.L1Head().Number {
		// Note: the derivation pipeline does not run, we are just sequencing a block on top of the existing L2 chain.
		s.ActL2StartBlock(t)
		s.ActL2EndBlock(t)
	}
}

// ActBuildToL1HeadExcl builds empty blocks until (excl.) the L1 head becomes the L1 origin of the L2 head
func (s *L2Sequencer) ActBuildToL1HeadExcl(t Testing) {
	for {
		s.ActL2PipelineFull(t)
		nextOrigin, err := s.mockL1OriginSelector.FindL1Origin(t.Ctx(), s.derivation.UnsafeL2Head())
		require.NoError(t, err)
		if nextOrigin.Number >= s.l1State.L1Head().Number {
			break
		}
		s.ActL2StartBlock(t)
		s.ActL2EndBlock(t)
	}
}

// ActBuildToL1HeadExclUnsafe builds empty blocks until (excl.) the L1 head becomes the L1 origin of the L2 head, without safe-head progression.
func (s *L2Sequencer) ActBuildToL1HeadExclUnsafe(t Testing) {
	for {
		// Note: the derivation pipeline does not run, we are just sequencing a block on top of the existing L2 chain.
		nextOrigin, err := s.mockL1OriginSelector.FindL1Origin(t.Ctx(), s.derivation.UnsafeL2Head())
		require.NoError(t, err)
		if nextOrigin.Number >= s.l1State.L1Head().Number {
			break
		}
		s.ActL2StartBlock(t)
		s.ActL2EndBlock(t)
	}
}
