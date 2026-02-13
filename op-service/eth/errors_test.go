package eth

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/stretchr/testify/require"
)

func TestMaybeAsNotFoundErr(t *testing.T) {
	otherErr := errors.New("something else went wrong")

	tests := []struct {
		name               string
		err                error
		expectedIsNotFound bool
		expectedSameErr    bool
	}{
		{
			name:               "already ethereum.NotFound",
			err:                ethereum.NotFound,
			expectedIsNotFound: true,
			expectedSameErr:    true,
		},
		{
			name:               "block not found",
			err:                errors.New("block not found"),
			expectedIsNotFound: true,
		},
		{
			name:               "header not found",
			err:                errors.New("header not found"),
			expectedIsNotFound: true,
		},
		{
			name:               "Unknown block",
			err:                errors.New("Unknown block"),
			expectedIsNotFound: true,
		},
		{
			name:               "unknown block",
			err:                errors.New("unknown block"),
			expectedIsNotFound: true,
		},
		{
			name:               "BLOCK NOT FOUND",
			err:                errors.New("BLOCK NOT FOUND"),
			expectedIsNotFound: true,
		},
		{
			name:               "block not found in the middle",
			err:                fmt.Errorf("rpc error: %w for hash 0x123", errors.New("block not found")),
			expectedIsNotFound: true,
		},
		{
			name:               "header not found in context",
			err:                fmt.Errorf("failed to get header: %w", errors.New("header not found")),
			expectedIsNotFound: true,
		},
		{
			name:               "Unknown block with context",
			err:                fmt.Errorf("geth error: %w", errors.New("Unknown block")),
			expectedIsNotFound: true,
		},
		{
			name:               "unknown block with context",
			err:                fmt.Errorf("chain query failed: %w", errors.New("unknown block")),
			expectedIsNotFound: true,
		},
		{
			name:            "different error",
			err:             otherErr,
			expectedSameErr: true,
		},
		{
			name:            "similar but not matching text",
			err:             errors.New("block is not available"),
			expectedSameErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MaybeAsNotFoundErr(tc.err)

			if tc.expectedIsNotFound {
				require.ErrorIs(t, result, ethereum.NotFound)
				require.ErrorIs(t, result, tc.err, "original error should be preserved")
			} else {
				require.NotErrorIs(t, result, ethereum.NotFound)
			}
			if tc.expectedSameErr {
				require.Equal(t, tc.err, result)
			} else {
				require.NotEqual(t, tc.err, result)
			}
		})
	}
}
