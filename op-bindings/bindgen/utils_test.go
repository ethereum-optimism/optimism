package bindgen

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadExpectedAbigenVersion(t *testing.T) {
	// Create a temporary directory for the version control file.
	tmpDir := path.Join(os.TempDir(), "version-tests")
	defer os.RemoveAll(tmpDir)
	require.NoError(t, os.MkdirAll(tmpDir, 0755))

	// Create a temporary version control file.
	versionFile := path.Join(tmpDir, "versions.json")
	versions := Versions{Abigen: "v1.2.3"}

	// Marshal the versions to JSON.
	versionsJSON, err := json.Marshal(versions)
	require.NoError(t, err)

	// Write the JSON to the version control file.
	require.NoError(t, os.WriteFile(versionFile, versionsJSON, 0644))

	// Read the expected version from the version control file.
	// The read version should not have a "v" prefix.
	expectedVersion, err := readExpectedAbigenVersion(tmpDir)
	require.NoError(t, err)
	require.Equal(t, expectedVersion, "1.2.3")
}
