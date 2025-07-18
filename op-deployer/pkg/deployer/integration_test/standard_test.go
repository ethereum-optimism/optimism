package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/stretchr/testify/require"
)

// TestContractArtifactsIntegrity checks that the artifacts exist on GCP and are valid. Since the
// artifacts are large, this test is skipped in short mode to preserve bandwidth.
func TestContractArtifactsIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	for _, tag := range standard.AllTags() {
		t.Run(tag, func(t *testing.T) {
			t.Parallel()

			_, err := artifacts.Download(
				context.Background(),
				&artifacts.Locator{Tag: tag},
				artifacts.NoopProgressor(),
				t.TempDir(),
			)
			require.NoError(t, err)
		})
	}
}

// TestContractArtifactsExistence checks that the artifacts exist on GCP. It does not download them.
// As a result, this test does not validate the integrity of the artifacts.
func TestContractArtifactsExistence(t *testing.T) {
	for _, tag := range standard.AllTags() {
		t.Run(tag, func(t *testing.T) {
			t.Parallel()

			url, err := standard.ArtifactsURLForTag(tag)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodHead, url.String(), nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}
