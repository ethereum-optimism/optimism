package env

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKurtosisURL(t *testing.T) {
	tests := []struct {
		name           string
		urlStr         string
		wantEnclave    string
		wantArtifact   string
		wantFile       string
		wantParseError bool
	}{
		{
			name:         "basic url",
			urlStr:       "kt://myenclave",
			wantEnclave:  "myenclave",
			wantArtifact: "",
			wantFile:     "env.json",
		},
		{
			name:         "with artifact",
			urlStr:       "kt://myenclave/custom-artifact",
			wantEnclave:  "myenclave",
			wantArtifact: "custom-artifact",
			wantFile:     "env.json",
		},
		{
			name:         "with artifact and file",
			urlStr:       "kt://myenclave/custom-artifact/config.json",
			wantEnclave:  "myenclave",
			wantArtifact: "custom-artifact",
			wantFile:     "config.json",
		},
		{
			name:         "with trailing slash",
			urlStr:       "kt://enclave/artifact/",
			wantEnclave:  "enclave",
			wantArtifact: "artifact",
			wantFile:     "env.json",
		},
		{
			name:           "invalid url",
			urlStr:         "://invalid",
			wantParseError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.urlStr)
			if tt.wantParseError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			enclave, artifact, file := ktFetcher.parseKurtosisURL(u)
			assert.Equal(t, tt.wantEnclave, enclave)
			assert.Equal(t, tt.wantArtifact, artifact)
			assert.Equal(t, tt.wantFile, file)
		})
	}
}
