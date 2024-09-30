package backend

import (
	"context"
	"errors"
	"io"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/frontend"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type MockBackend struct {
	started atomic.Bool
}

var _ frontend.Backend = (*MockBackend)(nil)

var _ io.Closer = (*MockBackend)(nil)

func NewMockBackend() *MockBackend {
	return &MockBackend{}
}

func (m *MockBackend) Start(ctx context.Context) error {
	if !m.started.CompareAndSwap(false, true) {
		return errors.New("already started")
	}
	return nil
}

func (m *MockBackend) Stop(ctx context.Context) error {
	if !m.started.CompareAndSwap(true, false) {
		return errors.New("already stopped")
	}
	return nil
}

func (m *MockBackend) AddL2RPC(ctx context.Context, rpc string) error {
	return nil
}

func (m *MockBackend) CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error) {
	return types.CrossUnsafe, nil
}

func (m *MockBackend) CheckMessages(messages []types.Message, minSafety types.SafetyLevel) error {
	return nil
}

func (m *MockBackend) CheckBlock(chainID *hexutil.U256, blockHash common.Hash, blockNumber hexutil.Uint64) (types.SafetyLevel, error) {
	return types.CrossUnsafe, nil
}

func (m *MockBackend) DerivedFrom(ctx context.Context, t types.ChainID, parentHash common.Hash, n uint64) (eth.BlockRef, error) {
	return eth.BlockRef{}, nil
}

func (m *MockBackend) UnsafeView(ctx context.Context, chainID types.ChainID, unsafe types.ReferenceView) (types.ReferenceView, error) {
	return types.ReferenceView{}, nil
}

func (m *MockBackend) SafeView(ctx context.Context, chainID types.ChainID, safe types.ReferenceView) (types.ReferenceView, error) {
	return types.ReferenceView{}, nil
}

func (m *MockBackend) Finalized(ctx context.Context, chainID types.ChainID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}

func (m *MockBackend) UpdateLocalUnsafe(chainID types.ChainID, head eth.L2BlockRef) {
}

func (m *MockBackend) UpdateLocalSafe(chainID types.ChainID, derivedFrom eth.L1BlockRef, lastDerived eth.L2BlockRef) {
}

func (m *MockBackend) UpdateFinalizedL1(chainID types.ChainID, finalized eth.L1BlockRef) {
}

func (m *MockBackend) Close() error {
	return nil
}
