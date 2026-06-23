package script

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsExpectedRevertPanic(t *testing.T) {
	tests := []struct {
		name string
		r    any
		want bool
	}{
		{"revision id 1 error", fmt.Errorf("revision id 1 cannot be reverted"), true},
		// Regression: a revision id not starting with "1" must still be recognised.
		// The previous literal "revision id 1" substring match dropped these and re-panicked.
		{"revision id 36 error", fmt.Errorf("revision id 36 cannot be reverted"), true},
		{"revision id 200 string", "revision id 200 cannot be reverted", true},
		{"revision id 9 error", errors.New("revision id 9 cannot be reverted"), true},
		{"uppercase message", errors.New("Revision ID 7 cannot be reverted"), true},
		{"unrelated string", "something else blew up", false},
		{"unrelated error", errors.New("out of gas"), false},
		{"revision id without number", errors.New("revision id cannot be reverted"), false},
		{"non-string non-error", 42, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isExpectedRevertPanic(tt.r))
		})
	}
}
