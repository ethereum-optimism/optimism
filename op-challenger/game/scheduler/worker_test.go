package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkerShouldProcessJobsUntilContextDone(t *testing.T) {
	in := make(chan job, 2)
	out := make(chan job, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go progressGames(ctx, in, out, &wg)

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

	// Cancel the context which should exit the worker
	cancel()
	wg.Wait()
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
