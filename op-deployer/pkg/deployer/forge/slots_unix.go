//go:build unix

package forge

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// tryLock locks f without waiting, and returns false when another holder has it. Any other error is
// returned: a cache directory on NFS without a lock daemon would otherwise look busy forever.
// Closing f unlocks it.
func tryLock(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}
	return err == nil, err
}

// checkPrivateDir checks that dir is a directory, not a link, of this user that no other user can
// write. In the shared temporary directory, where slotDir falls back to, its name is predictable,
// so another user could create it first to hold its locks or plant links in it; the sticky bit there
// keeps them from replacing it after this check.
func checkPrivateDir(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("failed to check the forge slot directory: %w", err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || !ok || int(st.Uid) != os.Getuid() || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("the forge slot directory %s (mode %v) is not a private directory of uid %d: remove it, or, if it is in the shared temporary directory, set TMPDIR to a private directory", dir, info.Mode(), os.Getuid())
	}
	return nil
}
