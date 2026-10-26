//go:build !unix

package forge

import "os"

// tryLock does not limit the forge runs where op-deployer has no file locks.
func tryLock(*os.File) bool {
	return true
}
