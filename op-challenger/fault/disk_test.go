package fault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDiskManager_DirForGame(t *testing.T) {
	baseDir := t.TempDir()
	addr := common.Address{0x53}
	disk := newDiskManager(baseDir)
	result := disk.DirForGame(addr)
	require.Equal(t, filepath.Join(baseDir, addr.Hex()), result)
}

func TestDiskManager_RemoveGameData(t *testing.T) {
	baseDir := t.TempDir()
	addr := common.Address{0x53}
	disk := newDiskManager(baseDir)
	dir := disk.DirForGame(addr)

	require.NoError(t, os.MkdirAll(dir, 0777))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.txt"), []byte("foo"), 0644))
	nestedDirs := filepath.Join(dir, "subdir", "deep")
	require.NoError(t, os.MkdirAll(nestedDirs, 0777))
	require.NoError(t, os.WriteFile(filepath.Join(nestedDirs, ".foo.txt"), []byte("foo"), 0644))

	require.NoError(t, disk.RemoveGameData(addr))
	require.NoDirExists(t, dir, "should have deleted directory")
}
