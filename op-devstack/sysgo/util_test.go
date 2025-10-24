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
		{osValue: strPtr("a"), defaultValue: "b", expected: "a"},
		{osValue: strPtr("a"), defaultValue: "", expected: "a"},
		{osValue: strPtr(""), defaultValue: "b", expected: "b"},
		{osValue: strPtr(""), defaultValue: "b", expected: "b"},
		{osValue: strPtr(unsetVar), defaultValue: "b", expected: "b"},
		{osValue: strPtr(unsetVar), defaultValue: "", expected: ""},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			varName := fmt.Sprintf("%s_%d", envVarName, i)

			if test.osValue != nil {
				err := os.Setenv(varName, *test.osValue)
				require.NoErrorf(t, err, "Error setting env var %s", err)
			}

			require.Equal(t, GetEnvVarOrDefault(varName, test.defaultValue), test.expected)
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
		{osValue: strPtr("a"), defaultValue: "b", expected: "a"},
		{osValue: strPtr("a"), defaultValue: "", expected: "a"},
		{osValue: strPtr(""), defaultValue: "b", expected: "b"},
		{osValue: strPtr(""), defaultValue: "b", expected: "b"},
		{osValue: strPtr(unsetVar), defaultValue: "b", expected: "b"},
		{osValue: strPtr(unsetVar), defaultValue: "", expected: ""},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			varName := fmt.Sprintf("%s_%d", envVarName, i)

			if test.osValue != nil {
				err := os.Setenv(varName, *test.osValue)
				require.NoErrorf(t, err, "Error setting env var %s", err)
			}

			res := PropagateEnvVarOrDefault(varName, test.defaultValue)
			if (test.osValue == nil || *test.osValue == "") && test.defaultValue == "" {
				require.Equal(t, res, "")
			} else {
				require.Equal(t, res, fmt.Sprintf("%s=%s", varName, test.expected))
			}
		})
	}
}
