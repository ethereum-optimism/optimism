package client

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// SequencerControl defines the interface for controlling the sequencer.
//
//go:generate mockery --name SequencerControl --output mocks/ --with-expecter=true
type SequencerControl interface {
	StartSequencer(ctx context.Context, hash common.Hash) error
	StopSequencer(ctx context.Context) (common.Hash, error)
	SequencerActive(ctx context.Context) (bool, error)
	LatestUnsafeBlock(ctx context.Context) (eth.BlockInfo, error)
	PostUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error
	ConductorEnabled(ctx context.Context) (bool, error)
}

// NewSequencerControl creates a new SequencerControl instance.
func NewSequencerControl(exec *sources.EthClient, node *sources.RollupClient) SequencerControl {
	return &sequencerController{
		exec: exec,
		node: node,
	}
}

type sequencerController struct {
	exec *sources.EthClient
	node *sources.RollupClient
}

var _ SequencerControl = (*sequencerController)(nil)

// LatestUnsafeBlock implements SequencerControl.
func (s *sequencerController) LatestUnsafeBlock(ctx context.Context) (eth.BlockInfo, error) {
	return s.exec.InfoByLabel(ctx, eth.Unsafe)
}

// StartSequencer implements SequencerControl.
func (s *sequencerController) StartSequencer(ctx context.Context, hash common.Hash) error {
	return s.node.StartSequencer(ctx, hash)
}

// StopSequencer implements SequencerControl.
func (s *sequencerController) StopSequencer(ctx context.Context) (common.Hash, error) {
	return s.node.StopSequencer(ctx)
}

// SequencerActive implements SequencerControl.
func (s *sequencerController) SequencerActive(ctx context.Context) (bool, error) {
	return s.node.SequencerActive(ctx)
}

// PostUnsafePayload implements SequencerControl.
func (s *sequencerController) PostUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	return s.node.PostUnsafePayload(ctx, payload)
}

// ConductorEnabled implements SequencerControl.
func (s *sequencerController) ConductorEnabled(ctx context.Context) (bool, error) {
	return s.node.ConductorEnabled(ctx)
}
