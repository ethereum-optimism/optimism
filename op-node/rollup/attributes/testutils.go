package attributes

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MockEngineController struct {
	mock.Mock
}

var _ EngineController = (*MockEngineController)(nil)

func (m *MockEngineController) TryUpdatePendingSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	m.Mock.MethodCalled("TryUpdatePendingSafe", ctx, ref, concluding, source)
}

func (m *MockEngineController) TryUpdateLocalSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	m.Mock.MethodCalled("TryUpdateLocalSafe", ctx, ref, concluding, source)
}

func (m *MockEngineController) RequestForkchoiceUpdate(ctx context.Context) {
	m.Mock.MethodCalled("RequestForkchoiceUpdate", ctx)
}

func (m *MockEngineController) RequestPendingSafeUpdate(ctx context.Context) {
	m.Mock.MethodCalled("RequestPendingSafeUpdate", ctx)
}

func (m *MockEngineController) IsEngineInitialELSyncing() bool {
	out := m.Mock.MethodCalled("IsEngineInitialELSyncing")
	return out.Bool(0)
}

func (m *MockEngineController) IsDenied(blockNumber uint64, payloadHash common.Hash) (bool, error) {
	out := m.Mock.MethodCalled("IsDenied", blockNumber, payloadHash)
	return out.Bool(0), out.Error(1)
}
