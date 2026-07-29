package sysgo

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const unsetVar = "unset"

func strPtr(s string) *string {
	if s == unsetVar {
		return nil
	}
	return &s
}

func TestGetEnvVarOrDefault(t *testing.T) {
	const envVarName = "TestGetEnvVarOrDefaultEnvVarName"
	tests := []struct {
		name         string
		osValue      *string
		defaultValue string
		expected     string
	}{
		{name: "Use OS value, no default", osValue: strPtr("a"), defaultValue: "", expected: "a"},
		{name: "Use OS value with default", osValue: strPtr("a"), defaultValue: "b", expected: "a"},
		{name: "Use empty OS Value with default", osValue: strPtr(""), defaultValue: "b", expected: ""},
		{name: "Use empty OS Value, no default", osValue: strPtr(""), defaultValue: "", expected: ""},
		{name: "Use default", osValue: strPtr(unsetVar), defaultValue: "b", expected: "b"},
		{name: "Use empty default", osValue: strPtr(unsetVar), defaultValue: "", expected: ""},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			varName := fmt.Sprintf("%s_%d", envVarName, i)

			if test.osValue != nil {
				err := os.Setenv(varName, *test.osValue)
				require.NoErrorf(t, err, "Error setting env var %s", err)
			}

			require.Equal(t, getEnvVarOrDefault(varName, test.defaultValue), test.expected)
		})
	}

}

func TestPropagateEnvVarOrDefault(t *testing.T) {
	const envVarName = "TestPropagateEnvVarOrDefaultEnvVarName"
	tests := []struct {
		name         string
		osValue      *string
		defaultValue string
		expected     string
	}{
		{name: "Use OS value, no default", osValue: strPtr("a"), defaultValue: "", expected: "a"},
		{name: "Use OS value with default", osValue: strPtr("a"), defaultValue: "b", expected: "a"},
		{name: "Use empty OS Value with default", osValue: strPtr(""), defaultValue: "b", expected: ""},
		{name: "Use empty OS Value with empty default", osValue: strPtr(""), defaultValue: "", expected: ""},
		{name: "Use default", osValue: strPtr(unsetVar), defaultValue: "b", expected: "b"},
		{name: "Use empty default", osValue: strPtr(unsetVar), defaultValue: "", expected: ""},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			varName := fmt.Sprintf("%s_%d", envVarName, i)

			if test.osValue != nil {
				err := os.Setenv(varName, *test.osValue)
				require.NoErrorf(t, err, "Error setting env var %s", err)
			}

			res := propagateEnvVarOrDefault(varName, test.defaultValue)
			if (test.osValue != nil && *test.osValue == "") || (test.osValue == nil && test.defaultValue == "") {
				require.Equal(t, res, "")
			} else {
				require.Equal(t, res, fmt.Sprintf("%s=%s", varName, test.expected))
			}
		})
	}
}

func TestParseBoundAddressLog(t *testing.T) {
	t.Run("rollup boost RPC", func(t *testing.T) {
		actual, ok := parseBoundAddressLog(
			"RPC server listening on 127.0.0.1:43123",
			"RPC server listening on ",
			"http",
		)
		require.True(t, ok)
		require.Equal(t, "http://127.0.0.1:43123", actual)
	})

	t.Run("unrelated log", func(t *testing.T) {
		_, ok := parseBoundAddressLog(
			"server starting",
			"RPC server listening on ",
			"http",
		)
		require.False(t, ok)
	})

	t.Run("zero port", func(t *testing.T) {
		_, ok := parseBoundAddressLog(
			"RPC server listening on 127.0.0.1:0",
			"RPC server listening on ",
			"http",
		)
		require.False(t, ok)
	})
}
