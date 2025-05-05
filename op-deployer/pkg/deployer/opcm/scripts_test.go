package opcm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewScripts(t *testing.T) {
	t.Run("should not fail with current versions of script contracts", func(t *testing.T) {
		// First we grab a test host
		host := createTestHost(t)

		// Then we load the scripts
		//
		// This would raise an error if the Go types didn't match the ABI
		scripts, err := NewScripts(host)
		require.NoError(t, err)

		// And we just make sure we have all the scripts loaded
		require.NotNil(t, scripts.DeployImplementations)
		require.NotNil(t, scripts.DeploySuperchain)
		require.NotNil(t, scripts.DeployAlphabetVM)
		require.NotNil(t, scripts.DeployAltDA)
		require.NotNil(t, scripts.DeployAsterisc)
		require.NotNil(t, scripts.DeployDisputeGame)
		require.NotNil(t, scripts.DeployMIPS)
		require.NotNil(t, scripts.DeployPreimageOracle)
		require.NotNil(t, scripts.DeployProxy)
	})
}
