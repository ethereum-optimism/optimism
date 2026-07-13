package interopgen

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// provisionEngineBinary makes the op-script-engine binary resolvable for DeployWithEngine:
// prefer a CI-provided pre-built binary (Go executors without cargo), else cargo-build the
// debug binary, else skip (a machine with neither cannot run the Rust engine).
func provisionEngineBinary(t *testing.T) {
	if _, ok := rustengine.PrebuiltEngineBinary(); ok {
		return
	}
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skipf("no pre-built op-script-engine (%s unset) and cargo unavailable", rustengine.EngineBinaryPathEnv)
	}
	cmd := exec.Command("cargo", "build", "-p", "op-script-engine", "--bin", "op-script-engine")
	cmd.Dir = "../../rust"
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "cargo build -p op-script-engine failed:\n%s", out)
	bin, err := filepath.Abs("../../rust/target/debug/op-script-engine")
	require.NoError(t, err)
	t.Setenv(rustengine.EngineBinaryPathEnv, bin)
}

// TestRustEngineInteropgenParity runs the full interopgen world deployment on both script
// engines and requires byte-identical results: the WorldDeployment (all contract addresses)
// and the WorldOutput (L1 genesis, and every L2's genesis + rollup config).
func TestRustEngineInteropgenParity(t *testing.T) {
	provisionEngineBinary(t)

	logger := testlog.Logger(t, log.LevelInfo)
	fa := foundry.OpenArtifactsDir("../../packages/contracts-bedrock/forge-artifacts")
	srcFS := foundry.NewSourceMapFS(os.DirFS("../../packages/contracts-bedrock"))

	// A fresh, identical world config per leg (Deploy must not depend on shared state).
	buildCfg := func() *WorldConfig {
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

	goDep, goOut, err := DeployWithEngine(logger, fa, srcFS, buildCfg(), env.ScriptEngineGo)
	require.NoError(t, err, "go-engine deploy failed")

	rsDep, rsOut, err := DeployWithEngine(logger, fa, srcFS, buildCfg(), env.ScriptEngineRust)
	require.NoError(t, err, "rust-engine deploy failed")

	requireJSONEq(t, goDep, rsDep, "WorldDeployment")
	requireJSONEq(t, goOut.L1.Genesis, rsOut.L1.Genesis, "L1 genesis")
	require.Equal(t, len(goOut.L2s), len(rsOut.L2s), "L2 output count")
	for id, goL2 := range goOut.L2s {
		rsL2, ok := rsOut.L2s[id]
		require.True(t, ok, "missing L2 output %s on rust engine", id)
		requireJSONEq(t, goL2.Genesis, rsL2.Genesis, "L2 %s genesis", id)
		requireJSONEq(t, goL2.RollupCfg, rsL2.RollupCfg, "L2 %s rollup config", id)
		t.Logf("L2 %s parity: %d genesis accounts, byte-identical across Go host and Rust engine", id, len(goL2.Genesis.Alloc))
	}
	t.Logf("L1 parity: %d genesis accounts, byte-identical across Go host and Rust engine", len(goOut.L1.Genesis.Alloc))
}

func requireJSONEq(t *testing.T, want, got any, format string, args ...any) {
	t.Helper()
	wantJSON, err := json.Marshal(want)
	require.NoError(t, err)
	gotJSON, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, string(wantJSON), string(gotJSON), append([]any{format}, args...)...)
}
