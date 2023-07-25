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
	require.NoError(t, Do(2, strategy, func() (any, error) {
		if i == 1 {
			return nil, nil
		}

		i++
		return nil, dummyErr
	}))
	require.True(t, time.Since(start) > 10*time.Millisecond)

	start = time.Now()
	// add one because the first attempt counts
	err := Do(3, strategy, func() (any, error) {
		return nil, dummyErr
	})
	require.Equal(t, dummyErr, err.(*ErrFailedPermanently).LastErr)
	require.True(t, time.Since(start) > 20*time.Millisecond)
}

func TestDoResult(t *testing.T) {
	strategy := Fixed(10 * time.Millisecond)
	dummyErr := errors.New("explode")

	start := time.Now()
	var i int
	res, err := DoResult(2, strategy, func() (int, error) {
		if i == 1 {
			return i, nil
		}

		i++
		return i, dummyErr
	})
	require.NoError(t, err)
	require.Equal(t, 1, res)
	require.True(t, time.Since(start) > 10*time.Millisecond)

	start = time.Now()
	// add one because the first attempt counts
	anyValue, err := DoResult(3, strategy, func() (any, error) {
		return nil, dummyErr
	})
	require.Nil(t, anyValue)
	require.Equal(t, dummyErr, err.(*ErrFailedPermanently).LastErr)
	require.True(t, time.Since(start) > 20*time.Millisecond)
}
