package actions

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type L2Sequencer struct {
	L2Verifier

	seqOldOrigin bool // stay on current L1 origin when sequencing a block, unless forced to adopt the next origin

	l1Chain derive.L1Fetcher

	payloadID eth.PayloadID

	failL2GossipUnsafeBlock error // mock error
}

var _ ActorL2Sequencer = (*L2Sequencer)(nil)

func NewL2Sequencer(log log.Logger, l1 derive.L1Fetcher, eng derive.Engine, cfg *rollup.Config) *L2Sequencer {
	ver := NewL2Verifier(log, l1, eng, cfg)
	return &L2Sequencer{
		L2Verifier:              *ver,
		seqOldOrigin:            false,
		l1Chain:                 l1,
		failL2GossipUnsafeBlock: nil,
	}
}

// start new L2 block on top of head
func (s *L2Sequencer) actL2StartBlock(t Testing) {
	if !s.l2PipelineIdle {
		t.InvalidAction("cannot start L2 build when derivation is not idle")
		return
	}
	if s.l2Building {
		t.InvalidAction("already started building L2 block")
		return
	}

	parent := s.derivation.UnsafeL2Head()
	l2Timestamp := parent.Time + s.rollupCfg.BlockTime

	currentOrigin, err := s.l1Chain.L1BlockRefByHash(t.Ctx(), parent.L1Origin.Hash)
	require.NoError(t, err, "failed to get current origin: %s", parent.L1Origin)

	// findL1Origin test equivalent
	nextOrigin, err := s.l1Chain.L1BlockRefByNumber(t.Ctx(), currentOrigin.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		err = nil
	} else if err != nil {
		require.NoError(t, err, "failed to get next l1 block")
	}
	origin := currentOrigin
	// if we have a next block, and are either forced to adopt it, or just don't want to stay on the old origin, then adopt it.
	if nextOrigin != (eth.L1BlockRef{}) && (l2Timestamp >= nextOrigin.Time || !s.seqOldOrigin) {
		origin = nextOrigin
	}
	s.seqOldOrigin = false

	attr, err := derive.PreparePayloadAttributes(t.Ctx(), s.rollupCfg, s.l1Chain, parent, l2Timestamp, origin.ID())
	require.NoError(t, err, "failed to prepare payload attributes")
	// sequencer may not include anything extra if we run out of drift
	attr.NoTxPool = l2Timestamp >= origin.Time+s.rollupCfg.MaxSequencerDrift

	fc := eth.ForkchoiceState{
		HeadBlockHash:      s.derivation.UnsafeL2Head().Hash,
		SafeBlockHash:      s.derivation.SafeL2Head().Hash,
		FinalizedBlockHash: s.derivation.Finalized().Hash,
	}
	id, errTyp, err := derive.StartPayload(t.Ctx(), s.log, s.eng, fc, attr)
	if err != nil {
		if errTyp == derive.BlockInsertTemporaryErr {
			s.log.Warn("temporary block insertion err", "err", err)
			return
		}
		t.Fatal(err)
	}
	s.l2Building = true
	s.payloadID = id
}

// finish new L2 block, apply to chain as unsafe block
func (s *L2Sequencer) actL2EndBlock(t Testing) {
	if !s.l2Building {
		t.InvalidAction("cannot end L2 block building when no block is being built")
		return
	}
	s.l2Building = false
	fc := eth.ForkchoiceState{
		HeadBlockHash:      s.derivation.UnsafeL2Head().Hash,
		SafeBlockHash:      s.derivation.SafeL2Head().Hash,
		FinalizedBlockHash: s.derivation.Finalized().Hash,
	}
	out, errTyp, err := derive.ConfirmPayload(t.Ctx(), s.log, s.eng, fc, s.payloadID, false)
	if err != nil {
		if errTyp == derive.BlockInsertTemporaryErr {
			s.log.Warn("temporary block insertion err", "err", err)
			return
		}
		t.Fatal(err)
	}
	ref, err := derive.PayloadToBlockRef(out, &s.rollupCfg.Genesis)
	require.NoError(t, err, "payload must convert to block ref")
	s.derivation.SetUnsafeHead(ref)
	// TODO: action-test publishing of payload on p2p
}

// attempt to keep current L1 origin, even if next origin is available
func (s *L2Sequencer) actL2TryKeepL1Origin(t Testing) {
	if s.seqOldOrigin { // don't do this twice
		t.InvalidAction("already decided to keep old L1 origin")
		return
	}
	s.seqOldOrigin = true
}

// make next gossip receive fail
func (s *L2Sequencer) actL2UnsafeGossipFail(t Testing) {
	t.InvalidAction("todo mock sequencer gossip publish fail")
}
