package rollup

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestEvent struct{}

func (ev TestEvent) String() string {
	return "X"
}

func TestSynchronousDerivers_OnEvent(t *testing.T) {
	result := ""
	a := DeriverFunc(func(ev Event) {
		result += fmt.Sprintf("A:%s\n", ev)
	})
	b := DeriverFunc(func(ev Event) {
		result += fmt.Sprintf("B:%s\n", ev)
	})
	c := DeriverFunc(func(ev Event) {
		result += fmt.Sprintf("C:%s\n", ev)
	})

	x := SynchronousDerivers{}
	x.OnEvent(TestEvent{})
	require.Equal(t, "", result)

	x = SynchronousDerivers{a}
	x.OnEvent(TestEvent{})
	require.Equal(t, "A:X\n", result)

	result = ""
	x = SynchronousDerivers{a, a}
	x.OnEvent(TestEvent{})
	require.Equal(t, "A:X\nA:X\n", result)

	result = ""
	x = SynchronousDerivers{a, b}
	x.OnEvent(TestEvent{})
	require.Equal(t, "A:X\nB:X\n", result)

	result = ""
	x = SynchronousDerivers{a, b, c}
	x.OnEvent(TestEvent{})
	require.Equal(t, "A:X\nB:X\nC:X\n", result)
}
