//go:build !cannon64
// +build !cannon64

package versions

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/op-service/serialize"
)

func TestNewFromState(t *testing.T) {
	for _, version := range StateVersionTypes {
		if IsSupportedSingleThreaded(version) {
			t.Run(version.String(), func(t *testing.T) {
				actual, err := NewFromState(version, singlethreaded.CreateEmptyState())
				require.NoError(t, err)
				require.IsType(t, &singlethreaded.State{}, actual.FPVMState)
				require.Equal(t, version, actual.Version)
			})
			t.Run(version.String()+"-mt-unsupported", func(t *testing.T) {
				_, err := NewFromState(version, multithreaded.CreateEmptyState())
				require.ErrorIs(t, err, ErrUnsupportedVersion)
			})
		} else if IsSupportedMultiThreaded(version) {
			t.Run(version.String(), func(t *testing.T) {
				actual, err := NewFromState(version, multithreaded.CreateEmptyState())
				require.NoError(t, err)
				require.IsType(t, &multithreaded.State{}, actual.FPVMState)
				require.Equal(t, version, actual.Version)
			})
			t.Run(version.String()+"-st-unsupported", func(t *testing.T) {
				_, err := NewFromState(version, singlethreaded.CreateEmptyState())
				require.ErrorIs(t, err, ErrUnsupportedVersion)
			})
		} else {
			t.Run(version.String()+"-unsupported", func(t *testing.T) {
				_, err := NewFromState(version, multithreaded.CreateEmptyState())
				require.ErrorIs(t, err, ErrUnsupportedVersion)
			})
		}
	}
}

func TestLoadStateFromFile(t *testing.T) {
	for _, version := range StateVersionTypes {
		if IsSupportedSingleThreaded(version) {
			t.Run(version.String(), func(t *testing.T) {
				expected, err := NewFromState(version, singlethreaded.CreateEmptyState())
				require.NoError(t, err)

				path := writeToFile(t, "state.bin.gz", expected)
				actual, err := LoadStateFromFile(path)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		}
		if IsSupportedMultiThreaded(version) {
			t.Run(version.String(), func(t *testing.T) {
				expected, err := NewFromState(version, multithreaded.CreateEmptyState())
				require.NoError(t, err)

				path := writeToFile(t, "state.bin.gz", expected)
				actual, err := LoadStateFromFile(path)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		}
	}

	t.Run("JSONUnsupported", func(t *testing.T) {
		filename := strconv.Itoa(int(VersionSingleThreaded)) + ".json"
		dir := t.TempDir()
		path := filepath.Join(dir, filename)
		in, err := historicStates.ReadFile(filepath.Join(statesPath, filename))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, in, 0o644))

		_, err = LoadStateFromFile(path)
		require.ErrorIs(t, err, ErrUnsupportedVersion)
	})
}

type versionAndStateCreator struct {
	version     StateVersion
	createState func() mipsevm.FPVMState
}

func TestVersionsOtherThanZeroDoNotSupportJSON(t *testing.T) {
	var tests []struct {
		version     StateVersion
		createState func() mipsevm.FPVMState
	}
	for _, version := range StateVersionTypes {
		if !IsSupportedSingleThreaded(version) {
			continue
		}
		tests = append(tests, versionAndStateCreator{version: version, createState: func() mipsevm.FPVMState { return singlethreaded.CreateEmptyState() }})
	}
	for _, version := range StateVersionTypes {
		if !IsSupportedMultiThreaded(version) {
			continue
		}
		tests = append(tests, versionAndStateCreator{version: version, createState: func() mipsevm.FPVMState { return multithreaded.CreateEmptyState() }})
	}
	for _, test := range tests {
		test := test
		t.Run(test.version.String(), func(t *testing.T) {
			state, err := NewFromState(test.version, test.createState())
			require.NoError(t, err)

			dir := t.TempDir()
			path := filepath.Join(dir, "test.json")
			err = serialize.Write(path, state, 0o644)
			require.ErrorIs(t, err, ErrJsonNotSupported)
		})
	}
}

func writeToFile(t *testing.T, filename string, data serialize.Serializable) string {
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	require.NoError(t, serialize.Write(path, data, 0o644))
	return path
}
