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
	c *ring.Ring
}

// NewRing creates a new RingBuffer
func NewRing[T any](size int) *Ring[T] {
	b := Ring[T]{c: ring.New(size)}
	return &b
}

// Add adds a value to the RingBuffer, overwriting the oldest value if full
func (r *Ring[T]) Add(block T) {
	r.c = r.c.Move(-1)
	r.c.Value = block
}

// Peek returns the RingBuffer value
// If the value is unset, the empty T is returned
func (r *Ring[T]) Peek() T {
	v, ok := r.c.Value.(T)
	if !ok {
		var t T
		return t
	}
	return v
}

func (r *Ring[T]) Len() int {
	return r.c.Len()
}

// Reset resets the buffer, unsetting all values
func (r *Ring[T]) Reset() {
	s := NewRing[T](r.Len()) // create a new buffer with the same size
	r.c = s.c
}

// Pop removes the value from the buffer
func (r *Ring[T]) Pop() T {
	b := r.Peek()
	var t T
	r.c.Value = t
	r.c = r.c.Move(1)
	return b
}
