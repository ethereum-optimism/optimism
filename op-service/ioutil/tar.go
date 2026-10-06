package ioutil

import (
	"archive/tar"
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Untar extracts tr into outDir, which must already exist. All writes go through an
// os.Root, so they stay inside outDir even if an entry name gets past sanitizeTarPath.
func Untar(outDir string, tr *tar.Reader) error {
	root, err := os.OpenRoot(outDir)
	if err != nil {
		return fmt.Errorf("failed to open output directory: %w", err)
	}
	defer root.Close()

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

		if err := root.MkdirAll(filepath.Dir(cleanedName), 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if hdr.FileInfo().IsDir() {
			if err := root.MkdirAll(cleanedName, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			if err := root.Chtimes(cleanedName, hdr.AccessTime, hdr.ModTime); err != nil {
				return fmt.Errorf("failed to set directory times: %w", err)
			}
			continue
		}

		if err := untarFile(root, cleanedName, tr, hdr); err != nil {
			return fmt.Errorf("failed to untar file: %w", err)
		}
	}
}

func untarFile(root *os.Root, name string, tr *tar.Reader, hdr *tar.Header) error {
	f, err := root.Create(name)
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
	if err := root.Chtimes(name, hdr.AccessTime, hdr.ModTime); err != nil {
		return fmt.Errorf("failed to set file times: %w", err)
	}
	return nil
}

// sanitizeTarPath ensures the path is safe to extract within the specified output directory.
func sanitizeTarPath(tarPath, outDir string) (string, error) {
	absBase, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base directory: %w", err)
	}

	cleaned := filepath.Clean(tarPath)
	if filepath.IsAbs(cleaned) {
		return "", errors.New("absolute paths are not allowed")
	}

	cleaned = strings.TrimLeft(cleaned, "/\\")
	destPath := filepath.Join(absBase, cleaned)
	absDest, err := filepath.Abs(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve destination path: %w", err)
	}

	basePrefix := absBase + string(os.PathSeparator)
	if !strings.HasPrefix(absDest, basePrefix) && absDest != absBase {
		return "", errors.New("path traversal detected")
	}

	return cleaned, nil
}
