package status

// ringBuffer is a circular buffer that can be used to store a fixed number of
// elements. When the buffer is full, the oldest element is overwritten.
// This buffer implementation supports indexed access to elements, as well as
// access to the first and last elements.
type ringbuffer[T any] struct {
	contents []T
	start    int
	size     int
}

func newRingBuffer[T any](size int) *ringbuffer[T] {
	return &ringbuffer[T]{
		contents: make([]T, size),
	}
}

func (rb *ringbuffer[T]) Len() int {
	return rb.size
}

func (rb *ringbuffer[T]) Get(idx int) (T, bool) {
	if idx < 0 || idx >= rb.size {
		var zero T
		return zero, false
	}
	return rb.contents[(rb.start+idx)%len(rb.contents)], true
}

func (rb *ringbuffer[T]) Start() (T, bool) {
	if rb.size == 0 {
		var zero T
		return zero, false
	}
	return rb.contents[rb.start], true
}

func (rb *ringbuffer[T]) End() (T, bool) {
	if rb.size == 0 {
		var zero T
		return zero, false
	}
	return rb.contents[(rb.start+rb.size+len(rb.contents)-1)%len(rb.contents)], true
}

func (rb *ringbuffer[T]) Push(val T) {
	rb.contents[(rb.start+rb.size)%len(rb.contents)] = val
	if rb.size == len(rb.contents) {
		rb.start = (rb.start + 1) % len(rb.contents)
	} else {
		rb.size++
	}
}

func (rb *ringbuffer[T]) Pop() (T, bool) {
	end, ok := rb.End()
	if ok {
		rb.size--
	}
	return end, ok
}

func (rb *ringbuffer[T]) Reset() {
	rb.start = 0
	rb.size = 0
}
