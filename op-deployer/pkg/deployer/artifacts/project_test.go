package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts/bundle"
)

// archiveProject includes incomplete projects too, to exercise validation of downloaded bundles.
func archiveProject(t *testing.T, project string, compression string) []byte {
	t.Helper()
	var buf bytes.Buffer
	var compressor io.WriteCloser
	if compression == "gzip" {
		compressor = gzip.NewWriter(&buf)
	} else {
		var err error
		compressor, err = zstd.NewWriter(&buf)
		require.NoError(t, err)
	}
	tw := tar.NewWriter(compressor)
	require.NoError(t, filepath.WalkDir(project, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{Name: filepath.ToSlash(rel), Mode: 0o644, Size: int64(len(data))}); err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	}))
	require.NoError(t, tw.Close())
	require.NoError(t, compressor.Close())
	return buf.Bytes()
}

func TestDownload_RequiresForgeProject(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(t *testing.T, project string)
		err    string
	}{
		{name: "complete project"},
		{
			name: "artifacts only",
			change: func(t *testing.T, project string) {
				for _, path := range []string{"foundry.toml", "cache", "src"} {
					require.NoError(t, os.RemoveAll(filepath.Join(project, path)))
				}
			},
			err: "artifacts-only directories are not supported",
		},
		{
			name: "missing artifacts",
			change: func(t *testing.T, project string) {
				require.NoError(t, os.RemoveAll(filepath.Join(project, "forge-artifacts")))
			},
			err: "missing forge-artifacts",
		},
		{
			name: "artifacts is a file",
			change: func(t *testing.T, project string) {
				require.NoError(t, os.RemoveAll(filepath.Join(project, "forge-artifacts")))
				writeFiles(t, project, map[string]string{"forge-artifacts": "not a directory"})
			},
			err: "forge-artifacts is not a directory",
		},
		{
			name: "config is a directory",
			change: func(t *testing.T, project string) {
				require.NoError(t, os.Remove(filepath.Join(project, "foundry.toml")))
				require.NoError(t, os.Mkdir(filepath.Join(project, "foundry.toml"), 0o755))
				writeFiles(t, project, map[string]string{"foundry.toml/child": ""})
			},
			err: "foundry.toml is not a regular file",
		},
		{
			name: "missing compile cache",
			change: func(t *testing.T, project string) {
				require.NoError(t, os.Remove(filepath.Join(project, bundle.CompileCacheFile)))
			},
			err: "read the compile cache",
		},
		{
			name: "invalid compile cache",
			change: func(t *testing.T, project string) {
				writeFiles(t, project, map[string]string{bundle.CompileCacheFile: "invalid JSON"})
			},
			err: "decode the compile cache",
		},
		{
			name: "unsupported artifacts path",
			change: func(t *testing.T, project string) {
				writeFiles(t, project, map[string]string{bundle.CompileCacheFile: `{"paths":{"artifacts":"out"}}`})
			},
			err: `op-deployer requires "forge-artifacts"`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := writeProject(t)
			if tt.change != nil {
				tt.change(t, project)
			}
			for _, transport := range []string{"file", "gzip", "zstd", "embedded extraction"} {
				t.Run(transport, func(t *testing.T) {
					var fsys fs.FS
					var err error
					cache := t.TempDir()
					switch transport {
					case "file":
						fsys, err = Download(context.Background(), MustNewFileLocator(project), nil, cache)
					case "embedded extraction":
						fsys, err = extractZstdBundle(cache, bytes.NewReader(archiveProject(t, project, "zstd")))
					default:
						data := archiveProject(t, project, transport)
						server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							_, _ = w.Write(data)
						}))
						defer server.Close()
						ext := ".tgz"
						if transport == "zstd" {
							ext = ".tzst"
						}
						fsys, err = Download(context.Background(), MustNewLocatorFromURL(server.URL+"/artifacts"+ext), nil, cache)
					}
					if tt.err != "" {
						require.ErrorContains(t, err, tt.err)
						require.Nil(t, fsys)
						return
					}
					require.NoError(t, err)
					require.Contains(t, readArtifact(t, fsys), `"abi"`)
					root := filepath.Dir(fmt.Sprint(fsys))
					require.FileExists(t, filepath.Join(root, "foundry.toml"))
					require.FileExists(t, filepath.Join(root, "src/A.sol"))
					require.FileExists(t, filepath.Join(root, bundle.CompileCacheFile))
				})
			}
		})
	}
}

func TestDownload_RejectsMissingLocalProject(t *testing.T) {
	fsys, err := Download(context.Background(), MustNewFileLocator(filepath.Join(t.TempDir(), "missing")), nil, t.TempDir())
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.Nil(t, fsys)
}
