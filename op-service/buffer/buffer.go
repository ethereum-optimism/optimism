package buffer

import (
	"container/ring"
)

// A Ring is a generic [container/ring.Ring]
// with convenient methods for adding and removing values.
// The generic type T should be a pointer type or interface
// because the default value of T will be returned by Pop and Peek if the value
// is unset.
type Ring[T any] struct {
	*ring.Ring
}

// NewBlockBuffer creates a new RingBuffer
func NewRing[T any](size int) *Ring[T] {
	b := Ring[T]{Ring: ring.New(size)}
	return &b
}

// Add adds a value to the RingBuffer, removing the oldest value
func (r *Ring[T]) Add(block T) {
	r.Ring = r.Ring.Move(-1)
	r.Value = block
}

// Peek returns the RingBuffer value
// If the value is unset, the empty T is returned
func (r *Ring[T]) Peek() T {
	v, ok := r.Value.(T)
	if !ok {
		var t T
		return t
	}
	return v
}

// Reset resets the buffer, unsetting all values
func (r *Ring[T]) Reset() {
	s := NewRing[T](r.Len()) // create a new buffer with the same size
	r.Ring = s.Ring
}

// Pop removes the value from the buffer
func (r *Ring[T]) Pop() T {
	b := r.Peek()
	var t T
	r.Value = t
	r.Ring = r.Ring.Move(1)
	return b
}
