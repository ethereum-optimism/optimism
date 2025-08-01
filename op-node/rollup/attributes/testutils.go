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
