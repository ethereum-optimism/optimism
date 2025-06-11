package monitor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockBuffer(t *testing.T) {
	f := NewBuffer[*int](3)
	require.Equal(t, 3, f.Len())

	var nilint *int
	require.Equal(t, nilint, f.Peek())

	f.Print()

	f.Add(ptr(1))
	f.Print()
	require.Equal(t, ptr(1), f.Peek())

	f.Add(ptr(2))
	f.Print()
	require.Equal(t, ptr(2), f.Peek())

	f.Add(ptr(3))
	f.Print()
	require.Equal(t, ptr(3), f.Peek())

	f.Add(ptr(4))
	f.Print()
	require.Equal(t, ptr(4), f.Peek())

	p := f.Pop()
	require.Equal(t, ptr(4), p)
	f.Print()

	p = f.Pop()
	require.Equal(t, ptr(3), p)
	f.Print()

	f.Reset()
	f.Print()
	require.Equal(t, 3, f.Len())
	require.Equal(t, nilint, f.Peek())
}

func ptr(i int) *int {
	return &i
}
