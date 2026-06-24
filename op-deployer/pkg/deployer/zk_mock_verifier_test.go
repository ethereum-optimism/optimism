package deployer

import (
	"context"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/stretchr/testify/require"
)

// TestZKMockVerifierArtifactPresent guards the artifact/contract names DeployZKMockVerifier reads:
// the test-only ZKMockVerifier must be present and deployable in a local (full) contracts build. If
// the name drifts, this fails here instead of in a devnet.
func TestZKMockVerifierArtifactPresent(t *testing.T) {
	_, afacts := testutil.LocalArtifacts(t)
	af := &foundry.ArtifactsFS{FS: afacts}
	artifact, err := af.ReadArtifact(zkMockVerifierArtifact, zkMockVerifierContract)
	require.NoError(t, err)
	require.NotEmpty(t, artifact.Bytecode.Object, "ZKMockVerifier must have bytecode in local artifacts")
}

// TestDeployZKMockVerifier_MissingArtifact verifies the helper fails with a clear, actionable error
// when the artifact is absent — e.g. when run against released artifacts (which omit test contracts)
// rather than a local full build. The artifact read happens before any client use, so a nil client
// is fine here.
func TestDeployZKMockVerifier_MissingArtifact(t *testing.T) {
	emptyFS, ok := os.DirFS(t.TempDir()).(foundry.StatDirFs)
	require.True(t, ok)

	_, err := DeployZKMockVerifier(context.Background(), nil, nil, emptyFS)
	require.ErrorContains(t, err, "ZKMockVerifier artifact")
}
