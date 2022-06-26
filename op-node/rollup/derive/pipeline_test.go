package derive

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

var _ Engine = (*testutils.MockEngine)(nil)

var _ L1Fetcher = (*testutils.MockL1Source)(nil)

type MockOriginStage struct {
	mock.Mock
	progress Progress
}

func (m *MockOriginStage) Progress() Progress {
	return m.progress
}

var _ StageProgress = (*MockOriginStage)(nil)

// RepeatResetStep is a test util that will repeat the ResetStep function until an error.
// If the step runs too many times, it will fail the test.
func RepeatResetStep(t *testing.T, step func(ctx context.Context, l1Fetcher L1Fetcher) error, l1Fetcher L1Fetcher, max int) error {
	ctx := context.Background()
	for i := 0; i < max; i++ {
		err := step(ctx, l1Fetcher)
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

// RepeatStep is a test util that will repeat the Step function until an error.
// If the step runs too many times, it will fail the test.
func RepeatStep(t *testing.T, step func(ctx context.Context, outer Progress) error, outer Progress, max int) error {
	ctx := context.Background()
	for i := 0; i < max; i++ {
		err := step(ctx, outer)
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
