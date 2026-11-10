//go:build !unix

package forge

import "os"

// tryLock always succeeds where there are no file locks.
func tryLock(*os.File) bool {
	return true
}
