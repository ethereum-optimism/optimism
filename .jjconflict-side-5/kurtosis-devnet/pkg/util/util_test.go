package util

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDir(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string
		src         string
		dst         string
		expectError bool
	}{
		{
			name: "successful copy of directory with files",
			setupFiles: map[string]string{
				"/src/file1.txt":        "content1",
				"/src/file2.txt":        "content2",
				"/src/subdir/file3.txt": "content3",
			},
			src:         "/src",
			dst:         "/dst",
			expectError: false,
		},
		{
			name:        "source directory does not exist",
			setupFiles:  map[string]string{},
			src:         "/nonexistent",
			dst:         "/dst",
			expectError: true,
		},
		{
			name: "source is not a directory",
			setupFiles: map[string]string{
				"/src": "file content",
			},
			src:         "/src",
			dst:         "/dst",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new memory filesystem for each test
			fs := afero.NewMemMapFs()

			// Set up test files
			for path, content := range tt.setupFiles {
				dir := filepath.Dir(path)
				err := fs.MkdirAll(dir, 0755)
				require.NoError(t, err, "Failed to create directory")

				err = afero.WriteFile(fs, path, []byte(content), 0644)
				require.NoError(t, err, "Failed to write test file")
			}

			// Execute the copy
			err := CopyDir(tt.src, tt.dst, fs)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Verify the copied files
			for srcPath, expectedContent := range tt.setupFiles {
				// Skip if the source path is not a file (e.g., when testing error cases)
				info, err := fs.Stat(srcPath)
				if err != nil || !info.Mode().IsRegular() {
					continue
				}

				// Calculate destination path
				relPath, err := filepath.Rel(tt.src, srcPath)
				require.NoError(t, err)
				dstPath := filepath.Join(tt.dst, relPath)

				// Verify file exists and content matches
				content, err := afero.ReadFile(fs, dstPath)
				assert.NoError(t, err, "Failed to read copied file: %s", dstPath)
				assert.Equal(t, expectedContent, string(content), "Content mismatch for file: %s", dstPath)

				// Verify permissions
				srcInfo, err := fs.Stat(srcPath)
				require.NoError(t, err)
				dstInfo, err := fs.Stat(dstPath)
				require.NoError(t, err)
				assert.Equal(t, srcInfo.Mode(), dstInfo.Mode(), "Mode mismatch for file: %s", dstPath)
			}
		})
	}
}
