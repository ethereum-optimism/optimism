package conductor

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	clientmocks "github.com/ethereum-optimism/optimism/op-conductor/client/mocks"
	consensusmocks "github.com/ethereum-optimism/optimism/op-conductor/consensus/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/health"
	healthmocks "github.com/ethereum-optimism/optimism/op-conductor/health/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// newWiringConductor builds an OpConductor wired for demotion-latch tests, with reorg
// recovery toggled by `enabled`. It mirrors Start's demotion-timer creation so
// handleHealthUpdate/handleDemotionExpiry can be driven directly without the full loop.
func newWiringConductor(t *testing.T, enabled bool) *OpConductor {
	cfg := mockConfig(t)
	cfg.ReorgRecoveryEnabled = enabled
	cfg.ExecutionWSURL = "ws://test:8546" // required by Config.Check when enabled

	ctrl := &clientmocks.SequencerControl{}
	cons := &consensusmocks.Consensus{}
	hmon := &healthmocks.HealthMonitor{}
	cons.EXPECT().ServerID().Return("SequencerA").Maybe()

	oc, err := NewOpConductor(context.Background(), &cfg, testlog.Logger(t, log.LevelDebug),
		&metrics.NoopMetricsImpl{}, "v0.0.1", ctrl, cons, hmon, nil)
	require.NoError(t, err)

	if enabled {
		oc.demotionTimer = time.NewTimer(reorgDemotionGrace)
		if !oc.demotionTimer.Stop() {
			<-oc.demotionTimer.C
		}
	}
	return oc
}

func TestHandleHealthUpdateFlagOffImmediateDemotion(t *testing.T) {
	oc := newWiringConductor(t, false)
	require.True(t, oc.healthy.Load())

	// With the flag off, staleness demotes immediately — today's behavior, unchanged.
	oc.handleHealthUpdate(health.ErrSequencerNotHealthy)
	require.False(t, oc.healthy.Load())
	require.ErrorIs(t, oc.hcerr, health.ErrSequencerNotHealthy)
}

func TestHandleHealthUpdateGracedStalenessStaysHealthyThenDemotes(t *testing.T) {
	oc := newWiringConductor(t, true)
	require.True(t, oc.healthy.Load())

	// Staleness arms the grace; the leader keeps reporting healthy.
	oc.handleHealthUpdate(health.ErrSequencerNotHealthy)
	require.True(t, oc.healthy.Load(), "stale leader must stay healthy during the grace")
	require.True(t, oc.demotionTimerArmed)
	require.Equal(t, latchArmed, oc.latch.state)

	// A healthy poll during the grace does not cancel the latch.
	oc.handleHealthUpdate(nil)
	require.True(t, oc.healthy.Load())
	require.True(t, oc.demotionTimerArmed)
	require.Equal(t, latchArmed, oc.latch.state)

	// Grace expiry demotes unconditionally and surfaces the staleness error to action().
	oc.handleDemotionExpiry()
	require.False(t, oc.healthy.Load(), "must demote at grace expiry")
	require.ErrorIs(t, oc.hcerr, health.ErrSequencerNotHealthy)
	require.False(t, oc.demotionTimerArmed)

	// Genuine recovery returns to healthy and re-arms for the future.
	oc.handleHealthUpdate(nil)
	require.True(t, oc.healthy.Load())
	require.Equal(t, latchDisarmed, oc.latch.state)
}

func TestHandleHealthUpdateGracedConnectionDownImmediate(t *testing.T) {
	oc := newWiringConductor(t, true)

	// Connection-down is not graced even with the flag on: fail over fast.
	oc.handleHealthUpdate(health.ErrSequencerConnectionDown)
	require.False(t, oc.healthy.Load())
	require.ErrorIs(t, oc.hcerr, health.ErrSequencerConnectionDown)
	require.False(t, oc.demotionTimerArmed)
}

// TestDemotionTimerFiresIntoLoop exercises the real timer firing through the control loop:
// a stale poll arms the grace, and after the grace elapses the loop demotes without any
// further health update (covering the fresh-but-wrong-fork case where polls stop arriving).
func TestDemotionTimerFiresIntoLoop(t *testing.T) {
	oc := newWiringConductor(t, true)

	// Arm via the real handler, then run a single loop iteration that should block on the
	// timer and demote when it fires. Use the real (5s) grace but bound the wait generously.
	oc.handleHealthUpdate(health.ErrSequencerNotHealthy)
	require.True(t, oc.healthy.Load())
	require.True(t, oc.demotionTimerArmed)

	done := make(chan struct{})
	go func() {
		// loopAction selects the demotion-timer case once the grace elapses.
		oc.loopAction()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(reorgDemotionGrace + 5*time.Second):
		t.Fatal("demotion timer did not fire into the loop within grace + margin")
	}
	require.False(t, oc.healthy.Load(), "loop must demote when the grace timer fires")
	require.ErrorIs(t, oc.hcerr, health.ErrSequencerNotHealthy)
}
