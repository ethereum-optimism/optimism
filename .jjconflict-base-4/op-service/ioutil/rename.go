package ioutil

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"errors"
)

// SafeRename attempts to rename a file from source to destination.
// If the rename fails due to cross-device link error, it falls back to copying the file and deleting the source.
func SafeRename(source, destination string) error {
	// First see if we can just rename the file normally
	err := os.Rename(source, destination)

	// If we get an "invalid cross-device link" error, we need to do a copy and delete
	if err != nil && errors.Is(err, syscall.EXDEV) {
		return renameCrossDevice(source, destination)
	}

	return err
}

func renameCrossDevice(source, destination string) error {
	// Open the source file
	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("rename: failed to open source file %s: %w", source, err)
	}

	// Create the destination file
	dst, err := os.Create(destination)
	if err != nil {
		// Make sure to close the source file before returning
		src.Close()

		return fmt.Errorf("rename: failed to create destination file %s: %w", destination, err)
	}

	// Copy the contents over
	_, err = io.Copy(dst, src)

	// Close both files
	src.Close()
	dst.Close()

	if err != nil {
		return fmt.Errorf("rename: failed to copy source %s to destination %s: %w", source, destination, err)
	}

	// Get source file permissions
	fileInfo, err := os.Stat(source)
	if err != nil {
		// Remove the destination file if we fail to stat the source
		os.Remove(destination)

		return fmt.Errorf("rename: failed to stat source %s: %w", source, err)
	}

	// Apply source file permissions to destination
	err = os.Chmod(destination, fileInfo.Mode())
	if err != nil {
		// Remove the destination file if we fail to apply the permissions
		os.Remove(destination)

		return fmt.Errorf("rename: failed to apply file permissions to destination %s: %w", destination, err)
	}

	// Delete the source file
	os.Remove(source)

	return nil
}
