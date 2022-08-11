package actions

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type L2Verifier struct {
	log log.Logger

	eng derive.Engine

	// L2 rollup
	derivation *derive.DerivationPipeline

	l1Head      eth.L1BlockRef
	l1Safe      eth.L1BlockRef
	l1Finalized eth.L1BlockRef

	l2PipelineIdle bool
	l2Building     bool

	rollupCfg *rollup.Config
}

var _ OutputRootAPI = (*L2Verifier)(nil)

var _ SyncStatusAPI = (*L2Verifier)(nil)

func NewL2Verifier(log log.Logger, l1 derive.L1Fetcher, eng derive.Engine, cfg *rollup.Config) *L2Verifier {
	pipeline := derive.NewDerivationPipeline(log, cfg, l1, eng, TestMetrics{})
	pipeline.Reset()
	return &L2Verifier{
		log:            log,
		eng:            eng,
		derivation:     pipeline,
		l2PipelineIdle: true,
		l2Building:     false,
		rollupCfg:      cfg,
	}
}

func (s *L2Verifier) OutputAtBlock(ctx context.Context, number rpc.BlockNumber) ([]eth.Bytes32, error) {
	return nil, fmt.Errorf("todo OutputAtBlock")
}

func (s *L2Verifier) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return &eth.SyncStatus{
		CurrentL1:   s.derivation.Progress().Origin,
		HeadL1:      s.l1Head,
		SafeL1:      s.l1Safe,
		FinalizedL1: s.l1Finalized,
		UnsafeL2:    s.derivation.UnsafeL2Head(),
		SafeL2:      s.derivation.SafeL2Head(),
		FinalizedL2: s.derivation.Finalized(),
	}, nil
}

// run L2 derivation pipeline
func (s *L2Verifier) actL2PipelineStep(t Testing) {
	if s.l2Building {
		t.InvalidAction("cannot derive new data while building L2 block")
		return
	}

	s.l2PipelineIdle = false
	err := s.derivation.Step(context.Background())
	if err == io.EOF {
		s.l2PipelineIdle = true
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

func (s *L2Verifier) actL2PipelineFull(t Testing) {
	s.l2PipelineIdle = false
	for !s.l2PipelineIdle {
		s.actL2PipelineStep(t)
	}
}

// process payload from gossip
func (s *L2Verifier) actL2UnsafeGossipReceive(t Testing) {
	t.InvalidAction("todo unsafe gossip receive action")
}
