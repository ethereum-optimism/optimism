package artifacts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTempDirTracker(t *testing.T) {
	tracker := NewTempDirTracker()

	require.Empty(t, tracker.GetTempDirs())

	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	tracker.Add(tempDir1)
	tracker.Add(tempDir2)

	dirs := tracker.GetTempDirs()
	require.Len(t, dirs, 2)
	require.Contains(t, dirs, tempDir1)
	require.Contains(t, dirs, tempDir2)

	testFile := filepath.Join(tempDir1, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	require.FileExists(t, testFile)

	require.NoError(t, tracker.Cleanup())

	require.NoDirExists(t, tempDir1)
	require.NoDirExists(t, tempDir2)

	require.Empty(t, tracker.GetTempDirs())
}

func TestTempDirTrackerCleanupNonexistent(t *testing.T) {
	tracker := NewTempDirTracker()

	tracker.Add("/nonexistent/directory")

	require.NoError(t, tracker.Cleanup())
}
