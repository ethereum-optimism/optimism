package opcm

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactsLocator_Marshaling(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  *ArtifactsLocator
		err  bool
	}{
		{
			name: "valid tag",
			in:   "tag://op-contracts/v1.6.0",
			out: &ArtifactsLocator{
				Tag: "op-contracts/v1.6.0",
			},
			err: false,
		},
		{
			name: "well-formed but nonexistent tag",
			in:   "tag://op-contracts/v1.5.0",
			out:  nil,
			err:  true,
		},
		{
			name: "mal-formed tag",
			in:   "tag://honk",
			out:  nil,
			err:  true,
		},
		{
			name: "valid HTTPS URL",
			in:   "https://example.com",
			out: &ArtifactsLocator{
				URL: parseUrl(t, "https://example.com"),
			},
			err: false,
		},
		{
			name: "valid file URL",
			in:   "file:///tmp/artifacts",
			out: &ArtifactsLocator{
				URL: parseUrl(t, "file:///tmp/artifacts"),
			},
			err: false,
		},
		{
			name: "empty",
			in:   "",
			out:  nil,
			err:  true,
		},
		{
			name: "no scheme",
			in:   "example.com",
			out:  nil,
			err:  true,
		},
		{
			name: "unsupported scheme",
			in:   "http://example.com",
			out:  nil,
			err:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a ArtifactsLocator
			err := a.UnmarshalText([]byte(tt.in))
			if tt.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.out, &a)

			marshalled, err := a.MarshalText()
			require.NoError(t, err)
			require.Equal(t, tt.in, string(marshalled))
		})
	}
}

func parseUrl(t *testing.T, u string) *url.URL {
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	return parsed
}
