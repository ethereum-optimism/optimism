package driver

import (
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

type ResetStepBackoffEvent struct {
}

func (ev ResetStepBackoffEvent) String() string {
	return "reset-step-backoff"
}

type StepDelayedReqEvent struct {
	Delay time.Duration
}

func (ev StepDelayedReqEvent) String() string {
	return "step-delayed-req"
}

type StepReqEvent struct {
	ResetBackoff bool
}

func (ev StepReqEvent) String() string {
	return "step-req"
}

type StepAttemptEvent struct{}

func (ev StepAttemptEvent) String() string {
	return "step-attempt"
}

type StepEvent struct{}

func (ev StepEvent) String() string {
	return "step"
}

// StepSchedulingDeriver is a deriver that emits StepEvent events.
// The deriver can be requested to schedule a step with a StepReqEvent.
//
// It is then up to the caller to translate scheduling into StepAttemptEvent emissions, by waiting for
// NextStep or NextDelayedStep channels (nil if there is nothing to wait for, for channel-merging purposes).
//
// Upon StepAttemptEvent the scheduler will then emit a StepEvent,
// while maintaining backoff state, to not spam steps.
//
// Backoff can be reset by sending a request with StepReqEvent.ResetBackoff
// set to true, or by sending a ResetStepBackoffEvent.
type StepSchedulingDeriver struct {

	// keep track of consecutive failed attempts, to adjust the backoff time accordingly
	stepAttempts int
	bOffStrategy retry.Strategy

	// channel, nil by default (not firing), but used to schedule re-attempts with delay
	delayedStepReq <-chan time.Time

	// stepReqCh is used to request that the driver attempts to step forward by one L1 block.
	stepReqCh chan struct{}

	log log.Logger

	emitter event.Emitter
}

func NewStepSchedulingDeriver(log log.Logger) *StepSchedulingDeriver {
	return &StepSchedulingDeriver{
		stepAttempts:   0,
		bOffStrategy:   retry.Exponential(),
		stepReqCh:      make(chan struct{}, 1),
		delayedStepReq: nil,
		log:            log,
	}
}

func (s *StepSchedulingDeriver) AttachEmitter(em event.Emitter) {
	s.emitter = em
}

// NextStep is a channel to await, and if triggered,
// the caller should emit a StepAttemptEvent to queue up a step while maintaining backoff.
func (s *StepSchedulingDeriver) NextStep() <-chan struct{} {
	return s.stepReqCh
}

// NextDelayedStep is a temporary channel to await, and if triggered,
// the caller should emit a StepAttemptEvent to queue up a step while maintaining backoff.
// The returned channel may be nil, if there is no requested step with delay scheduled.
func (s *StepSchedulingDeriver) NextDelayedStep() <-chan time.Time {
	return s.delayedStepReq
}

func (s *StepSchedulingDeriver) OnEvent(ev event.Event) bool {
	step := func() {
		s.delayedStepReq = nil
		select {
		case s.stepReqCh <- struct{}{}:
		// Don't deadlock if the channel is already full
		default:
		}
	}

	switch x := ev.(type) {
	case StepDelayedReqEvent:
		if s.delayedStepReq == nil {
			s.delayedStepReq = time.After(x.Delay)
		}
	case StepReqEvent:
		if x.ResetBackoff {
			s.stepAttempts = 0
		}
		if s.stepAttempts > 0 {
			// if this is not the first attempt, we re-schedule with a backoff, *without blocking other events*
			if s.delayedStepReq == nil {
				delay := s.bOffStrategy.Duration(s.stepAttempts)
				s.log.Debug("scheduling re-attempt with delay", "attempts", s.stepAttempts, "delay", delay)
				s.delayedStepReq = time.After(delay)
			} else {
				s.log.Debug("ignoring step request, already scheduled re-attempt after previous failure", "attempts", s.stepAttempts)
			}
		} else {
			step()
		}
	case StepAttemptEvent:
		// clear the delayed-step channel
		s.delayedStepReq = nil
		if s.stepAttempts > 0 {
			s.log.Debug("Running step retry", "attempts", s.stepAttempts)
		}
		// count as attempt by default. We reset to 0 if we are making healthy progress.
		s.stepAttempts += 1
		s.emitter.Emit(StepEvent{})
	case ResetStepBackoffEvent:
		s.stepAttempts = 0
	default:
		return false
	}
	return true
}
