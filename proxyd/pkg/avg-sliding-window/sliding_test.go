package avg_sliding_window

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSlidingWindow_AddWithTime_Single(t *testing.T) {
	now := ts("2023-04-21 15:04:05")
	clock := NewAdjustableClock(now)

	sw := NewSlidingWindow(
		WithWindowLength(10*time.Second),
		WithBucketSize(time.Second),
		WithClock(clock))
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 5)
	require.Equal(t, 5.0, sw.Avg())
	require.Equal(t, 5.0, sw.Sum())
	require.Equal(t, 1, int(sw.Count()))
	require.Equal(t, 1, sw.buckets.Size())
	require.Equal(t, 1, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 5.0, sw.buckets.Values()[0].(*bucket).sum)
}

func TestSlidingWindow_AddWithTime_TwoValues_SameBucket(t *testing.T) {
	now := ts("2023-04-21 15:04:05")
	clock := NewAdjustableClock(now)

	sw := NewSlidingWindow(
		WithWindowLength(10*time.Second),
		WithBucketSize(time.Second),
		WithClock(clock))
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 5)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 5)
	require.Equal(t, 5.0, sw.Avg())
	require.Equal(t, 10.0, sw.Sum())
	require.Equal(t, 2, int(sw.Count()))
	require.Equal(t, 1, sw.buckets.Size())
	require.Equal(t, 2, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 10.0, sw.buckets.Values()[0].(*bucket).sum)
}

func TestSlidingWindow_AddWithTime_ThreeValues_SameBucket(t *testing.T) {
	now := ts("2023-04-21 15:04:05")
	clock := NewAdjustableClock(now)

	sw := NewSlidingWindow(
		WithWindowLength(10*time.Second),
		WithBucketSize(time.Second),
		WithClock(clock))
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 4)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 5)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 6)
	require.Equal(t, 5.0, sw.Avg())
	require.Equal(t, 15.0, sw.Sum())
	require.Equal(t, 3, int(sw.Count()))
	require.Equal(t, 1, sw.buckets.Size())
	require.Equal(t, 15.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 3, int(sw.buckets.Values()[0].(*bucket).qty))
}

func TestSlidingWindow_AddWithTime_ThreeValues_ThreeBuckets(t *testing.T) {
	now := ts("2023-04-21 15:04:05")
	clock := NewAdjustableClock(now)

	sw := NewSlidingWindow(
		WithWindowLength(10*time.Second),
		WithBucketSize(time.Second),
		WithClock(clock))
	sw.AddWithTime(ts("2023-04-21 15:04:01"), 4)
	sw.AddWithTime(ts("2023-04-21 15:04:02"), 5)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 6)
	require.Equal(t, 5.0, sw.Avg())
	require.Equal(t, 15.0, sw.Sum())
	require.Equal(t, 3, int(sw.Count()))
	require.Equal(t, 3, sw.buckets.Size())
	require.Equal(t, 1, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 4.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[1].(*bucket).qty))
	require.Equal(t, 5.0, sw.buckets.Values()[1].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[2].(*bucket).qty))
	require.Equal(t, 6.0, sw.buckets.Values()[2].(*bucket).sum)
}

func TestSlidingWindow_AddWithTime_OutWindow(t *testing.T) {
	now := ts("2023-04-21 15:04:05")
	clock := NewAdjustableClock(now)

	sw := NewSlidingWindow(
		WithWindowLength(10*time.Second),
		WithBucketSize(time.Second),
		WithClock(clock))
	sw.AddWithTime(ts("2023-04-21 15:03:55"), 1000)
	sw.AddWithTime(ts("2023-04-21 15:04:01"), 4)
	sw.AddWithTime(ts("2023-04-21 15:04:02"), 5)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 6)
	require.Equal(t, 5.0, sw.Avg())
	require.Equal(t, 15.0, sw.Sum())
	require.Equal(t, 3, int(sw.Count()))
	require.Equal(t, 3, sw.buckets.Size())
	require.Equal(t, 1, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 4.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[1].(*bucket).qty))
	require.Equal(t, 5.0, sw.buckets.Values()[1].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[2].(*bucket).qty))
	require.Equal(t, 6.0, sw.buckets.Values()[2].(*bucket).sum)
}

func TestSlidingWindow_AdvanceClock(t *testing.T) {
	now := ts("2023-04-21 15:04:05")
	clock := NewAdjustableClock(now)

	sw := NewSlidingWindow(
		WithWindowLength(10*time.Second),
		WithBucketSize(time.Second),
		WithClock(clock))
	sw.AddWithTime(ts("2023-04-21 15:04:01"), 4)
	sw.AddWithTime(ts("2023-04-21 15:04:02"), 5)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 6)
	require.Equal(t, 5.0, sw.Avg())
	require.Equal(t, 15.0, sw.Sum())
	require.Equal(t, 3, int(sw.Count()))
	require.Equal(t, 3, sw.buckets.Size())

	require.Equal(t, 1, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 4.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[1].(*bucket).qty))
	require.Equal(t, 5.0, sw.buckets.Values()[1].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[2].(*bucket).qty))
	require.Equal(t, 6.0, sw.buckets.Values()[2].(*bucket).sum)

	// up until 15:04:05 we had 3 buckets
	// let's advance the clock to 15:04:11 and the first data point should be evicted
	clock.Set(ts("2023-04-21 15:04:11"))
	require.Equal(t, 5.5, sw.Avg())
	require.Equal(t, 11.0, sw.Sum())
	require.Equal(t, 2, int(sw.Count()))
	require.Equal(t, 2, sw.buckets.Size())
	require.Equal(t, 1, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 5.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[1].(*bucket).qty))
	require.Equal(t, 6.0, sw.buckets.Values()[1].(*bucket).sum)

	// let's advance the clock to 15:04:12 and another data point should be evicted
	clock.Set(ts("2023-04-21 15:04:12"))
	require.Equal(t, 6.0, sw.Avg())
	require.Equal(t, 6.0, sw.Sum())
	require.Equal(t, 1, int(sw.Count()))
	require.Equal(t, 1, sw.buckets.Size())
	require.Equal(t, 1, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 6.0, sw.buckets.Values()[0].(*bucket).sum)

	// let's advance the clock to 15:04:25 and all data point should be evicted
	clock.Set(ts("2023-04-21 15:04:25"))
	require.Equal(t, 0.0, sw.Avg())
	require.Equal(t, 0.0, sw.Sum())
	require.Equal(t, 0, int(sw.Count()))
	require.Equal(t, 0, sw.buckets.Size())
}

func TestSlidingWindow_MultipleValPerBucket(t *testing.T) {
	now := ts("2023-04-21 15:04:05")
	clock := NewAdjustableClock(now)

	sw := NewSlidingWindow(
		WithWindowLength(10*time.Second),
		WithBucketSize(time.Second),
		WithClock(clock))
	sw.AddWithTime(ts("2023-04-21 15:04:01"), 4)
	sw.AddWithTime(ts("2023-04-21 15:04:01"), 12)
	sw.AddWithTime(ts("2023-04-21 15:04:02"), 5)
	sw.AddWithTime(ts("2023-04-21 15:04:02"), 15)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 6)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 3)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 1)
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 3)
	require.Equal(t, 6.125, sw.Avg())
	require.Equal(t, 49.0, sw.Sum())
	require.Equal(t, 8, int(sw.Count()))
	require.Equal(t, 3, sw.buckets.Size())
	require.Equal(t, 2, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 16.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 2, int(sw.buckets.Values()[1].(*bucket).qty))
	require.Equal(t, 20.0, sw.buckets.Values()[1].(*bucket).sum)
	require.Equal(t, 4, int(sw.buckets.Values()[2].(*bucket).qty))
	require.Equal(t, 13.0, sw.buckets.Values()[2].(*bucket).sum)

	// up until 15:04:05 we had 3 buckets
	// let's advance the clock to 15:04:11 and the first data point should be evicted
	clock.Set(ts("2023-04-21 15:04:11"))
	require.Equal(t, 5.5, sw.Avg())
	require.Equal(t, 33.0, sw.Sum())
	require.Equal(t, 6, int(sw.Count()))
	require.Equal(t, 2, sw.buckets.Size())
	require.Equal(t, 2, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 20.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 4, int(sw.buckets.Values()[1].(*bucket).qty))
	require.Equal(t, 13.0, sw.buckets.Values()[1].(*bucket).sum)

	// let's advance the clock to 15:04:12 and another data point should be evicted
	clock.Set(ts("2023-04-21 15:04:12"))
	require.Equal(t, 3.25, sw.Avg())
	require.Equal(t, 13.0, sw.Sum())
	require.Equal(t, 4, int(sw.Count()))
	require.Equal(t, 1, sw.buckets.Size())
	require.Equal(t, 4, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 13.0, sw.buckets.Values()[0].(*bucket).sum)

	// let's advance the clock to 15:04:25 and all data point should be evicted
	clock.Set(ts("2023-04-21 15:04:25"))
	require.Equal(t, 0.0, sw.Avg())
	require.Equal(t, 0, sw.buckets.Size())
}

func TestSlidingWindow_CustomBucket(t *testing.T) {
	now := ts("2023-04-21 15:04:05")
	clock := NewAdjustableClock(now)

	sw := NewSlidingWindow(
		WithWindowLength(30*time.Second),
		WithBucketSize(10*time.Second),
		WithClock(clock))
	sw.AddWithTime(ts("2023-04-21 15:03:49"), 5)  // key: 03:50, sum: 5.0
	sw.AddWithTime(ts("2023-04-21 15:04:02"), 15) // key: 04:00
	sw.AddWithTime(ts("2023-04-21 15:04:03"), 5)  // key: 04:00
	sw.AddWithTime(ts("2023-04-21 15:04:04"), 1)  // key: 04:00, sum: 21.0
	sw.AddWithTime(ts("2023-04-21 15:04:05"), 3)  // key: 04:10, sum: 3.0
	require.Equal(t, 5.8, sw.Avg())
	require.Equal(t, 29.0, sw.Sum())
	require.Equal(t, 5, int(sw.Count()))
	require.Equal(t, 3, sw.buckets.Size())
	require.Equal(t, 5.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 21.0, sw.buckets.Values()[1].(*bucket).sum)
	require.Equal(t, 3, int(sw.buckets.Values()[1].(*bucket).qty))
	require.Equal(t, 3.0, sw.buckets.Values()[2].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[2].(*bucket).qty))

	// up until 15:04:05 we had 3 buckets
	// let's advance the clock to 15:04:21 and the first data point should be evicted
	clock.Set(ts("2023-04-21 15:04:21"))
	require.Equal(t, 6.0, sw.Avg())
	require.Equal(t, 24.0, sw.Sum())
	require.Equal(t, 4, int(sw.Count()))
	require.Equal(t, 2, sw.buckets.Size())
	require.Equal(t, 21.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 3, int(sw.buckets.Values()[0].(*bucket).qty))
	require.Equal(t, 3.0, sw.buckets.Values()[1].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[1].(*bucket).qty))

	// let's advance the clock to 15:04:32 and another data point should be evicted
	clock.Set(ts("2023-04-21 15:04:32"))
	require.Equal(t, 3.0, sw.Avg())
	require.Equal(t, 3.0, sw.Sum())
	require.Equal(t, 1, sw.buckets.Size())
	require.Equal(t, 1, int(sw.Count()))
	require.Equal(t, 3.0, sw.buckets.Values()[0].(*bucket).sum)
	require.Equal(t, 1, int(sw.buckets.Values()[0].(*bucket).qty))

	// let's advance the clock to 15:04:46 and all data point should be evicted
	clock.Set(ts("2023-04-21 15:04:46"))
	require.Equal(t, 0.0, sw.Avg())
	require.Equal(t, 0.0, sw.Sum())
	require.Equal(t, 0, int(sw.Count()))
	require.Equal(t, 0, sw.buckets.Size())
}

// ts is a convenient method that must parse a time.Time from a string in format `"2006-01-02 15:04:05"`
func ts(s string) time.Time {
	t, err := time.Parse(time.DateTime, s)
	if err != nil {
		panic(err)
	}
	return t
}
