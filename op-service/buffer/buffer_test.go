package buffer

import (
	"testing"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/stretchr/testify/require"
)

func TestBlockBuffer(t *testing.T) {
	var nilint *int
	one := opservice.Ptr(1)
	two := opservice.Ptr(2)
	three := opservice.Ptr(3)
	four := opservice.Ptr(4)

	r := NewRing[*int](3)

	// nil<-nil<-[nil]
	require.Equal(t, 3, r.Len())
	require.Equal(t, nilint, r.Peek())

	r.Add(one)
	// nil<-nil<-[ 1 ]
	require.Equal(t, one, r.Peek())

	r.Add(two)
	// nil<-1<-[ 2 ]
	require.Equal(t, two, r.Peek())

	r.Add(three)
	//  1<-2<-[ 3 ]
	require.Equal(t, three, r.Peek())

	r.Add(four)
	//  2<-3<-[ 4 ]
	require.Equal(t, four, r.Peek())

	p := r.Pop()
	// nil<-2<-[ 3 ]
	require.Equal(t, four, p)
	require.Equal(t, three, r.Peek())

	p = r.Pop()
	// nil<-nil<-[ 2 ]
	require.Equal(t, three, p)
	require.Equal(t, two, r.Peek())

	r.Reset()
	// nil<-nil<-[nil]
	require.Equal(t, 3, r.Len())
	require.Equal(t, nilint, r.Peek())

}
