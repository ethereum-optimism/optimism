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
	)

	err := afero.WriteFile(fs, "/some/absolute/path", absoluteContent, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "some/relative/path", relativeContent, 0644)
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
			wantName:    "path",
			wantContent: absoluteContent,
		},
		{
			name:        "absolute path",
			urlStr:      "/some/absolute/path",
			wantName:    "path",
			wantContent: absoluteContent,
		},
		{
			name:        "relative path",
			urlStr:      "some/relative/path",
			wantName:    "path",
			wantContent: relativeContent,
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
