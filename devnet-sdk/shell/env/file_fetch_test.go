package env

import (
	"net/url"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchFileDataFromOS(t *testing.T) {
	fs := afero.NewMemMapFs()

	var (
		absoluteContent = []byte(`{"name": "absolute"}`)
		relativeContent = []byte(`{"name": "relative"}`)
		noNameContent   = []byte(`{}`)
	)

	err := afero.WriteFile(fs, "/some/absolute/path", absoluteContent, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "some/relative/path", relativeContent, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "some/file/devnet-env.json", noNameContent, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "some/file/devnet", noNameContent, 0644)
	require.NoError(t, err)

	fetcher := &fileFetcher{
		fs: fs,
	}

	tests := []struct {
		name        string
		urlStr      string
		wantName    string
		wantContent []byte
		wantError   bool
	}{
		{
			name:        "file URL",
			urlStr:      "file:///some/absolute/path",
			wantName:    "absolute",
			wantContent: absoluteContent,
		},
		{
			name:        "absolute path",
			urlStr:      "/some/absolute/path",
			wantName:    "absolute",
			wantContent: absoluteContent,
		},
		{
			name:        "relative path",
			urlStr:      "some/relative/path",
			wantName:    "relative",
			wantContent: relativeContent,
		},
		{
			name:        "no name - file with extension",
			urlStr:      "some/file/devnet-env.json",
			wantName:    "devnet-env",
			wantContent: noNameContent,
		},
		{
			name:        "no name - file without extension",
			urlStr:      "some/file/devnet",
			wantName:    "devnet",
			wantContent: noNameContent,
		},
		{
			name:      "non-existent file",
			urlStr:    "file:///nonexistent/path",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.urlStr)
			require.NoError(t, err)

			env, err := fetcher.fetchFileData(u)
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, env.Name)
		})
	}
}
