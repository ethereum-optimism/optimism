package monitor

import (
	"container/ring"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BlockBuffer is a circular buffer of seen blocks.
// It can be used as a fix-sized stack of blocks to ensure
// a canonical and contiguous view of the block history.
type Buffer[T any] struct{ *ring.Ring }

type BlockBuffer = Buffer[eth.BlockInfo]

// NewBlockBuffer creates a new block buffer
func NewBuffer[T any](size int) *Buffer[T] {
	b := Buffer[T]{Ring: ring.New(size)}
	return &b
}

// Add adds a block to the buffer, removing the oldest block
func (r *Buffer[T]) Add(block T) {
	r.Ring = r.Ring.Move(-1)
	r.Value = block
}

// Peek returns the last added block to the buffer
// if the buffer is empty, it returns nil
// if the buffer is not empty, it returns the last added block
func (r *Buffer[T]) Peek() T {
	bi, ok := r.Value.(T)
	if !ok {
		var t T
		return t
	}
	return bi
}

// Reset resets the buffer to empty
func (r *Buffer[T]) Reset() {
	r = NewBuffer[T](r.Len()) // create a new buffer with the same size
}

func (r *Buffer[T]) Pop() T {
	b := r.Peek()
	r.Move(1)
	return b
}

func (r *Buffer[T]) Print() {
	fmt.Println("--------------------------------")
	r.Do(func(v any) {
		switch v := v.(type) {
		case *int:
			fmt.Printf("%v\n", *v)
		default:
			fmt.Printf("%v\n", v)
		}
	})
	p := r.Peek()
	n := r.Next()
	fmt.Printf("peek: %v\n", p)
	fmt.Printf("next: %v\n", n.Value)
	fmt.Println("--------------------------------")
}

func format(v any) string {
	switch v := v.(type) {
	case *int:
		return fmt.Sprintf("%v", *v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
