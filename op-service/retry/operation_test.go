package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	strategy := Fixed(10 * time.Millisecond)
	dummyErr := errors.New("explode")

	start := time.Now()
	var i int
	_, err := Do(context.Background(), 2, strategy, func() (int, error) {
		if i == 1 {
			return 0, nil
		}
		i++
		return 0, dummyErr
	})
	require.NoError(t, err)
	require.True(t, time.Since(start) > 10*time.Millisecond)
	start = time.Now()
	// add one because the first attempt counts
	_, err = Do(context.Background(), 3, strategy, func() (int, error) {
		return 0, dummyErr
	})
	require.Equal(t, dummyErr, err.(*ErrFailedPermanently).LastErr)
	require.True(t, time.Since(start) > 20*time.Millisecond)
}
