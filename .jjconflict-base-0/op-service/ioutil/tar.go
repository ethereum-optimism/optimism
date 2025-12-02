package ioutil

import (
	"archive/tar"
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Untar(outDir string, tr *tar.Reader) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		cleanedName, err := sanitizeTarPath(hdr.Name, outDir)
		if err != nil {
			return fmt.Errorf("invalid file path %q: %w", hdr.Name, err)
		}
		dst := path.Join(outDir, cleanedName)

		dirName := path.Dir(dst)
		if err := os.MkdirAll(dirName, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			if err := os.Chtimes(dst, hdr.AccessTime, hdr.ModTime); err != nil {
				return fmt.Errorf("failed to set directory times: %w", err)
			}
			continue
		}

		if err := untarFile(dst, tr, hdr); err != nil {
			return fmt.Errorf("failed to untar file: %w", err)
		}
	}
}

func untarFile(dst string, tr *tar.Reader, hdr *tar.Header) error {
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	buf := bufio.NewWriter(f)
	if _, err := io.Copy(buf, tr); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	if err := buf.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}
	if err := os.Chtimes(dst, hdr.AccessTime, hdr.ModTime); err != nil {
		return fmt.Errorf("failed to set file times: %w", err)
	}
	return nil
}

// sanitizeTarPath ensures the path is safe to extract within the specified output directory.
func sanitizeTarPath(tarPath, outDir string) (string, error) {
	cleaned := filepath.Clean(tarPath)

	if filepath.IsAbs(cleaned) {
		return "", errors.New("absolute paths are not allowed")
	}

	if strings.Contains(cleaned, "..") {
		return "", errors.New("path traversal detected")
	}

	cleaned = strings.TrimLeft(cleaned, "/\\")

	if strings.HasPrefix(cleaned, "..") {
		return "", errors.New("path traversal detected")
	}

	fullPath := filepath.Join(outDir, cleaned)
	relPath, err := filepath.Rel(outDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	if strings.HasPrefix(relPath, "..") {
		return "", errors.New("path traversal detected")
	}

	return cleaned, nil
}
