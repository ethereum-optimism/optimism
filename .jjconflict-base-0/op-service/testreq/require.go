package testreq

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestingT interface {
	require.TestingT
	Helper()
}

// Assertions extends and improves require.Assertions
type Assertions struct {
	require.Assertions
	t TestingT
}

func New(t TestingT) *Assertions {
	return &Assertions{
		Assertions: *require.New(t),
		t:          t,
	}
}

// wrapCondition protects against panics in the test-condition, and then turns them into test fails.
// Used by Assertions.Eventually and Assertions.Eventuallyf.
func wrapCondition(fn func() bool, panicV *any) func() bool {
	// condition runs on its own go-routine, and would crash the whole test-suite if a panic is not caught
	return func() (met bool) {
		defer func() {
			if v := recover(); v != nil {
				*panicV = v
				// consider it met; we want the condition to exit, not to retry
				met = true
			}
		}()
		return fn()
	}
}

// wrapConditionCollectT protects against panics in the test-condition, and then turns them into test fails.
// Used by Assertions.EventuallyWithT and Assertions.EventuallyWithTf.
func wrapConditionCollectT(fn func(collect *assert.CollectT), panicV *any) func(collect *assert.CollectT) {
	// condition runs on its own go-routine, and would crash the whole test-suite if a panic is not caught
	return func(collect *assert.CollectT) {
		defer func() {
			if v := recover(); v != nil {
				*panicV = v
				// collect no error; we want the condition to exit, not to retry
			}
		}()
		fn(collect)
	}
}

// Eventually implements require.Eventually, with panic-protection for the condition.
func (a *Assertions) Eventually(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	a.t.Helper()
	var panicV any
	a.Assertions.Eventually(wrapCondition(condition, &panicV), waitFor, tick, msgAndArgs...)
	a.Assertions.Nil(panicV, "condition must not panic, condition: "+fmt.Sprintln(msgAndArgs...))
}

// Eventuallyf implements require.Eventuallyf, with panic-protection for the condition.
func (a *Assertions) Eventuallyf(condition func() bool, waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) {
	a.t.Helper()
	var panicV any
	a.Assertions.Eventuallyf(wrapCondition(condition, &panicV), waitFor, tick, msg, args...)
	a.Assertions.Nil(panicV, "condition must not panic, condition: "+fmt.Sprintf(msg, args...))
}

// EventuallyWithT implements require.EventuallyWithT, with panic-protection for the condition.
func (a *Assertions) EventuallyWithT(condition func(collect *assert.CollectT), waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	a.t.Helper()
	var panicV any
	a.Assertions.EventuallyWithT(wrapConditionCollectT(condition, &panicV), waitFor, tick, msgAndArgs...)
	a.Assertions.Nil(panicV, "condition must not panic, condition: "+fmt.Sprintln(msgAndArgs...))
}

// EventuallyWithTf implements require.EventuallyWithTf, with panic-protection for the condition.
func (a *Assertions) EventuallyWithTf(condition func(collect *assert.CollectT), waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) {
	a.t.Helper()
	var panicV any
	a.Assertions.EventuallyWithTf(wrapConditionCollectT(condition, &panicV), waitFor, tick, msg, args...)
	a.Assertions.Nil(panicV, "condition must not panic, condition: "+fmt.Sprintf(msg, args...))
}
