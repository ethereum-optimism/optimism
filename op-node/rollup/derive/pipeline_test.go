package derive

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/ethereum-optimism/optimism/op-node/eth"
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

// TestMetrics implements the metrics used in the derivation pipeline as no-op operations.
// Optionally a test may hook into the metrics
type TestMetrics struct {
	recordL1Ref          func(name string, ref eth.L1BlockRef)
	recordL2Ref          func(name string, ref eth.L2BlockRef)
	recordUnsafePayloads func(length uint64, memSize uint64, next eth.BlockID)
}

func (t *TestMetrics) RecordL1Ref(name string, ref eth.L1BlockRef) {
	if t.recordL1Ref != nil {
		t.recordL1Ref(name, ref)
	}
}

func (t *TestMetrics) RecordL2Ref(name string, ref eth.L2BlockRef) {
	if t.recordL2Ref != nil {
		t.recordL2Ref(name, ref)
	}
}

func (t *TestMetrics) RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID) {
	if t.recordUnsafePayloads != nil {
		t.recordUnsafePayloads(length, memSize, next)
	}
}

var _ Metrics = (*TestMetrics)(nil)
