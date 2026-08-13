package deployer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyPipelineRejectsMockSP1VerifierOutsideGenesis(t *testing.T) {
	for _, target := range []DeploymentTarget{DeploymentTargetLive, DeploymentTargetCalldata, DeploymentTargetNoop} {
		t.Run(string(target), func(t *testing.T) {
			err := ApplyPipeline(context.Background(), ApplyPipelineOpts{
				DeploymentTarget:      target,
				DeployMockSP1Verifier: true,
			})
			require.EqualError(t, err, "mock SP1 verifier deployment is only supported for genesis")
		})
	}
}
