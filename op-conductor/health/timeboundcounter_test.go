package health

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimeBoundedRotatingCounterSetup(t *testing.T) {
	t.Parallel()
	t.Run("fail with 0 interval seconds value", func(t *testing.T) {
		counter, err := NewTimeBoundedRotatingCounter(0, 0)
		require.Error(t, err)
		require.Nil(t, counter)
	})

	t.Run("succeed with 0 max value", func(t *testing.T) {
		counter, err := NewTimeBoundedRotatingCounter(2, 0)
		require.NoError(t, err)
		require.NotNil(t, counter)
	})
}

func TestTimeBoundedRotatingCounterIncrement(t *testing.T) {
	t.Parallel()

	mockTimeProvider := &timeProvider{now: 0} // every access to .Now() will increment its value simulating a one-second time passing

	maxValue, resetInterval := uint64(2), uint64(6)
	counter, err := NewTimeBoundedRotatingCounter(resetInterval, maxValue)
	require.NoError(t, err)
	require.NotNil(t, counter)
	counter.timeProviderFn = mockTimeProvider.Now

	require.Equal(t, int(mockTimeProvider.now), 0)
	require.Equal(t, uint64(0), counter.CurrentValue())
	require.Equal(t, int(mockTimeProvider.now), 1)

	newValue, err := counter.Increment()
	require.NoError(t, err)
	require.Equal(t, uint64(1), newValue)
	require.Equal(t, int(mockTimeProvider.now), 2)
	require.Equal(t, uint64(1), counter.CurrentValue())
	require.Equal(t, int(mockTimeProvider.now), 3)

	newValue, err = counter.Increment()
	require.NoError(t, err)
	require.Equal(t, uint64(2), newValue)
	require.Equal(t, int(mockTimeProvider.now), 4)
	require.Equal(t, uint64(2), counter.CurrentValue())
	require.Equal(t, int(mockTimeProvider.now), 5)

	newValue, err = counter.Increment()
	require.Error(t, err)
	require.Equal(t, uint64(2), newValue)
	require.Equal(t, int(mockTimeProvider.now), 6)
	require.Equal(t, uint64(0), counter.CurrentValue()) // the next second counter rotates returning 0 as the current value
	require.Equal(t, int(mockTimeProvider.now), 7)

	newValue, err = counter.Increment()
	require.NoError(t, err)
	require.Equal(t, uint64(1), newValue)
	require.Equal(t, int(mockTimeProvider.now), 8)
	require.Equal(t, uint64(1), counter.CurrentValue())
	require.Equal(t, int(mockTimeProvider.now), 9)

	newValue, err = counter.Increment()
	require.NoError(t, err)
	require.Equal(t, uint64(2), newValue)
	require.Equal(t, int(mockTimeProvider.now), 10)
	require.Equal(t, uint64(2), counter.CurrentValue())
	require.Equal(t, int(mockTimeProvider.now), 11)

	newValue, err = counter.Increment()
	require.Error(t, err)
	require.Equal(t, uint64(2), newValue)
	require.Equal(t, int(mockTimeProvider.now), 12)
	require.Equal(t, uint64(0), counter.CurrentValue()) // the next second counter rotates returning 0 as the current value
	require.Equal(t, int(mockTimeProvider.now), 13)

}

// To test the bad path: comment out mut.RLock() and mut.RUnlock() in the CurrentValue() method, and run this test again
// you'll see a "fatal error: concurrent map read and map write"
func TestTimeBoundedRotatingCounterConcurrentAccess(t *testing.T) {
	mockTimeProvider := &timeProvider{now: 0}

	counter, err := NewTimeBoundedRotatingCounter(1, 9)
	require.NoError(t, err)
	require.NotNil(t, counter)
	counter.timeProviderFn = mockTimeProvider.Now

	wg := &sync.WaitGroup{}
	wg.Add(2000)

	write := func() {
		defer wg.Done()
		defer wg.Done()
		_, err := counter.Increment()
		require.NoError(t, err) // considering the max value is 9, the increment should never fail
	}
	read := func() {
		defer wg.Done()
		counter.CurrentValue()
	}
	require.NotPanics(t, func() {
		for i := 0; i < 1000; i++ {
			go write()
			go read()
		}
		wg.Wait()
	})
}

func TestTimeBoundedRotatingCounterLazyCleanup(t *testing.T) {
	mockTimeProvider := &timeProvider{now: 0}

	// a counter with a reset interval of 2 ensuring every two-seconds the counter's cache would track a new key:value
	// we'll trigger the 2-second increment by calling .Increment() and .CurrentValue() because both under the hood, would call .Now() of the mockTimeProvider
	counter, err := NewTimeBoundedRotatingCounter(2, 9)
	require.NoError(t, err)
	require.NotNil(t, counter)
	counter.timeProviderFn = mockTimeProvider.Now

	for i := 0; i < 1000; i++ {
		_, err := counter.Increment() // trigger a 1-second time increase
		require.NoError(t, err)
		counter.CurrentValue() // trigger another 1-second time increase, causing the counter interval to reset ensuring next Increment would write a new key in the cache
	}

	require.Equal(t, 1000, len(counter.temporalCache))

	// 1001th increment should trigger the lazy cleanup this time
	_, err = counter.Increment()
	require.NoError(t, err)
	require.Equal(t, 1, len(counter.temporalCache))
}
