package cannon

import (
	_ "embed"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
)

func TestLoadState(t *testing.T) {
	tests := []struct {
		name         string
		creator      func() mipsevm.FPVMState
		supportsJSON bool
	}{
		{
			name:         "singlethreaded",
			creator:      func() mipsevm.FPVMState { return singlethreaded.CreateInitialState(234, 82) },
			supportsJSON: true,
		},
		{
			name:         "multithreaded",
			creator:      func() mipsevm.FPVMState { return multithreaded.CreateInitialState(982, 492) },
			supportsJSON: false,
		},
	}
	for _, test := range tests {
		test := test
		loadExpectedState := func(t *testing.T) *versions.VersionedState {
			state, err := versions.NewFromState(test.creator())
			require.NoError(t, err)
			return state
		}
		t.Run(test.name, func(t *testing.T) {
			t.Run("Uncompressed", func(t *testing.T) {
				if !test.supportsJSON {
					t.Skip("JSON not supported by state version")
				}
				expected := loadExpectedState(t)
				path := writeState(t, "state.json", expected)

				state, err := parseState(path)
				require.NoError(t, err)

				require.Equal(t, expected, state)
			})

			t.Run("Gzipped", func(t *testing.T) {
				if !test.supportsJSON {
					t.Skip("JSON not supported by state version")
				}
				expected := loadExpectedState(t)
				path := writeState(t, "state.json.gz", expected)

				state, err := parseState(path)
				require.NoError(t, err)

				require.Equal(t, expected, state)
			})

			t.Run("Binary", func(t *testing.T) {
				expected := loadExpectedState(t)

				path := writeState(t, "state.bin", expected)

				state, err := parseState(path)
				require.NoError(t, err)
				require.Equal(t, expected, state)
			})

			t.Run("BinaryGzip", func(t *testing.T) {
				expected := loadExpectedState(t)

				path := writeState(t, "state.bin.gz", expected)

				state, err := parseState(path)
				require.NoError(t, err)
				require.Equal(t, expected, state)
			})
		})
	}
}

func writeState(t *testing.T, filename string, state *versions.VersionedState) string {
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	require.NoError(t, serialize.Write(path, state, 0644))
	return path
}
