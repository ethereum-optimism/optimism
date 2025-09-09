package errutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum"
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

func TestIsEthereumNotFound(t *testing.T) {
	var tests = []struct {
		name       string
		err        error
		isNotFound bool
	}{
		{"ActualError", ethereum.NotFound, true},
		{"WrappedActualError", fmt.Errorf("foo: %w", ethereum.NotFound), true},
		{"SerializedError", errors.New(ethereum.NotFound.Error()), true},
		{"SerializedWrappedError", fmt.Errorf("request failed: %s", ethereum.NotFound.Error()), true},
		{"Generic error", errors.New("boom"), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := IsEthereumNotFound(test.err)
			require.Equal(t, test.isNotFound, actual)
		})
	}
}
