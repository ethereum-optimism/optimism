package source

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPipelineStage(t *testing.T) {
	t.Run("ProcessEvents", func(t *testing.T) {
		in := make(chan string, 1)
		out := make(chan string, 1)
		handler := func(_ context.Context, e string) {
			out <- e
		}
		stage := NewPipelineStage[string](in, PipelineEventHandlerFn[string](handler))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		require.NoError(t, stage.Start(ctx))
		t.Cleanup(func() {
			require.NoError(t, stage.Stop(ctx))
		})

		in <- "evt1"
		waitForEvent(t, "evt1", out)
		in <- "evt2"
		waitForEvent(t, "evt2", out)
		in <- "evt3"
		waitForEvent(t, "evt3", out)
	})

	t.Run("StopShouldWaitUntilEventLoopExits", func(t *testing.T) {
		in := make(chan string, 1)
		out := make(chan string, 1)

		handler := func(_ context.Context, e string) {
			out <- e
		}
		stage := NewPipelineStage[string](in, PipelineEventHandlerFn[string](handler))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		require.NoError(t, stage.Start(ctx))
		t.Cleanup(func() {
			require.NoError(t, stage.Stop(ctx))
		})

		in <- "evt1"
		waitForEvent(t, "evt1", out)
		in <- "evt2"
		waitForEvent(t, "evt2", out)
		in <- "evt3"
		waitForEvent(t, "evt3", out)
	})
}

func waitForEvent[E any](t *testing.T, expected E, ch <-chan E) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		t.Fatal("timed out waiting for event")
	case actual := <-ch:
		require.Equal(t, expected, actual)
	}
}
