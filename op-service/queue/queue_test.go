package queue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	t.Run("enqueue/sequeue", func(t *testing.T) {
		q := Queue[int]{}
		q.Enqueue(1, 2, 3, 4)

		p, peekOk := q.Peek()
		require.True(t, peekOk)
		require.Equal(t, p, 1)

		d, dequeueOk := q.Dequeue()
		require.Equal(t, d, 1)
		require.True(t, dequeueOk)
		require.Equal(t, q.Len(), 3)
		p, peekOk = q.Peek()
		require.True(t, peekOk)
		require.Equal(t, p, 2)

		d, dequeueOk = q.Dequeue()
		require.Equal(t, d, 2)
		require.True(t, dequeueOk)
		require.Equal(t, q.Len(), 2)
		p, peekOk = q.Peek()
		require.True(t, peekOk)
		require.Equal(t, p, 3)

		d, dequeueOk = q.Dequeue()
		require.Equal(t, d, 3)
		require.True(t, dequeueOk)
		require.Equal(t, q.Len(), 1)
		p, peekOk = q.Peek()
		require.True(t, peekOk)
		require.Equal(t, p, 4)

		d, dequeueOk = q.Dequeue()
		require.Equal(t, d, 4)
		require.True(t, dequeueOk)
		require.Equal(t, q.Len(), 0)
		p, peekOk = q.Peek()
		require.False(t, peekOk)
		require.Equal(t, p, 0)

		d, dequeueOk = q.Dequeue()
		require.Equal(t, d, 0)
		require.False(t, dequeueOk)
		require.Equal(t, q.Len(), 0)
		p, peekOk = q.Peek()
		require.False(t, peekOk)
		require.Equal(t, p, 0)
		p, peekOk = q.Peek()
		require.False(t, peekOk)
		require.Equal(t, p, 0)
	})
	t.Run("enqueue/clear", func(t *testing.T) {
		q := Queue[int]{}
		q.Enqueue(5, 6, 7)

		q.Clear()
		require.Equal(t, q.Len(), 0)

		d, ok := q.Dequeue()
		require.Equal(t, d, 0)
		require.False(t, ok)
	})

	t.Run("prepend", func(t *testing.T) {
		var q, r Queue[int]
		q.Enqueue(5, 6, 7)
		r.Enqueue(8, 9)

		q.Prepend(r...)
		require.Equal(t, q.Len(), 5)

		d, ok := q.Dequeue()
		require.Equal(t, d, 8)
		require.True(t, ok)
	})
}
