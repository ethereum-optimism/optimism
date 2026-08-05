package deployer

import (
	"context"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/stretchr/testify/require"
)

func TestMockSP1VerifierArtifactPresent(t *testing.T) {
	_, afacts := testutil.LocalArtifacts(t)
	af := &foundry.ArtifactsFS{FS: afacts}
	artifact, err := af.ReadArtifact(mockSP1VerifierArtifact, mockSP1VerifierContract)
	require.NoError(t, err)
	require.NotEmpty(t, artifact.Bytecode.Object, "MockSP1Verifier must have bytecode in local artifacts")
}

func TestDeployMockSP1Verifier_MissingArtifact(t *testing.T) {
	emptyFS, ok := os.DirFS(t.TempDir()).(foundry.StatDirFs)
	require.True(t, ok)

	_, err := DeployMockSP1Verifier(context.Background(), nil, nil, emptyFS)
	require.ErrorContains(t, err, "MockSP1Verifier artifact")
}
