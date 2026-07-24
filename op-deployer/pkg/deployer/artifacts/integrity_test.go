package artifacts

import (
	"crypto/sha256"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHashIntegrityChecker_CheckIntegrity(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		setupHash   [32]byte
		expectError bool
	}{
		{
			name:        "valid hash matches data",
			data:        []byte("test data"),
			setupHash:   sha256.Sum256([]byte("test data")),
			expectError: false,
		},
		{
			name:        "invalid hash doesn't match data",
			data:        []byte("test data"),
			setupHash:   sha256.Sum256([]byte("different data")),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &hashIntegrityChecker{
				hash: tt.setupHash,
			}

			err := checker.CheckIntegrity(tt.data)

			if tt.expectError {
				require.ErrorContains(t, err, "integrity check failed")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestContentDigestDeterministicAndMetadataIndependent(t *testing.T) {
	first := fstest.MapFS{
		"a.json":        &fstest.MapFile{Data: []byte("alpha"), Mode: 0o600, ModTime: time.Unix(1, 0)},
		"nested/b.json": &fstest.MapFile{Data: []byte("beta"), Mode: 0o644, ModTime: time.Unix(2, 0)},
	}
	second := fstest.MapFS{
		"empty":         &fstest.MapFile{Mode: fs.ModeDir | 0o700, ModTime: time.Unix(30, 0)},
		"nested/b.json": &fstest.MapFile{Data: []byte("beta"), Mode: 0o400, ModTime: time.Unix(20, 0)},
		"a.json":        &fstest.MapFile{Data: []byte("alpha"), Mode: 0o777, ModTime: time.Unix(10, 0)},
	}

	firstDigest, err := ContentDigest(first)
	require.NoError(t, err)
	secondDigest, err := ContentDigest(second)
	require.NoError(t, err)
	require.Equal(t, firstDigest, secondDigest)
}

func TestContentDigestCommitsToPathsAndContents(t *testing.T) {
	original, err := ContentDigest(fstest.MapFS{
		"artifact.json": &fstest.MapFile{Data: []byte("contents")},
	})
	require.NoError(t, err)
	renamed, err := ContentDigest(fstest.MapFS{
		"renamed.json": &fstest.MapFile{Data: []byte("contents")},
	})
	require.NoError(t, err)
	changed, err := ContentDigest(fstest.MapFS{
		"artifact.json": &fstest.MapFile{Data: []byte("changed")},
	})
	require.NoError(t, err)

	require.NotEqual(t, original, renamed)
	require.NotEqual(t, original, changed)
}

func TestContentDigestPropagatesFileErrors(t *testing.T) {
	expected := errors.New("test file failure")
	base := fstest.MapFS{
		"artifact.json": &fstest.MapFile{Data: []byte("contents")},
	}

	for _, test := range []struct {
		name string
		fs   fs.FS
	}{
		{
			name: "open",
			fs: faultyFS{
				FS:        base,
				faultName: "artifact.json",
				openErr:   expected,
			},
		},
		{
			name: "read",
			fs: faultyFS{
				FS:        base,
				faultName: "artifact.json",
				readErr:   expected,
			},
		},
		{
			name: "close",
			fs: faultyFS{
				FS:        base,
				faultName: "artifact.json",
				closeErr:  expected,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ContentDigest(test.fs)
			require.ErrorIs(t, err, expected)
		})
	}
}

type faultyFS struct {
	fs.FS
	faultName string
	openErr   error
	readErr   error
	closeErr  error
}

func (f faultyFS) Open(name string) (fs.File, error) {
	if name == f.faultName && f.openErr != nil {
		return nil, f.openErr
	}
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, err
	}
	if name != f.faultName {
		return file, nil
	}
	return &faultyFile{
		File:     file,
		readErr:  f.readErr,
		closeErr: f.closeErr,
	}, nil
}

type faultyFile struct {
	fs.File
	readErr  error
	closeErr error
}

func (f *faultyFile) Read(p []byte) (int, error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return f.File.Read(p)
}

func (f *faultyFile) Close() error {
	underlyingErr := f.File.Close()
	if f.closeErr != nil {
		return f.closeErr
	}
	return underlyingErr
}
