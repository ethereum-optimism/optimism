package attributes

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/mock"
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

func (m *MockEngineController) PendingSafeL2Head() eth.L2BlockRef {
	m.Mock.MethodCalled("PendingSafeL2Head")
	return eth.L2BlockRef{}
}

func (m *MockEngineController) SafeL2Head() eth.L2BlockRef {
	m.Mock.MethodCalled("SafeL2Head")
	return eth.L2BlockRef{}
}

func (m *MockEngineController) Finalized() eth.L2BlockRef {
	m.Mock.MethodCalled("Finalized")
	return eth.L2BlockRef{}
}

func (m *MockEngineController) SetPendingSafeL2Head(r eth.L2BlockRef) {
	m.Mock.MethodCalled("SetPendingSafeL2Head", r)
}

func (m *MockEngineController) BackupUnsafeL2Head() eth.L2BlockRef {
	m.Mock.MethodCalled("BackupUnsafeL2Head")
	return eth.L2BlockRef{}
}

func (m *MockEngineController) SetBackupUnsafeL2Head(r eth.L2BlockRef, triggerReorg bool) {
	m.Mock.MethodCalled("SetBackupUnsafeL2Head", r, triggerReorg)
}
