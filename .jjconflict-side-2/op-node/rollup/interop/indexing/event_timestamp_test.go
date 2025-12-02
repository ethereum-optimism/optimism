package indexing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewEventTimestamp(t *testing.T) {
	ttl := 5 * time.Second
	et := newEventTimestamp[string](ttl)

	require.Equal(t, ttl, et.ttl)
	require.Equal(t, "", et.lastValue) // zero value for string
	require.True(t, et.lastTime.IsZero())
}

func TestEventTimestamp_Update(t *testing.T) {
	ttl := 100 * time.Millisecond
	et := newEventTimestamp[string](ttl)

	// First update should always return true
	require.True(t, et.Update("foo"))
	require.Equal(t, "foo", et.lastValue)
	require.False(t, et.lastTime.IsZero())

	// Same value within TTL should return false
	require.False(t, et.Update("foo"))
	require.Equal(t, "foo", et.lastValue)

	// Different value should return true and update both value and time
	oldTime := et.lastTime
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	require.True(t, et.Update("bar"))
	require.Equal(t, "bar", et.lastValue)
	require.True(t, et.lastTime.After(oldTime))

	// Wait for TTL to expire
	time.Sleep(ttl + 10*time.Millisecond)
	oldTime = et.lastTime
	require.True(t, et.Update("baz"))
	require.Equal(t, "baz", et.lastValue)
	require.True(t, et.lastTime.After(oldTime))
}

func TestEventTimestamp_ZeroTTL(t *testing.T) {
	// Test with zero TTL - should always return true even in short succession
	et := newEventTimestamp[string](0)
	require.True(t, et.Update("test"))
	require.True(t, et.Update("test"))
	require.True(t, et.Update("test"))
}
