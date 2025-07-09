package event

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestEvent struct {
}

func (ev TestEvent) String() string {
	return "X"
}

func TestDeriverMux_OnEvent(t *testing.T) {
	result := ""
	a := DeriverFunc(func(ctx context.Context, ev Event) bool {
		result += fmt.Sprintf("A:%s\n", ev)
		return true
	})
	b := DeriverFunc(func(ctx context.Context, ev Event) bool {
		result += fmt.Sprintf("B:%s\n", ev)
		return true
	})
	c := DeriverFunc(func(ctx context.Context, ev Event) bool {
		result += fmt.Sprintf("C:%s\n", ev)
		return true
	})

	x := DeriverMux{}
	x.OnEvent(context.Background(), TestEvent{})
	require.Equal(t, "", result)

	x = DeriverMux{a}
	x.OnEvent(context.Background(), TestEvent{})
	require.Equal(t, "A:X\n", result)

	result = ""
	x = DeriverMux{a, a}
	x.OnEvent(context.Background(), TestEvent{})
	require.Equal(t, "A:X\nA:X\n", result)

	result = ""
	x = DeriverMux{a, b}
	x.OnEvent(context.Background(), TestEvent{})
	require.Equal(t, "A:X\nB:X\n", result)

	result = ""
	x = DeriverMux{a, b, c}
	x.OnEvent(context.Background(), TestEvent{})
	require.Equal(t, "A:X\nB:X\nC:X\n", result)
}
