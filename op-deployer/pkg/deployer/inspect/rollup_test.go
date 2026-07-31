package inspect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/stretchr/testify/require"
)

func TestRollupCLI_WritesFileAfterPrepareOnly(t *testing.T) {
	workdir, chainID := buildPreparedWorkdir(t)
	outfile := filepath.Join(t.TempDir(), "rollup.json")

	ctx := newInspectCtx(t, workdir, outfile, chainID.Hex())
	require.NoError(t, RollupCLI(ctx))

	data, err := os.ReadFile(outfile)
	require.NoError(t, err)

	var got rollup.Config
	require.NoError(t, json.Unmarshal(data, &got))
	require.NotZero(t, got.L2ChainID, "rendered rollup config must carry the chain ID")
}

func TestRollupCLI_ErrorsWhenNeitherPreparedNorApplied(t *testing.T) {
	workdir, chainID := buildPreparedWorkdir(t)

	st, err := pipeline.ReadState(workdir)
	require.NoError(t, err)
	st.Prepared = false
	st.PreparedDeployment = nil
	require.NoError(t, pipeline.WriteState(workdir, st))

	outfile := filepath.Join(t.TempDir(), "rollup.json")
	ctx := newInspectCtx(t, workdir, outfile, chainID.Hex())
	err = RollupCLI(ctx)
	require.ErrorContains(t, err, "neither prepared nor applied")
}

// TestRollupCLI_StillUsesAppliedIntentWhenApplied mirrors the genesis regression guard: once
// applied, rollup rendering must keep using the frozen AppliedIntent, not a diverged live
// intent.toml.
func TestRollupCLI_StillUsesAppliedIntentWhenApplied(t *testing.T) {
	workdir, chainID := buildPreparedWorkdir(t)

	intent, err := pipeline.ReadIntent(workdir)
	require.NoError(t, err)
	st, err := pipeline.ReadState(workdir)
	require.NoError(t, err)
	st.AppliedIntent = intent
	require.NoError(t, pipeline.WriteState(workdir, st))

	intent.FundDevAccounts = true
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))

	outfile := filepath.Join(t.TempDir(), "rollup.json")
	ctx := newInspectCtx(t, workdir, outfile, chainID.Hex())
	require.NoError(t, RollupCLI(ctx), "must render from the frozen AppliedIntent, not the diverged live intent.toml")
}
