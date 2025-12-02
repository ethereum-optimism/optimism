package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// CopyDir copies a directory from src to dst using the provided filesystem.
// If no filesystem is provided, it uses the OS filesystem.
func CopyDir(src string, dst string, fs afero.Fs) error {
	if fs == nil {
		fs = afero.NewOsFs()
	}

	// First ensure the source exists
	srcInfo, err := fs.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source path %s is not a directory", src)
	}

	// Create the destination directory
	err = fs.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	// Walk through the source directory
	return afero.Walk(fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Construct destination path
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directories with same permissions
			return fs.MkdirAll(dstPath, info.Mode())
		}

		// Copy files
		srcFile, err := fs.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := fs.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
