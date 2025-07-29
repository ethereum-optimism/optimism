package devtest

import (
	"fmt"
)

// RunParallel runs the test-function, for each of the given elements, in parallel.
// This awaits the result of all sub-tests to complete,
// by grouping everything into a regular sub-test.
func RunParallel[V any](t T, elems []V, fn func(t T, v V)) {
	t.Run("group", func(t T) {
		for _, elem := range elems {
			elem := elem
			t.Run(fmt.Sprintf("%v", elem), func(t T) {
				t.Parallel()
				fn(t, elem)
			})
		}
	})
}
