package queue

type Queue[T any] struct {
	slice []T
}

func (q *Queue[T]) Enqueue(t ...T) {
	q.slice = append(q.slice, t...)
}
func (q *Queue[T]) Dequeue() (T, bool) {
	if len(q.slice) == 0 {
		var zeroValue T
		return zeroValue, false
	}
	t := q.slice[0]
	q.slice = q.slice[1:]
	return t, true
}
func (q *Queue[T]) Prepend(t ...T) {
	q.slice = append(t, q.slice...)
}
func (q *Queue[T]) Clear() {
	q.slice = q.slice[:0]
}
func (q *Queue[T]) Len() int {
	return len(q.slice)
}
