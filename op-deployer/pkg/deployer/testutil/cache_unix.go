//go:build unix

package testutil

import (
	"os"
	"syscall"
)

// isPrivateDir reports whether dir is a directory, not a link, of this user that no other user can
// write.
func isPrivateDir(dir string) bool {
	info, err := os.Lstat(dir)
	if err != nil {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return info.IsDir() && ok && int(st.Uid) == os.Geteuid() && info.Mode().Perm()&0o022 == 0
}
