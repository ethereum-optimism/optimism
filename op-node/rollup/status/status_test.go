package status

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type NoopMetrics struct{}

func (m NoopMetrics) RecordL1ReorgDepth(d uint64)                 {}
func (m NoopMetrics) RecordL1Ref(name string, ref eth.L1BlockRef) {}

func TestStatus(t *testing.T) {

	tracker := NewStatusTracker(testlog.Logger(t, log.LevelDebug), NoopMetrics{})

	initialStatus := tracker.SyncStatus()
	require.Equal(t, eth.SyncStatus{}, *initialStatus)

	oneHundred := eth.L2BlockRef{Number: 100}
	tracker.OnEvent(engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    oneHundred,
		SafeL2Head:      oneHundred,
		FinalizedL2Head: oneHundred,
	})

	finalStatus := tracker.SyncStatus()

	// this is a general invariant which should hold both pre and post interop
	require.GreaterOrEqual(t, finalStatus.LocalSafeL2.Number, finalStatus.SafeL2.Number)

	// If this were to happen while other fields remain nonzero
	// the batcher might try and load blocks from genesis
	// which would cause a major issue:
	require.NotZero(t, finalStatus.LocalSafeL2.Number)
}
