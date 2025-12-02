package errutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTryAddRevertReason(t *testing.T) {
	t.Run("AddsReason", func(t *testing.T) {
		err := stubError{}
		result := TryAddRevertReason(err)
		require.Contains(t, result.Error(), "kaboom")
	})

	t.Run("ReturnOriginalWhenNoErrorDataMethod", func(t *testing.T) {
		err := errors.New("boom")
		result := TryAddRevertReason(err)
		require.Same(t, err, result)
	})
}

type stubError struct{}

func (s stubError) Error() string {
	return "where's the"
}

func (s stubError) ErrorData() interface{} {
	return "kaboom"
}
