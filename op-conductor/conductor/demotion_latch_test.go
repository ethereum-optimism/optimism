package conductor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-conductor/health"
)

// The latch is a pure state machine: observe() processes scripted health results and
// expire() simulates the grace timer firing. Driving expire() manually stands in for a
// fake clock — the test controls exactly when the grace elapses.

func TestDemotionLatchDisabledPassThrough(t *testing.T) {
	l := &demotionLatch{enabled: false}

	// Every result passes through unchanged; nothing ever arms.
	report, arm := l.observe(nil)
	require.NoError(t, report)
	require.False(t, arm)

	report, arm = l.observe(health.ErrSequencerNotHealthy)
	require.ErrorIs(t, report, health.ErrSequencerNotHealthy, "disabled latch must not mask staleness")
	require.False(t, arm)

	report, arm = l.observe(health.ErrSequencerConnectionDown)
	require.ErrorIs(t, report, health.ErrSequencerConnectionDown)
	require.False(t, arm)
}

func TestDemotionLatchStaleWithinGraceReportsHealthy(t *testing.T) {
	l := &demotionLatch{enabled: true}

	// First staleness arms the grace and reports healthy.
	report, arm := l.observe(health.ErrSequencerNotHealthy)
	require.NoError(t, report, "stale leader must report healthy during the grace")
	require.True(t, arm)
	require.Equal(t, latchArmed, l.state)

	// Repeated staleness keeps reporting healthy and does not re-arm (no deadline reset).
	report, arm = l.observe(health.ErrSequencerNotHealthy)
	require.NoError(t, report)
	require.False(t, arm, "repeated staleness must not re-arm the timer")
	require.Equal(t, latchArmed, l.state)
}

func TestDemotionLatchExpiryDemotes(t *testing.T) {
	l := &demotionLatch{enabled: true}

	_, arm := l.observe(health.ErrSequencerNotHealthy)
	require.True(t, arm)

	// Grace elapses → unconditional demotion with the original staleness error.
	report, ok := l.expire()
	require.True(t, ok)
	require.ErrorIs(t, report, health.ErrSequencerNotHealthy)
	require.Equal(t, latchFired, l.state)
}

func TestDemotionLatchHealthyDuringGraceStillDemotes(t *testing.T) {
	l := &demotionLatch{enabled: true}

	_, arm := l.observe(health.ErrSequencerNotHealthy)
	require.True(t, arm)

	// A healthy poll during the grace must NOT cancel the pending demotion: a fresh
	// timestamp does not prove a correct chain.
	report, arm := l.observe(nil)
	require.NoError(t, report)
	require.False(t, arm)
	require.Equal(t, latchArmed, l.state, "healthy poll must not disarm the latch")

	report, ok := l.expire()
	require.True(t, ok)
	require.ErrorIs(t, report, health.ErrSequencerNotHealthy, "must still demote at expiry despite the healthy poll")
}

func TestDemotionLatchOscillationDoesNotEvade(t *testing.T) {
	l := &demotionLatch{enabled: true}

	_, arm := l.observe(health.ErrSequencerNotHealthy)
	require.True(t, arm)

	// Oscillate stale↔healthy throughout the grace; none of it can cancel the latch.
	for i := 0; i < 5; i++ {
		report, arm := l.observe(nil)
		require.NoError(t, report)
		require.False(t, arm)
		report, arm = l.observe(health.ErrSequencerNotHealthy)
		require.NoError(t, report)
		require.False(t, arm)
		require.Equal(t, latchArmed, l.state)
	}

	_, ok := l.expire()
	require.True(t, ok, "oscillating onto a fresh-but-wrong fork must not evade demotion")
}

func TestDemotionLatchConnectionDownImmediate(t *testing.T) {
	l := &demotionLatch{enabled: true}

	// Connection-down from disarmed: immediate unhealthy, no arming.
	report, arm := l.observe(health.ErrSequencerConnectionDown)
	require.ErrorIs(t, report, health.ErrSequencerConnectionDown)
	require.False(t, arm)
	require.Equal(t, latchDisarmed, l.state)
}

func TestDemotionLatchConnectionDownDuringGraceDemotesNow(t *testing.T) {
	l := &demotionLatch{enabled: true}

	_, arm := l.observe(health.ErrSequencerNotHealthy)
	require.True(t, arm)

	// A hard failure during the grace short-circuits to immediate demotion.
	report, arm := l.observe(health.ErrSequencerConnectionDown)
	require.ErrorIs(t, report, health.ErrSequencerConnectionDown)
	require.False(t, arm)
	require.Equal(t, latchFired, l.state)

	// The grace timer, if it fires afterward, is a superseded no-op.
	_, ok := l.expire()
	require.False(t, ok)
}

func TestDemotionLatchReArmsAfterRecovery(t *testing.T) {
	l := &demotionLatch{enabled: true}

	_, _ = l.observe(health.ErrSequencerNotHealthy)
	_, ok := l.expire()
	require.True(t, ok)
	require.Equal(t, latchFired, l.state)

	// Still unhealthy while staleness persists.
	report, arm := l.observe(health.ErrSequencerNotHealthy)
	require.ErrorIs(t, report, health.ErrSequencerNotHealthy)
	require.False(t, arm)
	require.Equal(t, latchFired, l.state)

	// Genuine recovery returns to disarmed.
	report, arm = l.observe(nil)
	require.NoError(t, report)
	require.False(t, arm)
	require.Equal(t, latchDisarmed, l.state)

	// A fresh staleness episode arms again.
	_, arm = l.observe(health.ErrSequencerNotHealthy)
	require.True(t, arm, "latch must re-arm for a future episode after recovery")
	require.Equal(t, latchArmed, l.state)
}

func TestDemotionLatchExpireWhenNotArmedIsNoOp(t *testing.T) {
	l := &demotionLatch{enabled: true}
	report, ok := l.expire()
	require.False(t, ok)
	require.NoError(t, report)
	require.Equal(t, latchDisarmed, l.state)
}

func TestDemotionLatchNonStalenessErrorNotGraced(t *testing.T) {
	l := &demotionLatch{enabled: true}

	// A non-staleness, non-connection-down error (e.g. rollup-boost) is treated as a hard
	// failure: immediate, not graced.
	report, arm := l.observe(errors.New("some other unhealthy condition"))
	require.Error(t, report)
	require.False(t, arm)
	require.Equal(t, latchDisarmed, l.state)
}
