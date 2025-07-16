package driver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestStepSchedulingDeriver(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	var queued []event.Event
	emitter := event.EmitterFunc(func(ctx context.Context, ev event.Event) {
		queued = append(queued, ev)
	})
	sched := NewStepSchedulingDeriver(logger)
	sched.AttachEmitter(emitter)
	require.Len(t, sched.NextStep(), 0, "start empty")
	sched.OnEvent(context.Background(), StepReqEvent{})
	require.Len(t, sched.NextStep(), 1, "take request")
	sched.OnEvent(context.Background(), StepReqEvent{})
	require.Len(t, sched.NextStep(), 1, "ignore duplicate request")
	require.Empty(t, queued, "only scheduled so far, no step attempts yet")
	<-sched.NextStep()
	sched.OnEvent(context.Background(), StepAttemptEvent{})
	require.Equal(t, []event.Event{StepEvent{}}, queued, "got step event")
	require.Nil(t, sched.NextDelayedStep(), "no delayed steps yet")
	sched.OnEvent(context.Background(), StepReqEvent{})
	require.NotNil(t, sched.NextDelayedStep(), "2nd attempt before backoff reset causes delayed step to be scheduled")
	sched.OnEvent(context.Background(), StepReqEvent{})
	require.NotNil(t, sched.NextDelayedStep(), "can continue to request attempts")

	sched.OnEvent(context.Background(), StepReqEvent{})
	require.Len(t, sched.NextStep(), 0, "no step requests accepted without delay if backoff is counting")

	sched.OnEvent(context.Background(), StepReqEvent{ResetBackoff: true})
	require.Len(t, sched.NextStep(), 1, "request accepted if backoff is reset")
	<-sched.NextStep()

	sched.OnEvent(context.Background(), StepReqEvent{})
	require.Len(t, sched.NextStep(), 1, "no backoff, no attempt has been made yet")
	<-sched.NextStep()
	sched.OnEvent(context.Background(), StepAttemptEvent{})
	sched.OnEvent(context.Background(), StepReqEvent{})
	require.Len(t, sched.NextStep(), 0, "backoff again")

	sched.OnEvent(context.Background(), ResetStepBackoffEvent{})
	sched.OnEvent(context.Background(), StepReqEvent{})
	require.Len(t, sched.NextStep(), 1, "reset backoff accepted, was able to schedule non-delayed step")
}
