package backoff

import (
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
	require.NoError(t, Do(2, strategy, func() error {
		if i == 1 {
			return nil
		}

		i++
		return dummyErr
	}))
	require.True(t, time.Since(start) > 10*time.Millisecond)

	start = time.Now()
	// add one because the first attempt counts
	err := Do(3, strategy, func() error {
		return dummyErr
	})
	require.Equal(t, dummyErr, err.(*ErrFailedPermanently).LastErr)
	require.True(t, time.Since(start) > 20*time.Millisecond)
}
