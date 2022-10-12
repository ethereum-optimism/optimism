package proxyd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type FrontendRateLimiter interface {
	// Take consumes a key, and a maximum number of requests
	// per time interval. It returns a boolean denoting if
	// the limit could be taken, or an error if a failure
	// occurred in the backing rate limit implementation.
	//
	// No error will be returned if the limit could not be taken
	// as a result of the requestor being over the limit.
	Take(ctx context.Context, key string, max int) (bool, error)
}

// limitedKeys is a wrapper around a map that stores a truncated
// timestamp and a mutex. The map is used to keep track of rate
// limit keys, and their used limits.
type limitedKeys struct {
	truncTS int64
	keys    map[string]int
	mtx     sync.Mutex
}

func newLimitedKeys(t int64) *limitedKeys {
	return &limitedKeys{
		truncTS: t,
		keys:    make(map[string]int),
	}
}

func (l *limitedKeys) Take(key string, max int) bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	val, ok := l.keys[key]
	if !ok {
		l.keys[key] = 0
		val = 0
	}
	l.keys[key] = val + 1
	return val < max
}

// MemoryFrontendRateLimiter is a rate limiter that stores
// all rate limiting information in local memory. It works
// by storing a limitedKeys struct that references the
// truncated timestamp at which the struct was created. If
// the current truncated timestamp doesn't match what's
// referenced, the limit is reset. Otherwise, values in
// a map are incremented to represent the limit.
type MemoryFrontendRateLimiter struct {
	currGeneration *limitedKeys
	dur            time.Duration
	mtx            sync.Mutex
}

func NewMemoryFrontendRateLimit(dur time.Duration) FrontendRateLimiter {
	return &MemoryFrontendRateLimiter{
		dur: dur,
	}
}

func (m *MemoryFrontendRateLimiter) Take(ctx context.Context, key string, max int) (bool, error) {
	m.mtx.Lock()
	// Create truncated timestamp
	truncTS := truncateNow(m.dur)

	// If there is no current rate limit map or the rate limit map reference
	// a different timestamp, reset limits.
	if m.currGeneration == nil || m.currGeneration.truncTS != truncTS {
		m.currGeneration = newLimitedKeys(truncTS)
	}

	// Pull out the limiter so we can unlock before incrementing the limit.
	limiter := m.currGeneration

	m.mtx.Unlock()

	return limiter.Take(key, max), nil
}

// RedisFrontendRateLimiter is a rate limiter that stores data in Redis.
// It uses the basic rate limiter pattern described on the Redis best
// practices website: https://redis.com/redis-best-practices/basic-rate-limiting/.
type RedisFrontendRateLimiter struct {
	r   *redis.Client
	dur time.Duration
}

func NewRedisFrontendRateLimiter(r *redis.Client, dur time.Duration) FrontendRateLimiter {
	return &RedisFrontendRateLimiter{r: r, dur: dur}
}

func (r *RedisFrontendRateLimiter) Take(ctx context.Context, key string, max int) (bool, error) {
	var incr *redis.IntCmd
	truncTS := truncateNow(r.dur)
	fullKey := fmt.Sprintf("%s:%d", key, truncTS)
	_, err := r.r.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		incr = pipe.Incr(ctx, fullKey)
		pipe.Expire(ctx, fullKey, r.dur-time.Second)
		return nil
	})
	if err != nil {
		return false, err
	}

	return incr.Val()-1 < int64(max), nil
}

// truncateNow truncates the current timestamp
// to the specified duration.
func truncateNow(dur time.Duration) int64 {
	return time.Now().Truncate(dur).Unix()
}
