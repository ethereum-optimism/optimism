package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShouldProcessJobs(t *testing.T) {
	in := make(chan job, 2)
	out := make(chan job, 2)
	w := &worker{
		in:  in,
		out: out,
	}
	w.Start(context.Background())

	in <- job{
		player: &stubPlayer{done: false},
	}
	in <- job{
		player: &stubPlayer{done: true},
	}

	result1 := readWithTimeout(t, out)
	result2 := readWithTimeout(t, out)

	require.Equal(t, result1.resolved, false)
	require.Equal(t, result2.resolved, true)
	defer w.Stop()
}

func TestWorkerStopsWhenContextDone(t *testing.T) {
	in := make(chan job, 2)
	out := make(chan job, 2)
	w := &worker{
		in:  in,
		out: out,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// Make sure the worker is up and running
	in <- job{
		player: &stubPlayer{done: false},
	}
	readWithTimeout(t, out)

	// Cancel the context which should exit the worker
	cancel()
	w.wg.Wait()
}

type stubPlayer struct {
	done bool
}

func (s *stubPlayer) ProgressGame(ctx context.Context) bool {
	return s.done
}

func readWithTimeout[T any](t *testing.T, ch <-chan T) T {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		var val T
		t.Fatal("Did not receive event from channel")
		return val // Won't be reached but makes the compiler happy
	case val := <-ch:
		return val
	}
}
