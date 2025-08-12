package testutils

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/mock"
)

type MockEngDeriver struct {
	mock.Mock
}

func (m *MockEngDeriver) TryUpdatePendingSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	m.Mock.MethodCalled("TryUpdatePendingSafe", ctx, ref, concluding, source)
}

func (m *MockEngDeriver) TryUpdateLocalSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
	m.Mock.MethodCalled("TryUpdateLocalSafe", ctx, ref, concluding, source)
}
