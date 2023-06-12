package avg_sliding_window

import (
	"sync"
	"time"

	lm "github.com/emirpasic/gods/maps/linkedhashmap"
)

type Clock interface {
	Now() time.Time
}

// DefaultClock provides a clock that gets current time from the system time
type DefaultClock struct{}

func NewDefaultClock() *DefaultClock {
	return &DefaultClock{}
}
func (c DefaultClock) Now() time.Time {
	return time.Now()
}

// AdjustableClock provides a static clock to easily override the system time
type AdjustableClock struct {
	now time.Time
}

func NewAdjustableClock(now time.Time) *AdjustableClock {
	return &AdjustableClock{now: now}
}
func (c *AdjustableClock) Now() time.Time {
	return c.now
}
func (c *AdjustableClock) Set(now time.Time) {
	c.now = now
}

type bucket struct {
	sum float64
	qty uint
}

// AvgSlidingWindow calculates moving averages efficiently.
// Data points are rounded to nearest bucket of size `bucketSize`,
// and evicted when they are too old based on `windowLength`
type AvgSlidingWindow struct {
	mux          sync.Mutex
	bucketSize   time.Duration
	windowLength time.Duration
	clock        Clock
	buckets      *lm.Map
	qty          uint
	sum          float64
}

type SlidingWindowOpts func(sw *AvgSlidingWindow)

func NewSlidingWindow(opts ...SlidingWindowOpts) *AvgSlidingWindow {
	sw := &AvgSlidingWindow{
		buckets: lm.New(),
	}
	for _, opt := range opts {
		opt(sw)
	}
	if sw.bucketSize == 0 {
		sw.bucketSize = time.Second
	}
	if sw.windowLength == 0 {
		sw.windowLength = 5 * time.Minute
	}
	if sw.clock == nil {
		sw.clock = NewDefaultClock()
	}
	return sw
}

func WithWindowLength(windowLength time.Duration) SlidingWindowOpts {
	return func(sw *AvgSlidingWindow) {
		sw.windowLength = windowLength
	}
}

func WithBucketSize(bucketSize time.Duration) SlidingWindowOpts {
	return func(sw *AvgSlidingWindow) {
		sw.bucketSize = bucketSize
	}
}

func WithClock(clock Clock) SlidingWindowOpts {
	return func(sw *AvgSlidingWindow) {
		sw.clock = clock
	}
}

func (sw *AvgSlidingWindow) inWindow(t time.Time) bool {
	now := sw.clock.Now().Round(sw.bucketSize)
	windowStart := now.Add(-sw.windowLength)
	return windowStart.Before(t) && !t.After(now)
}

// Add inserts a new data point into the window, with value `val` and the current time
func (sw *AvgSlidingWindow) Add(val float64) {
	t := sw.clock.Now()
	sw.AddWithTime(t, val)
}

// Incr is an alias to insert a data point with value float64(1) and the current time
func (sw *AvgSlidingWindow) Incr() {
	sw.Add(1)
}

// AddWithTime inserts a new data point into the window, with value `val` and time `t`
func (sw *AvgSlidingWindow) AddWithTime(t time.Time, val float64) {
	sw.advance()

	defer sw.mux.Unlock()
	sw.mux.Lock()

	key := t.Round(sw.bucketSize)
	if !sw.inWindow(key) {
		return
	}

	var b *bucket
	current, found := sw.buckets.Get(key)
	if !found {
		b = &bucket{}
	} else {
		b = current.(*bucket)
	}

	// update bucket
	bsum := b.sum
	b.qty += 1
	b.sum = bsum + val

	// update window
	wsum := sw.sum
	sw.qty += 1
	sw.sum = wsum - bsum + b.sum
	sw.buckets.Put(key, b)
}

// advance evicts old data points
func (sw *AvgSlidingWindow) advance() {
	defer sw.mux.Unlock()
	sw.mux.Lock()
	now := sw.clock.Now().Round(sw.bucketSize)
	windowStart := now.Add(-sw.windowLength)
	keys := sw.buckets.Keys()
	for _, k := range keys {
		if k.(time.Time).After(windowStart) {
			break
		}
		val, _ := sw.buckets.Get(k)
		b := val.(*bucket)
		sw.buckets.Remove(k)
		if sw.buckets.Size() > 0 {
			sw.qty -= b.qty
			sw.sum = sw.sum - b.sum
		} else {
			sw.qty = 0
			sw.sum = 0.0
		}
	}
}

// Avg retrieves the current average for the sliding window
func (sw *AvgSlidingWindow) Avg() float64 {
	sw.advance()
	if sw.qty == 0 {
		return 0
	}
	return sw.sum / float64(sw.qty)
}

// Sum retrieves the current sum for the sliding window
func (sw *AvgSlidingWindow) Sum() float64 {
	sw.advance()
	return sw.sum
}

// Count retrieves the data point count for the sliding window
func (sw *AvgSlidingWindow) Count() uint {
	sw.advance()
	return sw.qty
}
