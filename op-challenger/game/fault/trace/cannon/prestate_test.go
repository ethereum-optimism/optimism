package cannon

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func newCannonPrestateProvider(dataDir string, prestate string) *CannonPrestateProvider {
	return &CannonPrestateProvider{
		prestate: filepath.Join(dataDir, prestate),
	}
}

func TestAbsolutePreStateCommitment(t *testing.T) {
	dataDir := t.TempDir()

	prestate := "state.json"

	t.Run("StateUnavailable", func(t *testing.T) {
		provider := newCannonPrestateProvider("/dir/does/not/exist", prestate)
		_, err := provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("InvalidStateFile", func(t *testing.T) {
		setupPreState(t, dataDir, "invalid.json")
		provider := newCannonPrestateProvider(dataDir, prestate)
		_, err := provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorContains(t, err, "invalid mipsevm state")
	})

	t.Run("ExpectedAbsolutePreState", func(t *testing.T) {
		setupPreState(t, dataDir, "state.json")
		provider := newCannonPrestateProvider(dataDir, prestate)
		actual, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		state := mipsevm.State{
			Memory:         mipsevm.NewMemory(),
			PreimageKey:    common.HexToHash("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
			PreimageOffset: 0,
			PC:             0,
			NextPC:         1,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Step:           0,
			Registers:      [32]uint32{},
		}
		expected, err := state.EncodeWitness().StateHash()
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func setupPreState(t *testing.T, dataDir string, filename string) {
	srcDir := filepath.Join("test_data")
	path := filepath.Join(srcDir, filename)
	file, err := testData.ReadFile(path)
	require.NoErrorf(t, err, "reading %v", path)
	err = os.WriteFile(filepath.Join(dataDir, "state.json"), file, 0o644)
	require.NoErrorf(t, err, "writing %v", path)
}
