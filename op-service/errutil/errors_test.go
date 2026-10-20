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
		require.Contains(t, result.Error(), "kaboom")
	})

	t.Run("ReturnOriginalWhenNoErrorDataMethod", func(t *testing.T) {
		err := errors.New("boom")
		result := TryAddRevertReason(err)
		require.Same(t, err, result)
	})

	t.Run("NilDataPreservesOriginal", func(t *testing.T) {
		err := &nilDataError{}
		require.Same(t, err, TryAddRevertReason(err))
	})

	t.Run("WrappedNilDataPreservesOriginal", func(t *testing.T) {
		err := fmt.Errorf("RPC request failed: %w", &nilDataError{})
		require.Same(t, err, TryAddRevertReason(err))
	})
}

type nilDataError struct{}

func (e *nilDataError) Error() string {
	return "reverted without data"
}

func (e *nilDataError) ErrorData() interface{} {
	return nil
}

type stubError struct{}

func (s stubError) Error() string {
	return "where's the"
}

func (s stubError) ErrorData() interface{} {
	return "kaboom"
}
