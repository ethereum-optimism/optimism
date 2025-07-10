package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestTaskSync(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	ctx := context.Background()
	ex := NewGlobalSynchronous(ctx)
	sys := NewSystem(logger, ex)
	busA := sys.Register("alice", nil)
	busB := sys.Register("bob", nil)

	onTestEvent0 := func(ctx context.Context, ev TestEvent) error {
		logger.Info("running onTestEvent0 handler")
		if err := Emit(ctx, FooEvent{}); err != nil {
			return err
		}
		return nil
	}
	onTestEvent1 := func(ctx context.Context, ev TestEvent) error {
		logger.Info("running onTestEvent1 handler")
		if err := Emit(ctx, FooEvent{}); err != nil {
			return err
		}
		return nil
	}
	busA.Task(ErrFn[TestEvent](onTestEvent0), Domain("chain0"))
	busA.Task(ErrFn[TestEvent](onTestEvent1), Domain("chain1"))

	onFoo := func(ctx context.Context, ev FooEvent) {
		logger.Info("got foo", "ev", ev)
	}
	onBar := func(ctx context.Context, ev BarEvent) {
		logger.Info("got bar", "ev", ev)
	}
	task := NewTask(busB).
		Handle(Fn[FooEvent](onFoo)).
		Handle(Fn[BarEvent](onBar)).Run(CtxWithDomain(ctx, "chain0"), TestEvent{})
	defer task.Close()
}

func TestTaskAwait(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	ctx := context.Background()
	ex := NewGlobalSynchronous(ctx)
	sys := NewSystem(logger, ex)
	busA := sys.Register("alice", nil)
	busB := sys.Register("bob", nil)

	onTestEvent := func(ctx context.Context, ev TestEvent) error {
		logger.Info("running onTestEvent handler")
		if err := Emit(ctx, FooEvent{}); err != nil {
			return err
		}
		return nil
	}
	busA.Task(ErrFn[TestEvent](onTestEvent))

	onFoo := Await[FooEvent](1)
	onBar := Await[BarEvent](1)

	task := NewTask(busB).
		Handle(onFoo).
		Handle(onBar).Run(ctx, TestEvent{})
	defer task.Close()

	require.NoError(t, ex.Drain())

	select {
	case ev := <-onFoo.C():
		t.Log("got foo", ev)
	case ev := <-onBar.C():
		t.Log("got bar", ev)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
