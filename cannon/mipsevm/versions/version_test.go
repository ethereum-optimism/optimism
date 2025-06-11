package versions

import (
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
