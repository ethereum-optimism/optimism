package opnode

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func newDepsetCtx(t *testing.T, args ...string) *cli.Context {
	t.Helper()
	app := &cli.App{Flags: []cli.Flag{flags.InteropDependencySet}}
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	require.NoError(t, flags.InteropDependencySet.Apply(set))
	require.NoError(t, set.Parse(args))
	return cli.NewContext(app, set, nil)
}

func TestNewDependencySetFromCLI_ExplicitJSONOverridesRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "depset.json")
	// Fictional chain ID so the JSON contents are distinguishable from registry output.
	payload := []byte(`{"dependencies":{"4242":{}}}`)
	require.NoError(t, os.WriteFile(path, payload, 0o600))

	ctx := newDepsetCtx(t, "--"+flags.InteropDependencySet.Name+"="+path)
	ds, err := NewDependencySetFromCLI(ctx, eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	require.NotNil(t, ds)
	require.True(t, ds.HasChain(eth.ChainIDFromUInt64(4242)))
	require.False(t, ds.HasChain(eth.ChainIDFromUInt64(10)),
		"explicit JSON must win over the registry fallback")
}

func TestNewDependencySetFromCLI_RegistryFallback_KnownChain(t *testing.T) {
	ctx := newDepsetCtx(t)
	ds, err := NewDependencySetFromCLI(ctx, eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	require.NotNil(t, ds, "registry-known chain must yield a depset")
	require.True(t, ds.HasChain(eth.ChainIDFromUInt64(10)),
		"registry depset must at minimum contain the chain itself")
}

func TestNewDependencySetFromCLI_RegistryFallback_UnknownChain_Nil(t *testing.T) {
	ctx := newDepsetCtx(t)
	ds, err := NewDependencySetFromCLI(ctx, eth.ChainIDFromUInt64(999_999))
	require.NoError(t, err, "unknown chain must not error here; config.Check gates this")
	require.Nil(t, ds)
}

func TestNewDependencySetFromCLI_RegistryFallback_RoundTripsViaJSON(t *testing.T) {
	ctx := newDepsetCtx(t)
	ds, err := NewDependencySetFromCLI(ctx, eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	require.NotNil(t, ds)

	raw, err := json.Marshal(ds)
	require.NoError(t, err)

	var roundTripped depset.StaticConfigDependencySet
	require.NoError(t, json.Unmarshal(raw, &roundTripped))
	require.Equal(t, ds.Chains(), roundTripped.Chains())
}
