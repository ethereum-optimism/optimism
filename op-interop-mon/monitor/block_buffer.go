package monitor

import (
	"container/ring"
	"fmt"
)

// A buffer element may always have a nil default value
type Buffer[T any] struct {
	*ring.Ring
}

// NewBlockBuffer creates a new block buffer
func NewBuffer[T any](size int) *Buffer[T] {
	b := Buffer[T]{Ring: ring.New(size)}
	return &b
}

// Add adds a value to the buffer, removing the oldest value
func (r *Buffer[T]) Add(block T) {
	r.Ring = r.Ring.Move(-1)
	r.Value = block
}

// Peek returns the last added value to the buffer
// if the buffer is empty, it returns nil
// if the buffer is not empty, it returns the last added value
func (r *Buffer[T]) Peek() T {
	v, ok := r.Value.(T)
	if !ok {
		var t T
		return t
	}
	return v
}

// Reset resets the buffer to empty
func (r *Buffer[T]) Reset() {
	s := NewBuffer[T](r.Len()) // create a new buffer with the same size
	r.Ring = s.Ring
}

// Pop removes the last added value from the buffer
func (r *Buffer[T]) Pop() T {
	b := r.Peek()
	var t T
	r.Value = t
	r.Ring = r.Ring.Move(1)
	return b
}

func (r *Buffer[T]) Print() {
	fmt.Println("--------------------------------")
	r.Do(func(v any) {
		fmt.Printf("%v\n", format(v))
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
		if v == nil {
			return "nil"
		}
		return fmt.Sprintf("%v", *v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
