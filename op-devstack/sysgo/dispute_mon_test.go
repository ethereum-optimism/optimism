package sysgo

import (
	"log/slog"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsURLFromCapturedAddr(t *testing.T) {
	addr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 3456,
	}
	tests := []struct {
		name        string
		value       slog.Value
		captured    bool
		expected    string
		errorString string
	}{
		{
			name:     "valid address",
			value:    slog.AnyValue(addr),
			captured: true,
			expected: "http://127.0.0.1:3456",
		},
		{
			name:        "missing address",
			errorString: "not found",
		},
		{
			name:        "wrong attribute type",
			value:       slog.StringValue("127.0.0.1:3456"),
			captured:    true,
			errorString: "net.Addr",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := metricsURLFromCapturedAddr(test.value, test.captured)
			if test.errorString != "" {
				require.ErrorContains(t, err, test.errorString)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}
}
