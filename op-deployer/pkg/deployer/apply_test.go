package deployer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyPipelineRejectsMockSP1VerifierOutsideGenesis(t *testing.T) {
	err := ApplyPipeline(context.Background(), ApplyPipelineOpts{
		DeploymentTarget:      DeploymentTargetLive,
		DeployMockSP1Verifier: true,
	})
	require.EqualError(t, err, "mock SP1 verifier deployment is only supported for genesis")
}
