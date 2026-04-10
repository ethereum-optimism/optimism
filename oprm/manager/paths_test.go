package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateMonorepoRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "op-node"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte(optimismModuleLine+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "justfile"), []byte("default:\n\t@true\n"), 0o644))
	require.NoError(t, ValidateMonorepoRoot(root))
}

func TestValidateMonorepoRootRejectsWrongDirectory(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/not-optimism\n"), 0o644))
	err := ValidateMonorepoRoot(root)
	require.Error(t, err)
	require.Contains(t, err.Error(), "oprm must be run from the optimism monorepo root")
}

func TestOpGethPathDefaultsRelativeToMonorepoRoot(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MonorepoPath = "/tmp/optimism"
	app := NewWithProviders(cfg, nil, nil, nil, nil, nil, nil)
	require.Equal(t, filepath.Clean("/tmp/op-geth"), app.OpGethPath())
}
