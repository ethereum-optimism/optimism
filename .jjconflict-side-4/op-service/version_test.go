package op_service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatVersion(t *testing.T) {
	tests := []struct {
		version   string
		gitCommit string
		gitDate   string
		meta      string
		expected  string
	}{
		{
			version:   "v1.0.0",
			gitCommit: "c90a760cfaccefb60b942ffe4ccf4f9692587cec",
			gitDate:   "1698107786",
			meta:      "",
			expected:  "v1.0.0-c90a760c-1698107786",
		},
		{
			version:   "v1.0.0",
			gitCommit: "dev",
			gitDate:   "1698107786",
			meta:      "",
			expected:  "v1.0.0-dev-1698107786",
		},
		{
			version:   "v1.0.0",
			gitCommit: "",
			gitDate:   "1698107786",
			meta:      "",
			expected:  "v1.0.0-1698107786",
		},
		{
			version:   "v1.0.0",
			gitCommit: "dev",
			gitDate:   "",
			meta:      "",
			expected:  "v1.0.0-dev",
		},
		{
			version:   "v1.0.0",
			gitCommit: "",
			gitDate:   "",
			meta:      "rc.1",
			expected:  "v1.0.0-rc.1",
		},
		{
			version:   "v1.0.0",
			gitCommit: "",
			gitDate:   "",
			meta:      "",
			expected:  "v1.0.0",
		},
		{
			version:   "v1.0.0",
			gitCommit: "c90a760cfaccefb60b942ffe4ccf4f9692587cec",
			gitDate:   "1698107786",
			meta:      "beta",
			expected:  "v1.0.0-c90a760c-1698107786-beta",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.expected, func(t *testing.T) {
			actual := FormatVersion(test.version, test.gitCommit, test.gitDate, test.meta)
			require.Equal(t, test.expected, actual)
		})
	}
}
