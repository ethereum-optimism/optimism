//go:build !unix

package forge

import "os"

// tryLock always succeeds where there are no file locks.
func tryLock(*os.File) (bool, error) {
	return true, nil
}

// checkPrivateDir accepts any directory where there are no file locks.
func checkPrivateDir(string) error {
	return nil
}
