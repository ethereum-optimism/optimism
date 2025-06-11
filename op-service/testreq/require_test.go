package testreq

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTester struct {
	errStr  []string
	failNow bool
}

func (f *mockTester) Errorf(format string, args ...interface{}) {
	f.errStr = append(f.errStr, fmt.Sprintf(format, args...))
}

func (f *mockTester) FailNow() {
	f.failNow = true
}

func (f *mockTester) Helper() {
}

var _ TestingT = (*mockTester)(nil)

func TestEventually(t *testing.T) {
	t.Run("Eventually", func(t *testing.T) {
		m := &mockTester{}
		req := New(m)
		req.Eventually(func() bool {
			panic("testing the panic")
		}, time.Second, time.Millisecond*50, "abc")
		require.Len(t, m.errStr, 1)
		require.Contains(t, m.errStr[0], "testing the panic")
		require.Contains(t, m.errStr[0], "condition must not panic")
		require.Contains(t, m.errStr[0], "abc")
		require.True(t, m.failNow)
	})
	t.Run("Eventuallyf", func(t *testing.T) {
		m := &mockTester{}
		req := New(m)
		req.Eventuallyf(func() bool {
			panic("testing the panic")
		}, time.Second, time.Millisecond*50, "abc")
		require.Len(t, m.errStr, 1)
		require.Contains(t, m.errStr[0], "testing the panic")
		require.Contains(t, m.errStr[0], "condition must not panic")
		require.Contains(t, m.errStr[0], "abc")
		require.True(t, m.failNow)
	})
	t.Run("EventuallyWithT", func(t *testing.T) {
		m := &mockTester{}
		req := New(m)
		req.EventuallyWithT(func(t *assert.CollectT) {
			panic("testing the panic")
		}, time.Second, time.Millisecond*50, "abc")
		require.Len(t, m.errStr, 1)
		require.Contains(t, m.errStr[0], "testing the panic")
		require.Contains(t, m.errStr[0], "condition must not panic")
		require.Contains(t, m.errStr[0], "abc")
		require.True(t, m.failNow)
	})
	t.Run("EventuallyWithTf", func(t *testing.T) {
		m := &mockTester{}
		req := New(m)
		req.EventuallyWithTf(func(t *assert.CollectT) {
			panic("testing the panic")
		}, time.Second, time.Millisecond*50, "abc")
		require.Len(t, m.errStr, 1)
		require.Contains(t, m.errStr[0], "testing the panic")
		require.Contains(t, m.errStr[0], "condition must not panic")
		require.Contains(t, m.errStr[0], "abc")
		require.True(t, m.failNow)
	})
}
