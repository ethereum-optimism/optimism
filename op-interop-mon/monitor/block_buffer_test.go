package monitor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockBuffer(t *testing.T) {
	f := NewBuffer[*int](3)
	require.Equal(t, 3, f.Len())

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

}

func ptr(i int) *int {
	return &i
}
