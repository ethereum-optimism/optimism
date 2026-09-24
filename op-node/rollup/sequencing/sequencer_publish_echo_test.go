package sequencing

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
)

func TestSequencerPreservesBacklogOnDelayedForkchoiceEcho(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, net, queued, _ := newPublishQueueSetup(t)
		// StartBuild emits the build parent as a forkchoice update. It may
		// arrive after ProcessPayload has already advanced s.unsafeHead inline.
		// That event is an old snapshot, not a canonical-chain rewind.
		deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: s.head})
		require.NoError(t, net.firstCtx.Err(), "a delayed build-parent echo must not cancel publishing")
		close(net.gate)
		synctest.Wait()
		require.Equal(t, queued, net.published)
	})
}

func TestSequencerPublishInvalidationPrecedesForkchoiceDelivery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, net, _, ec := newPublishQueueSetup(t)
		// A real rewind invalidates the publisher immediately, even if the
		// sequencer is busy and has not replayed any notification about it yet.
		ec.SetUnsafeHead(s.head)
		require.NotEqual(t, ec.UnsafeL2Head(), s.seq.unsafeHead)
		require.ErrorIs(t, net.firstCtx.Err(), context.Canceled)
		synctest.Wait()
		require.Empty(t, net.published)
		require.Equal(t, 1, net.attempts, "no abandoned descendant is attempted")
	})
}
