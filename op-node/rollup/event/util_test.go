package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type FooEvent struct{}

func (ev FooEvent) String() string {
	return "foo"
}

type BarEvent struct{}

func (ev BarEvent) String() string {
	return "bar"
}

func TestIs(t *testing.T) {
	require.False(t, Is[TestEvent](FooEvent{}))
	require.False(t, Is[TestEvent](BarEvent{}))
	require.True(t, Is[FooEvent](FooEvent{}))
	require.True(t, Is[BarEvent](BarEvent{}))
}

func TestAny(t *testing.T) {
	require.False(t, Any()(FooEvent{}))
	require.False(t, Any(Is[BarEvent])(FooEvent{}))
	require.True(t, Any(Is[FooEvent])(FooEvent{}))
	require.False(t, Any(Is[TestEvent], Is[BarEvent])(FooEvent{}))
	require.True(t, Any(Is[TestEvent], Is[BarEvent], Is[FooEvent])(FooEvent{}))
	require.True(t, Any(Is[FooEvent], Is[BarEvent], Is[TestEvent])(FooEvent{}))
	require.True(t, Any(Is[FooEvent], Is[FooEvent], Is[FooEvent])(FooEvent{}))
	require.False(t, Any(Is[FooEvent], Is[FooEvent], Is[FooEvent])(BarEvent{}))
}
