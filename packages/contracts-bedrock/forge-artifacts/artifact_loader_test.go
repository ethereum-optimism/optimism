package forge_artifacts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadABIs(t *testing.T) {
	tests := []struct {
		contract string
		method   func() (*Artifact, error)
	}{
		{"MIPS", LoadMIPS},
		{"PreimageOracle", LoadPreimageOracle},
	}
	for _, test := range tests {
		test := test
		t.Run(test.contract, func(t *testing.T) {
			actual, err := test.method()
			require.NoError(t, err)
			require.NotNil(t, actual)
		})
	}
}
