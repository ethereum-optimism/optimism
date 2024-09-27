package queue

type Queue[T any] []T

func (q *Queue[T]) Enqueue(t ...T) {
	*q = append(*q, t...)
}
func (q *Queue[T]) Dequeue() (T, bool) {
	if len(*q) == 0 {
		var zeroValue T
		return zeroValue, false
	}
	t := (*q)[0]
	*q = (*q)[1:]
	return t, true
}
func (q *Queue[T]) Prepend(t ...T) {
	*q = append(t, *q...)
}
func (q *Queue[T]) Clear() {
	*q = (*q)[:0]
}
func (q *Queue[T]) Len() int {
	return len(*q)
}
