package ioutil

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
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

		cleanedName := path.Clean(hdr.Name)
		if strings.Contains(cleanedName, "..") {
			return fmt.Errorf("invalid file path: %s", hdr.Name)
		}
		dst := path.Join(outDir, cleanedName)
		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		if err := untarFile(dst, tr); err != nil {
			return fmt.Errorf("failed to untar file: %w", err)
		}
	}
}

func untarFile(dst string, tr *tar.Reader) error {
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
	return nil
}
