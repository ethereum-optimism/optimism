package conductor

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-conductor/health"
)

// latchState is the state of the demotion-grace latch.
type latchState int

const (
	// latchDisarmed is the normal state: health results pass through unchanged.
	latchDisarmed latchState = iota
	// latchArmed means a staleness grace is running. The node reports healthy until
	// the grace expires, at which point it is demoted unconditionally.
	latchArmed
	// latchFired means the demotion has been applied. The node stays unhealthy until
	// a genuine healthy poll returns it to latchDisarmed (re-arming for a future episode).
	latchFired
)

// demotionLatch implements the non-cancellable demotion-grace latch (Phase 7 Fix 1).
//
// With reorg recovery enabled, a leader that goes stale (ErrSequencerNotHealthy) is
// given a fixed grace during which it keeps reporting healthy, so it can still receive
// and commit the op-reth reorg notification before leadership transfers away. At grace
// expiry the node is demoted UNCONDITIONALLY: the latch can never be cancelled, even by
// a healthy poll during the grace, because a fresh block timestamp does not prove the
// node is sequencing the correct chain — it could be extending a reorged-out fork that
// still carries the invalid message. A node oscillating onto a fresh-but-wrong fork must
// not be able to evade demotion by polling healthy.
//
// A healthy result only matters once a demotion has fired: it returns the latch to
// disarmed, re-arming it for a future episode. ErrSequencerConnectionDown and any other
// non-staleness error demote immediately (no grace) — a genuine connection loss should
// fail over fast.
//
// The latch is a pure state machine with no time dependency: arming reports that a timer
// should start, and expire() (driven by that timer in the control loop) performs the
// armed→fired transition. All methods run on the single control-loop goroutine.
type demotionLatch struct {
	// enabled mirrors reorg-recovery being on. When false the latch is inert and every
	// health result passes through unchanged (byte-for-byte today's behavior).
	enabled bool
	state   latchState
	// armErr is the staleness error that armed the latch; re-applied as the active health
	// error when the demotion fires so action() behaves as if it had just arrived.
	armErr error
}

// observe processes a health result. It returns the health error that should be reported
// to the rest of the conductor (nil == healthy). arm reports whether the latch just
// transitioned into the armed state (the caller starts the grace timer); when false and
// the latch is not armed, the caller stops any pending timer.
func (l *demotionLatch) observe(hcerr error) (report error, arm bool) {
	if !l.enabled {
		return hcerr, false
	}

	healthy := hcerr == nil
	staleness := errors.Is(hcerr, health.ErrSequencerNotHealthy)

	switch l.state {
	case latchDisarmed:
		switch {
		case healthy:
			return nil, false
		case staleness:
			// Arm the grace: keep reporting healthy so the leader can commit the reorg
			// notification before staleness would otherwise transfer leadership away.
			l.state = latchArmed
			l.armErr = hcerr
			return nil, true
		default:
			// Connection-down and any other hard failure: demote immediately.
			return hcerr, false
		}
	case latchArmed:
		switch {
		case healthy, staleness:
			// Cannot cancel a pending demotion, and cannot reset the deadline: a healthy
			// poll does not prove a correct chain, and repeated staleness must not extend
			// the grace. Keep reporting healthy until the timer fires.
			return nil, false
		default:
			// A hard failure during the grace demotes immediately.
			l.state = latchFired
			return hcerr, false
		}
	case latchFired:
		if healthy {
			// Genuine recovery after a demotion: return to normal, re-arm-eligible.
			l.state = latchDisarmed
			return nil, false
		}
		// Still unhealthy (staleness or hard): stay demoted.
		return hcerr, false
	default:
		return hcerr, false
	}
}

// expire performs the armed→fired transition when the grace timer fires. It returns the
// health error to apply (the staleness error that armed the latch) and ok=false when the
// latch was not armed (a superseded timer fire), in which case the caller ignores it.
func (l *demotionLatch) expire() (report error, ok bool) {
	if l.state != latchArmed {
		return nil, false
	}
	l.state = latchFired
	return l.armErr, true
}
