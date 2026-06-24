package deployer

import (
	"context"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/stretchr/testify/require"
)

// Guards the artifact/contract names DeployZKMockVerifier reads: ZKMockVerifier must be present in a
// local (full) build, else the name drifted.
func TestZKMockVerifierArtifactPresent(t *testing.T) {
	_, afacts := testutil.LocalArtifacts(t)
	af := &foundry.ArtifactsFS{FS: afacts}
	artifact, err := af.ReadArtifact(zkMockVerifierArtifact, zkMockVerifierContract)
	require.NoError(t, err)
	require.NotEmpty(t, artifact.Bytecode.Object, "ZKMockVerifier must have bytecode in local artifacts")
}

// The helper must fail clearly when the artifact is absent (e.g. released artifacts). The read
// happens before any client use, so a nil client is fine.
func TestDeployZKMockVerifier_MissingArtifact(t *testing.T) {
	emptyFS, ok := os.DirFS(t.TempDir()).(foundry.StatDirFs)
	require.True(t, ok)

	_, err := DeployZKMockVerifier(context.Background(), nil, nil, emptyFS)
	require.ErrorContains(t, err, "ZKMockVerifier artifact")
}
