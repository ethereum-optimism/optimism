package game

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
	require.Equal(t, filepath.Join(baseDir, gameDirPrefix+addr.Hex()), result)
}

func TestDiskManager_RemoveAllExcept(t *testing.T) {
	baseDir := t.TempDir()
	keep := common.Address{0x53}
	delete := common.Address{0xaa}
	disk := newDiskManager(baseDir)
	keepDir := disk.DirForGame(keep)
	deleteDir := disk.DirForGame(delete)

	unexpectedFile := filepath.Join(baseDir, "file.txt")
	require.NoError(t, os.WriteFile(unexpectedFile, []byte("test"), 0644))
	unexpectedDir := filepath.Join(baseDir, "notagame")
	require.NoError(t, os.MkdirAll(unexpectedDir, 0777))
	invalidHexDir := filepath.Join(baseDir, gameDirPrefix+"0xNOPE")
	require.NoError(t, os.MkdirAll(invalidHexDir, 0777))

	populateDir := func(dir string) []string {
		require.NoError(t, os.MkdirAll(dir, 0777))
		file1 := filepath.Join(dir, "test.txt")
		require.NoError(t, os.WriteFile(file1, []byte("foo"), 0644))
		nestedDirs := filepath.Join(dir, "subdir", "deep")
		require.NoError(t, os.MkdirAll(nestedDirs, 0777))
		file2 := filepath.Join(nestedDirs, ".foo.txt")
		require.NoError(t, os.WriteFile(file2, []byte("foo"), 0644))
		return []string{file1, file2}
	}

	keepFiles := populateDir(keepDir)
	populateDir(deleteDir)

	require.NoError(t, disk.RemoveAllExcept([]common.Address{keep}))
	require.NoDirExists(t, deleteDir, "should have deleted directory")
	for _, file := range keepFiles {
		require.FileExists(t, file, "should have kept file for active game")
	}
	require.FileExists(t, unexpectedFile, "should not delete unexpected file")
	require.DirExists(t, unexpectedDir, "should not delete unexpected dir")
	require.DirExists(t, invalidHexDir, "should not delete dir with invalid address")
}
