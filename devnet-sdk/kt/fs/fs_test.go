package fs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEnclaveContext struct {
	artifacts map[string][]byte
	uploaded  map[string]map[string][]byte // artifactName -> path -> content
	fs        afero.Fs                     // filesystem to use for operations
}

func (m *mockEnclaveContext) DownloadFilesArtifact(_ context.Context, name string) ([]byte, error) {
	return m.artifacts[name], nil
}

func (m *mockEnclaveContext) UploadFiles(pathToUpload string, artifactName string) (services.FilesArtifactUUID, services.FileArtifactName, error) {
	if m.uploaded == nil {
		m.uploaded = make(map[string]map[string][]byte)
	}
	m.uploaded[artifactName] = make(map[string][]byte)

	err := afero.Walk(m.fs, pathToUpload, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(pathToUpload, path)
		if err != nil {
			return err
		}

		content, err := afero.ReadFile(m.fs, path)
		if err != nil {
			return err
		}

		m.uploaded[artifactName][relPath] = content
		return nil
	})

	return "test-uuid", services.FileArtifactName(artifactName), err
}

func (m *mockEnclaveContext) GetAllFilesArtifactNamesAndUuids(ctx context.Context) ([]*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid, error) {
	return nil, nil
}

var _ EnclaveContextIface = (*mockEnclaveContext)(nil)

func createTarGzArtifact(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	for name, content := range files {
		err := tarWriter.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		})
		require.NoError(t, err)

		_, err = tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzWriter.Close())
	return buf.Bytes()
}

func TestArtifactExtraction(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		requests map[string]string
		wantErr  bool
	}{
		{
			name: "simple path",
			files: map[string]string{
				"file1.txt": "content1",
			},
			requests: map[string]string{
				"file1.txt": "content1",
			},
		},
		{
			name: "path with dot prefix",
			files: map[string]string{
				"./file1.txt": "content1",
			},
			requests: map[string]string{
				"file1.txt": "content1",
			},
		},
		{
			name: "mixed paths",
			files: map[string]string{
				"./file1.txt":  "content1",
				"file2.txt":    "content2",
				"./dir/f3.txt": "content3",
			},
			requests: map[string]string{
				"file1.txt":  "content1",
				"file2.txt":  "content2",
				"dir/f3.txt": "content3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock context with artifact
			mockCtx := &mockEnclaveContext{
				artifacts: map[string][]byte{
					"test-artifact": createTarGzArtifact(t, tt.files),
				},
				fs: afero.NewMemMapFs(),
			}

			fs, err := NewEnclaveFS(context.Background(), "test-enclave", WithEnclaveCtx(mockCtx), WithFs(mockCtx.fs))
			require.NoError(t, err)

			artifact, err := fs.GetArtifact(context.Background(), "test-artifact")
			require.NoError(t, err)

			// Create writers for all requested files
			writers := make([]*ArtifactFileWriter, 0, len(tt.requests))
			buffers := make(map[string]*bytes.Buffer, len(tt.requests))
			for reqPath := range tt.requests {
				buf := &bytes.Buffer{}
				buffers[reqPath] = buf
				writers = append(writers, NewArtifactFileWriter(reqPath, buf))
			}

			// Extract all files at once
			err = artifact.ExtractFiles(writers...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify contents
			for reqPath, wantContent := range tt.requests {
				require.Equal(t, wantContent, buffers[reqPath].String(), "content mismatch for %s", reqPath)
			}
		})
	}
}

func TestPutArtifact(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		wantErr bool
	}{
		{
			name: "single file",
			files: map[string]string{
				"file1.txt": "content1",
			},
		},
		{
			name: "multiple files",
			files: map[string]string{
				"file1.txt":     "content1",
				"dir/file2.txt": "content2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			mockCtx := &mockEnclaveContext{
				artifacts: make(map[string][]byte),
				fs:        fs,
			}

			enclaveFs := &EnclaveFS{
				enclaveCtx: mockCtx,
				fs:         fs,
			}

			// Create readers for all files
			var readers []*ArtifactFileReader
			for path, content := range tt.files {
				readers = append(readers, NewArtifactFileReader(
					path,
					bytes.NewReader([]byte(content)),
				))
			}

			// Put the artifact
			err := enclaveFs.PutArtifact(context.Background(), "test-artifact", readers...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify uploaded contents
			require.NotNil(t, mockCtx.uploaded)
			uploaded := mockCtx.uploaded["test-artifact"]
			require.NotNil(t, uploaded)
			require.Equal(t, len(tt.files), len(uploaded))

			for path, wantContent := range tt.files {
				content, exists := uploaded[path]
				require.True(t, exists, "missing file: %s", path)
				require.Equal(t, wantContent, string(content), "content mismatch for %s", path)
			}
		})
	}
}

func TestMultipleExtractCalls(t *testing.T) {
	// Create a test artifact with multiple files
	files := map[string]string{
		"file1.txt":     "content1",
		"file2.txt":     "content2",
		"dir/file3.txt": "content3",
		"dir/file4.txt": "content4",
	}

	// Create mock context with artifact
	mockCtx := &mockEnclaveContext{
		artifacts: map[string][]byte{
			"test-artifact": createTarGzArtifact(t, files),
		},
		fs: afero.NewMemMapFs(),
	}

	fs, err := NewEnclaveFS(context.Background(), "test-enclave", WithEnclaveCtx(mockCtx), WithFs(mockCtx.fs))
	require.NoError(t, err)

	artifact, err := fs.GetArtifact(context.Background(), "test-artifact")
	require.NoError(t, err)

	// First extraction - get file1.txt and file3.txt
	firstExtractFiles := map[string]string{
		"file1.txt":     "content1",
		"dir/file3.txt": "content3",
	}

	firstWriters := make([]*ArtifactFileWriter, 0, len(firstExtractFiles))
	firstBuffers := make(map[string]*bytes.Buffer, len(firstExtractFiles))

	for reqPath := range firstExtractFiles {
		buf := &bytes.Buffer{}
		firstBuffers[reqPath] = buf
		firstWriters = append(firstWriters, NewArtifactFileWriter(reqPath, buf))
	}

	// First extraction
	err = artifact.ExtractFiles(firstWriters...)
	require.NoError(t, err)

	// Verify first extraction
	for reqPath, wantContent := range firstExtractFiles {
		require.Equal(t, wantContent, firstBuffers[reqPath].String(),
			"first extraction: content mismatch for %s", reqPath)
	}

	// Second extraction - get file2.txt and file4.txt
	secondExtractFiles := map[string]string{
		"file2.txt":     "content2",
		"dir/file4.txt": "content4",
	}

	secondWriters := make([]*ArtifactFileWriter, 0, len(secondExtractFiles))
	secondBuffers := make(map[string]*bytes.Buffer, len(secondExtractFiles))

	for reqPath := range secondExtractFiles {
		buf := &bytes.Buffer{}
		secondBuffers[reqPath] = buf
		secondWriters = append(secondWriters, NewArtifactFileWriter(reqPath, buf))
	}

	// Second extraction using the same artifact
	err = artifact.ExtractFiles(secondWriters...)
	require.NoError(t, err)

	// Verify second extraction
	for reqPath, wantContent := range secondExtractFiles {
		require.Equal(t, wantContent, secondBuffers[reqPath].String(),
			"second extraction: content mismatch for %s", reqPath)
	}

	// Third extraction - extract all files again to prove we can keep extracting
	allFiles := map[string]string{
		"file1.txt":     "content1",
		"file2.txt":     "content2",
		"dir/file3.txt": "content3",
		"dir/file4.txt": "content4",
	}

	allWriters := make([]*ArtifactFileWriter, 0, len(allFiles))
	allBuffers := make(map[string]*bytes.Buffer, len(allFiles))

	for reqPath := range allFiles {
		buf := &bytes.Buffer{}
		allBuffers[reqPath] = buf
		allWriters = append(allWriters, NewArtifactFileWriter(reqPath, buf))
	}

	// Third extraction
	err = artifact.ExtractFiles(allWriters...)
	require.NoError(t, err)

	// Verify third extraction
	for reqPath, wantContent := range allFiles {
		require.Equal(t, wantContent, allBuffers[reqPath].String(),
			"third extraction: content mismatch for %s", reqPath)
	}
}

func TestArtifact_Download(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string][]byte // map of filepath to content
		wantErr  bool
		validate func(t *testing.T, fs afero.Fs)
	}{
		{
			name: "single file download",
			files: map[string][]byte{
				"test.txt": []byte("hello world"),
			},
			validate: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "test.txt")
				require.NoError(t, err)
				assert.Equal(t, []byte("hello world"), content)
			},
		},
		{
			name: "nested directory structure",
			files: map[string][]byte{
				"dir/test.txt":        []byte("hello"),
				"dir/subdir/test.txt": []byte("world"),
			},
			validate: func(t *testing.T, fs afero.Fs) {
				content1, err := afero.ReadFile(fs, "dir/test.txt")
				require.NoError(t, err)
				assert.Equal(t, []byte("hello"), content1)

				content2, err := afero.ReadFile(fs, "dir/subdir/test.txt")
				require.NoError(t, err)
				assert.Equal(t, []byte("world"), content2)
			},
		},
		{
			name: "empty directory",
			files: map[string][]byte{
				"dir/": nil,
			},
			validate: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.DirExists(fs, "dir")
				require.NoError(t, err)
				assert.True(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a tar.gz archive in memory
			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			tw := tar.NewWriter(gw)

			// Add files to the archive
			for path, content := range tt.files {
				header := &tar.Header{
					Name: path,
				}
				if content == nil {
					header.Typeflag = tar.TypeDir
					header.Mode = 0755
				} else {
					header.Typeflag = tar.TypeReg
					header.Size = int64(len(content))
					header.Mode = 0644
				}

				err := tw.WriteHeader(header)
				require.NoError(t, err)

				if content != nil {
					_, err = tw.Write(content)
					require.NoError(t, err)
				}
			}

			err := tw.Close()
			require.NoError(t, err)
			err = gw.Close()
			require.NoError(t, err)

			// Create in-memory filesystem
			memFs := afero.NewMemMapFs()

			// Create an Artifact from the archive
			rawData := buf.Bytes()
			zipReader, err := gzip.NewReader(bytes.NewReader(rawData))
			require.NoError(t, err)
			artifact := &Artifact{
				rawData: rawData,
				reader:  tar.NewReader(zipReader),
				fs:      memFs,
			}

			// Test Download function
			err = artifact.Download("")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Run validation
			tt.validate(t, memFs)
		})
	}
}
