//go:build !unix

package testutil

// isPrivateDir reports false: without owners to check, each test gets its own directory.
func isPrivateDir(string) bool {
	return false
}
