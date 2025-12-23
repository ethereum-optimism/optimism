//go:build windows

package preimage

func setNonblock(fd int) {
	// SetNonblock is not supported on Windows
}
