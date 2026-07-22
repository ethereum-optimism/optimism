package inspect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"
)

func TestGenesisCLI_WritesFileAfterPrepareOnly(t *testing.T) {
	workdir, chainID := buildPreparedWorkdir(t)
	outfile := filepath.Join(t.TempDir(), "genesis.json")

	ctx := newInspectCtx(t, workdir, outfile, chainID.Hex())
	require.NoError(t, GenesisCLI(ctx))

	data, err := os.ReadFile(outfile)
	require.NoError(t, err)

	var got core.Genesis
	require.NoError(t, json.Unmarshal(data, &got))
	require.NotZero(t, got.Timestamp, "rendered genesis must carry the pinned genesis time")
}

func TestGenesisCLI_ErrorsWhenNeitherPreparedNorApplied(t *testing.T) {
	workdir, chainID := buildPreparedWorkdir(t)

	st, err := pipeline.ReadState(workdir)
	require.NoError(t, err)
	st.Prepared = false
	require.NoError(t, pipeline.WriteState(workdir, st))

	outfile := filepath.Join(t.TempDir(), "genesis.json")
	ctx := newInspectCtx(t, workdir, outfile, chainID.Hex())
	err = GenesisCLI(ctx)
	require.ErrorContains(t, err, "neither prepared nor applied")
}

// TestGenesisCLI_StillUsesAppliedIntentWhenApplied guards the frozen-snapshot invariant: once a
// chain is applied, genesis must render from the AppliedIntent snapshot, never a live intent.toml
// that has since diverged.
func TestGenesisCLI_StillUsesAppliedIntentWhenApplied(t *testing.T) {
	workdir, chainID := buildPreparedWorkdir(t)

	intent, err := pipeline.ReadIntent(workdir)
	require.NoError(t, err)
	st, err := pipeline.ReadState(workdir)
	require.NoError(t, err)
	st.AppliedIntent = intent
	require.NoError(t, pipeline.WriteState(workdir, st))

	// The live intent.toml now diverges from what was actually applied.
	intent.FundDevAccounts = true
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))

	outfile := filepath.Join(t.TempDir(), "genesis.json")
	ctx := newInspectCtx(t, workdir, outfile, chainID.Hex())
	require.NoError(t, GenesisCLI(ctx), "must render from the frozen AppliedIntent, not the diverged live intent.toml")
}
