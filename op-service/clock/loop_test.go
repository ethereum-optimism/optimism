package clock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoopFn(t *testing.T) {
	cl := NewDeterministicClock(time.Now())
	calls := make(chan struct{}, 10)
	testErr := errors.New("test close error")
	loopFn := NewLoopFn(cl, func(ctx context.Context) {
		calls <- struct{}{}
	}, func() error {
		close(calls)
		return testErr
	}, time.Second*10)
	cl.AdvanceTime(time.Second * 15)
	<-calls
	cl.AdvanceTime(time.Second * 10)
	<-calls
	select {
	case <-calls:
		t.Fatal("more calls than expected")
	default:
	}
	require.ErrorIs(t, loopFn.Close(), testErr)
}
