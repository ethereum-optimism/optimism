package versions

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStateVersion(t *testing.T) {
	for _, version := range StateVersionTypes {
		t.Run(version.String(), func(t *testing.T) {
			result, err := ParseStateVersion(version.String())
			require.NoError(t, err)
			require.Equal(t, version, result)
		})
	}
}

func TestIsSupported(t *testing.T) {
	type TestCase struct {
		name     string
		input    int
		expected bool
	}
	cases := []TestCase{}

	maxSupportedValue := 0
	for _, ver := range StateVersionTypes {
		versionValue := int(ver)
		if versionValue > maxSupportedValue {
			maxSupportedValue = versionValue
		}
		if IsSupportedMultiThreaded64(ver) {
			cases = append(cases, TestCase{name: "Supported version " + ver.String(), input: versionValue, expected: true})
		} else {
			cases = append(cases, TestCase{name: "Unsupported version " + ver.String(), input: versionValue, expected: false})
		}
	}

	cases = append(cases,
		TestCase{name: "Min unsupported version", input: maxSupportedValue + 1, expected: false},
		TestCase{name: "Min unsupported version + 1", input: maxSupportedValue + 2, expected: false},
		TestCase{name: "Unsupported version overflows uint8", input: 256, expected: false},
		TestCase{name: "Unsupported version overflows uint8", input: 257, expected: false},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsSupported(tc.input)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	require.True(t, IsSupported(int(GetCurrentVersion())))
}

func TestGetExperimentalVersion(t *testing.T) {
	require.True(t, IsSupported(int(GetExperimentalVersion())))

	require.GreaterOrEqual(t, GetExperimentalVersion(), GetCurrentVersion())

	// Experimental version should be equal to the latest version
	expectedValue := slices.Max(StateVersionTypes)
	require.Equal(t, expectedValue, GetExperimentalVersion())
}

func TestStateVersionTypes(t *testing.T) {
	// Versions should be in ascending order
	lastVersion := StateVersion(0)
	for i, version := range StateVersionTypes {
		if i == 0 {
			require.GreaterOrEqual(t, version, lastVersion)
		} else {
			require.Greater(t, version, lastVersion)
		}
		lastVersion = version
	}
}
