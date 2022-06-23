package derive

import (
	"context"
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

var _ Engine = (*testutils.MockEngine)(nil)

var _ L1Fetcher = (*testutils.MockL1Source)(nil)

type MockOriginStage struct {
	mock.Mock
	originOpen    bool
	currentOrigin eth.L1BlockRef
}

func (m *MockOriginStage) CurrentOrigin() eth.L1BlockRef {
	return m.currentOrigin
}

func (m *MockOriginStage) OpenOrigin(origin eth.L1BlockRef) error {
	m.originOpen = true
	m.currentOrigin = origin
	out := m.Mock.MethodCalled("OpenOrigin", origin)
	return *out[0].(*error)
}

func (m *MockOriginStage) ExpectOpenOrigin(origin eth.L1BlockRef, err error) {
	m.Mock.On("OpenOrigin", origin).Once().Return(&err)
}

func (m *MockOriginStage) CloseOrigin() {
	m.originOpen = false
	m.Mock.MethodCalled("CloseOrigin")
}

func (m *MockOriginStage) ExpectCloseOrigin() {
	m.Mock.On("CloseOrigin").Once().Return()
}

func (m *MockOriginStage) IsOriginOpen() bool {
	return m.originOpen
}

var _ OriginStage = (*MockOriginStage)(nil)

// RepeatResetStep is a test util that will repeat the ResetStep function until an error.
// If the step runs too many times, it will fail the test.
func RepeatResetStep(t *testing.T, step func(ctx context.Context, l1Fetcher L1Fetcher) error,
	l1Fetcher L1Fetcher, max int) error {
	return RepeatStep(t, func(ctx context.Context) error {
		return step(ctx, l1Fetcher)
	}, max)
}

// RepeatStep is a test util that will repeat the Step function until an error.
// If the step runs too many times, it will fail the test.
func RepeatStep(t *testing.T, step func(ctx context.Context) error, max int) error {
	ctx := context.Background()
	for i := 0; i < max; i++ {
		err := step(ctx)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
	t.Fatal("ran out of steps")
	return nil
}
