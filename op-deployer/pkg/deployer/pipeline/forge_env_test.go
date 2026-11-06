package pipeline

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// newTestGenesisEnv returns an env that runs the local scripts offline on an empty genesis state.
func newTestGenesisEnv(t *testing.T, deployer common.Address) *Env {
	t.Helper()
	_, afs := testutil.LocalArtifacts(t)
	client, err := opcm.NewForgeClient(afs, "", opcm.L1ForgeScripts)
	require.NoError(t, err)
	ctx := context.Background()
	return &Env{
		Logger:   testlog.Logger(t, slog.LevelDebug),
		Deployer: deployer,
		Forge: &opcm.ForgeEnv{
			Client:  client,
			Context: ctx,
			Genesis: opcm.NewGenesisState(deployer, true),
		},
		L1Artifacts:               &foundry.ArtifactsFS{FS: afs},
		IsGenesis:                 true,
		AllowUnoptimizedContracts: true,
		Context:                   ctx,
	}
}
