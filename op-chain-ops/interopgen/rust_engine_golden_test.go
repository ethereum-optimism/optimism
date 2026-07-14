package interopgen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// buildInteropWorldConfig builds the fixed interop world config the parity/golden tests deploy. It is
// deterministic (pinned chain IDs + genesis timestamp + test mnemonic), so its genesis output can be
// hash-pinned. A faithful copy of TestRustEngineInteropgenParity's inline config.
func buildInteropWorldConfig(t *testing.T, logger log.Logger) *WorldConfig {
	t.Helper()
	rec := InteropDevRecipe{
		L1ChainID:        900100,
		L2s:              []InteropDevL2Recipe{{ChainID: 900200}, {ChainID: 900201}},
		GenesisTimestamp: 1234567,
	}
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	cfg, err := rec.Build(hd)
	require.NoError(t, err)
	require.NoError(t, cfg.Check(logger))
	return cfg
}

// TestRustEngineInteropgenGolden goldenizes TestRustEngineInteropgenParity: it runs the full interopgen
// world deployment through the Rust engine and pins the WorldDeployment (all contract addresses) and
// every rollup config raw, plus the L1 and per-L2 genesis dumps by SHA-256 of their canonical
// json.Marshal bytes (the dumps are multi-MB, so committing them raw would bloat the repo).
func TestRustEngineInteropgenGolden(t *testing.T) {
	provisionEngineBinary(t)

	logger := testlog.Logger(t, log.LevelInfo)
	fa := foundry.OpenArtifactsDir("../../packages/contracts-bedrock/forge-artifacts")

	rsDep, rsOut, err := Deploy(logger, fa, buildInteropWorldConfig(t, logger))
	require.NoError(t, err, "rust-engine deploy failed")

	// WorldDeployment: all contract addresses, committed raw.
	requireJSONMatchesGolden(t, "world_deployment.json", rsDep)

	// L1 genesis: hash-pinned. Non-vacuity guard before the hash compare.
	require.NotEmpty(t, rsOut.L1.Genesis.Alloc, "L1 genesis alloc must be non-empty")
	l1Canonical, err := json.Marshal(rsOut.L1.Genesis)
	require.NoError(t, err)
	requireHashMatchesGolden(t, "genesis.l1.sha256", l1Canonical)

	// Each L2: rollup config raw + genesis hash-pinned.
	require.Len(t, rsOut.L2s, 2, "expected two L2 outputs")
	for id, l2 := range rsOut.L2s {
		require.NotEmpty(t, l2.Genesis.Alloc, "L2 %s genesis alloc must be non-empty", id)
		requireJSONMatchesGolden(t, "rollup-l2-"+id+".json", l2.RollupCfg)
		l2Canonical, err := json.Marshal(l2.Genesis)
		require.NoError(t, err)
		requireHashMatchesGolden(t, "genesis.l2-"+id+".sha256", l2Canonical)
		t.Logf("L2 %s golden: %d genesis accounts pinned", id, len(l2.Genesis.Alloc))
	}
}
