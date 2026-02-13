package health

import (
	"fmt"
	"sync"
)

// this is a type of counter which keeps on incrementing until its reset interval is hit after which it resets to 0
// this can be used to track time-based rate-limit, error counts, etc.
type timeBoundedRotatingCounter struct {
	resetIntervalSeconds uint64
	timeProviderFn       func() uint64

	mut           *sync.RWMutex
	temporalCache map[int64]uint64
}

func NewTimeBoundedRotatingCounter(resetIntervalSeconds uint64) (*timeBoundedRotatingCounter, error) {
	if resetIntervalSeconds == 0 {
		return nil, fmt.Errorf("reset interval seconds must be more than 0")
	}
	return &timeBoundedRotatingCounter{
		resetIntervalSeconds: resetIntervalSeconds,
		mut:                  &sync.RWMutex{},
		temporalCache:        map[int64]uint64{},
		timeProviderFn:       currentTimeProvider,
	}, nil
}

func (t *timeBoundedRotatingCounter) Increment() uint64 {
	// let's take `resetIntervalSeconds` as 60s
	// truncatedTimestamp is current timestamp rounded off by 60s (resetIntervalSeconds)
	// thereby generating a value which stays same until the next 60s helping track and incrementing the counter corresponding to it for the next 60s
	currentTsSeconds := t.timeProviderFn()
	truncatedTimestamp := int64(currentTsSeconds / t.resetIntervalSeconds)
	t.mut.Lock()
	// a lazy cleanup subroutine to the clean the cache when it's grown enough, preventing memory leaks
	defer func() {
		defer t.mut.Unlock()
		if len(t.temporalCache) > 1000 {
			newCache := map[int64]uint64{
				truncatedTimestamp: t.temporalCache[truncatedTimestamp],
			}
			t.temporalCache = newCache // garbage collector should take care of the old cache
		}
	}()

	t.temporalCache[truncatedTimestamp]++
	return t.temporalCache[truncatedTimestamp]
}

func (t *timeBoundedRotatingCounter) CurrentValue() uint64 {
	currentTsSeconds := t.timeProviderFn()
	truncatedTimestamp := int64(currentTsSeconds / t.resetIntervalSeconds)
	t.mut.RLock()
	defer t.mut.RUnlock()
	return t.temporalCache[truncatedTimestamp]
}
