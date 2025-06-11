package monitor

import (
	"container/ring"
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
