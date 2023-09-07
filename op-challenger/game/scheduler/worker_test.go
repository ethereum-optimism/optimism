package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"

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
		player: &stubPlayer{status: types.GameStatusInProgress},
	}
	in <- job{
		player: &stubPlayer{status: types.GameStatusDefenderWon},
	}

	result1 := readWithTimeout(t, out)
	result2 := readWithTimeout(t, out)

	require.Equal(t, result1.status, types.GameStatusInProgress)
	require.Equal(t, result2.status, types.GameStatusDefenderWon)

	// Cancel the context which should exit the worker
	cancel()
	wg.Wait()
}

type stubPlayer struct {
	status types.GameStatus
}

func (s *stubPlayer) ProgressGame(ctx context.Context) types.GameStatus {
	return s.status
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
