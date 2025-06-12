package buffer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockBuffer(t *testing.T) {
	var nilint *int
	one := ptr(1)
	two := ptr(2)
	three := ptr(3)
	four := ptr(4)

	f := NewRing[*int](3)

	// nil<-nil<-[nil]
	require.Equal(t, 3, f.Len())
	require.Equal(t, nilint, f.Peek())

	f.Add(one)
	// nil<-nil<-[ 1 ]
	require.Equal(t, one, f.Peek())

	f.Add(two)
	// nil<-1<-[ 2 ]
	require.Equal(t, two, f.Peek())

	f.Add(three)
	//  1<-2<-[ 3 ]
	require.Equal(t, three, f.Peek())

	f.Add(four)
	//  2<-3<-[ 4 ]
	require.Equal(t, four, f.Peek())

	p := f.Pop()
	// nil<-2<-[ 3 ]
	require.Equal(t, four, p)
	require.Equal(t, three, f.Peek())

	p = f.Pop()
	// nil<-nil<-[ 2 ]
	require.Equal(t, three, p)
	require.Equal(t, two, f.Peek())

	f.Reset()
	// nil<-nil<-[nil]
	require.Equal(t, 3, f.Len())
	require.Equal(t, nilint, f.Peek())
}

func ptr(i int) *int {
	return &i
}
