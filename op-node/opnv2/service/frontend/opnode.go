package frontend

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type OptimismFrontend struct {
}

func (f *OptimismFrontend) OutputAtBlock(ctx context.Context, number hexutil.Uint64) (*eth.OutputResponse, error) {
	return nil, ErrNotImplemented
}
func (f *OptimismFrontend) SafeHeadAtL1Block(ctx context.Context, number hexutil.Uint64) (*eth.SafeHeadResponse, error) {
	return nil, ErrNotImplemented
}
func (f *OptimismFrontend) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return nil, ErrNotImplemented
}
func (f *OptimismFrontend) RollupConfig(_ context.Context) (*rollup.Config, error) {
	return nil, ErrNotImplemented
}
func (f *OptimismFrontend) DependencySet(_ context.Context) (depset.DependencySet, error) {
	return nil, ErrNotImplemented
}
func (f *OptimismFrontend) Version(ctx context.Context) (string, error) {
	return "", ErrNotImplemented
}

type OpnodeAdminFrontend struct {
}

func (f *OpnodeAdminFrontend) ResetDerivationPipeline(ctx context.Context) error {
	return ErrNotImplemented
}
func (f *OpnodeAdminFrontend) StartSequencer(ctx context.Context, blockHash common.Hash) error {
	return ErrNotImplemented
}
func (f *OpnodeAdminFrontend) StopSequencer(ctx context.Context) (common.Hash, error) {
	return common.Hash{}, ErrNotImplemented
}
func (f *OpnodeAdminFrontend) SequencerActive(ctx context.Context) (bool, error) {
	return false, ErrNotImplemented
}
func (f *OpnodeAdminFrontend) PostUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	return ErrNotImplemented
}
func (f *OpnodeAdminFrontend) OverrideLeader(ctx context.Context) error {
	return ErrNotImplemented
}
func (f *OpnodeAdminFrontend) ConductorEnabled(ctx context.Context) (bool, error) {
	return false, ErrNotImplemented
}
func (f *OpnodeAdminFrontend) SetRecoverMode(ctx context.Context, mode bool) error {
	return ErrNotImplemented
}

type OpstackFrontend struct {
}

// TODO: p2p api from p2p package
