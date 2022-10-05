package actions

import (
	"errors"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

// L2Verifier is an actor that functions like a rollup node,
// without the full P2P/API/Node stack, but just the derivation state, and simplified driver.
type L2Verifier struct {
	log log.Logger

	eng derive.Engine

	// L2 rollup
	derivation *derive.DerivationPipeline

	l1      derive.L1Fetcher
	l1State *driver.L1State

	l2PipelineIdle bool
	l2Building     bool

	rollupCfg *rollup.Config
}

func NewL2Verifier(log log.Logger, l1 derive.L1Fetcher, eng derive.Engine, cfg *rollup.Config) *L2Verifier {
	metrics := &testutils.TestDerivationMetrics{}
	pipeline := derive.NewDerivationPipeline(log, cfg, l1, eng, metrics)
	pipeline.Reset()
	return &L2Verifier{
		log:            log,
		eng:            eng,
		derivation:     pipeline,
		l1:             l1,
		l1State:        driver.NewL1State(log, metrics),
		l2PipelineIdle: true,
		l2Building:     false,
		rollupCfg:      cfg,
	}
}

func (s *L2Verifier) SyncStatus() *eth.SyncStatus {
	return &eth.SyncStatus{
		CurrentL1:   s.derivation.Origin(),
		HeadL1:      s.l1State.L1Head(),
		SafeL1:      s.l1State.L1Safe(),
		FinalizedL1: s.l1State.L1Finalized(),
		UnsafeL2:    s.derivation.UnsafeL2Head(),
		SafeL2:      s.derivation.SafeL2Head(),
		FinalizedL2: s.derivation.Finalized(),
	}
}

func (s *L2Verifier) ActL1HeadSignal(t Testing) {
	head, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	s.l1State.HandleNewL1HeadBlock(head)
}

func (s *L2Verifier) ActL1SafeSignal(t Testing) {
	head, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Safe)
	require.NoError(t, err)
	s.l1State.HandleNewL1SafeBlock(head)
}

func (s *L2Verifier) ActL1FinalizedSignal(t Testing) {
	head, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Finalized)
	require.NoError(t, err)
	s.l1State.HandleNewL1FinalizedBlock(head)
}

// ActL2PipelineStep runs one iteration of the L2 derivation pipeline
func (s *L2Verifier) ActL2PipelineStep(t Testing) {
	if s.l2Building {
		t.InvalidAction("cannot derive new data while building L2 block")
		return
	}

	s.l2PipelineIdle = false
	err := s.derivation.Step(t.Ctx())
	if err == io.EOF {
		s.l2PipelineIdle = true
		return
	} else if err != nil && errors.Is(err, derive.NotEnoughData) {
		return
	} else if err != nil && errors.Is(err, derive.ErrReset) {
		s.log.Warn("Derivation pipeline is reset", "err", err)
		s.derivation.Reset()
		return
	} else if err != nil && errors.Is(err, derive.ErrTemporary) {
		s.log.Warn("Derivation process temporary error", "err", err)
		return
	} else if err != nil && errors.Is(err, derive.ErrCritical) {
		t.Fatalf("derivation failed critically: %v", err)
	} else {
		return
	}
}

func (s *L2Verifier) ActL2PipelineFull(t Testing) {
	s.l2PipelineIdle = false
	for !s.l2PipelineIdle {
		s.ActL2PipelineStep(t)
	}
}

// ActL2UnsafeGossipReceive creates an action that can receive an unsafe execution payload, like gossipsub
func (s *L2Verifier) ActL2UnsafeGossipReceive(payload *eth.ExecutionPayload) Action {
	return func(t Testing) {
		s.derivation.AddUnsafePayload(payload)
	}
}
