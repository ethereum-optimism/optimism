package checks

import (
	"errors"
	"fmt"
	"testing"
)

func TestInteropTxRejected(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "op-geth filter rejection",
			err:  errors.New("transaction filtered out"),
			want: true,
		},
		{
			name: "op-interop-filter access entry parse failure",
			err:  errors.New("failed to parse access entry: bad data"),
			want: true,
		},
		{
			name: "op-reth fast-path failsafe",
			err:  errors.New("interop failsafe is active"),
			want: true,
		},
		{
			name: "op-interop-filter failsafe enabled",
			err:  errors.New("failsafe is enabled"),
			want: true,
		},
		{
			name: "rejection wrapped in another error",
			err:  fmt.Errorf("submit failed: %w", errors.New("transaction filtered out")),
			want: true,
		},
		{
			name: "unrelated rpc error",
			err:  errors.New("nonce too low"),
			want: false,
		},
		{
			name: "context deadline exceeded is not a rejection",
			err:  errors.New("context deadline exceeded"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := interopTxRejected(tt.err); got != tt.want {
				t.Errorf("interopTxRejected(%q) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
