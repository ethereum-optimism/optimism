package driver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestSyncDeriverStepScheduling(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	var queued []event.Event

	// Create a unified SyncDeriver with step scheduling functionality
	syncDeriver := &SyncDeriver{
		Log:            logger,
		stepAttempts:   0,
		bOffStrategy:   retry.Exponential(),
		stepReqCh:      make(chan struct{}, 1),
		delayedStepReq: nil,
	}

	// Test the step scheduling functionality
	require.Len(t, syncDeriver.NextStep(), 0, "start empty")
	syncDeriver.RequestStep(context.Background(), false)
	require.Len(t, syncDeriver.NextStep(), 1, "take request")
	syncDeriver.RequestStep(context.Background(), false)
	require.Len(t, syncDeriver.NextStep(), 1, "ignore duplicate request")
	require.Empty(t, queued, "only scheduled so far, no step attempts yet")
	<-syncDeriver.NextStep()
	syncDeriver.AttemptStep(context.Background())
	require.Nil(t, syncDeriver.NextDelayedStep(), "no delayed steps yet")
	syncDeriver.RequestStep(context.Background(), false)
	require.NotNil(t, syncDeriver.NextDelayedStep(), "2nd attempt before backoff reset causes delayed step to be scheduled")
	syncDeriver.RequestStep(context.Background(), false)
	require.NotNil(t, syncDeriver.NextDelayedStep(), "can continue to request attempts")

	syncDeriver.RequestStep(context.Background(), false)
	require.Len(t, syncDeriver.NextStep(), 0, "no step requests accepted without delay if backoff is counting")

	syncDeriver.RequestStep(context.Background(), true)
	require.Len(t, syncDeriver.NextStep(), 1, "request accepted if backoff is reset")
	<-syncDeriver.NextStep()

	syncDeriver.RequestStep(context.Background(), false)
	require.Len(t, syncDeriver.NextStep(), 1, "no backoff, no attempt has been made yet")
	<-syncDeriver.NextStep()
	syncDeriver.AttemptStep(context.Background())
	syncDeriver.RequestStep(context.Background(), false)
	require.Len(t, syncDeriver.NextStep(), 0, "backoff again")

	syncDeriver.ResetStepBackoff(context.Background())
	syncDeriver.RequestStep(context.Background(), false)
	require.Len(t, syncDeriver.NextStep(), 1, "reset backoff accepted, was able to schedule non-delayed step")
}
