package errutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTryAddRevertReason(t *testing.T) {
	t.Run("AddsReason", func(t *testing.T) {
		err := stubError{}
		result := TryAddRevertReason(err)
		require.Equal(t, "where's the: EVM revert reason: kaboom", result.Error())
		require.ErrorIs(t, result, err)
	})

	t.Run("ReturnOriginalWhenNoErrorDataMethod", func(t *testing.T) {
		err := errors.New("boom")
		result := TryAddRevertReason(err)
		require.Same(t, err, result)
	})

	t.Run("WrappedErrorRetainsIdentity", func(t *testing.T) {
		original := stubError{}
		wrapped := fmt.Errorf("estimate gas: %w", original)
		result := TryAddRevertReason(wrapped)
		require.Equal(t, "estimate gas: where's the: EVM revert reason: kaboom", result.Error())
		require.ErrorIs(t, result, wrapped)
		require.ErrorIs(t, result, original)
	})

	t.Run("NilError", func(t *testing.T) {
		require.NoError(t, TryAddRevertReason(nil))
	})
}

type stubError struct{}

func (s stubError) Error() string {
	return "where's the"
}

func (s stubError) ErrorData() interface{} {
	return "kaboom"
}
