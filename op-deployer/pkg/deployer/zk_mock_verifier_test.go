package deployer

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestZKMockVerifierDeployData_EmbeddedArtifacts(t *testing.T) {
	afacts, err := artifacts.ExtractEmbedded(t.TempDir())
	require.NoError(t, err)

	deployData, err := zkMockVerifierDeployData(afacts, common.Address{'D'})
	require.NoError(t, err)
	require.NotEmpty(t, deployData)
}

func TestZKMockVerifierDeployData_MissingArtifact(t *testing.T) {
	emptyFS, ok := os.DirFS(t.TempDir()).(foundry.StatDirFs)
	require.True(t, ok)

	_, err := zkMockVerifierDeployData(emptyFS, common.Address{'D'})
	require.ErrorContains(t, err, "DeployZKMockVerifier script")
}
