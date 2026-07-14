package rustengine

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// opcmArtifactsRel points at the compiled OPCMExample synthetic Family-B script (see
// testdata/scripts/OPCMExample.s.sol).
const opcmArtifactsRel = "testdata/test-artifacts"

// OPCMExampleInput / OPCMExampleOutput are the Go input/output structs for OPCMExample. Their
// exported fields map to the Solidity getters/setter (owner(), blob(), result()) exactly as the
// real op-deployer OPCM input/output structs (e.g. manage.ScriptInput / InteropMigrationOutput).
type OPCMExampleInput struct {
	Owner common.Address
	Blob  []byte
}

type OPCMExampleOutput struct {
	Result common.Address
}

// TestRustEngineOPCMParity is the OPCM-milestone gate for the Rust script engine (op-geth
// decoupling spike). It drives the OPCM RunScriptSingle input/output-precompile path
// (op-deployer/pkg/deployer/opcm.RunScriptSingle — the production RunScript* Go code) through both
// the in-process Go host and the out-of-process Rust engine, and asserts byte-parity of BOTH the
// populated output struct and the state dump.
//
// Leg B uses the unidirectional design (design §4): the Go side snapshots I's getters and installs
// them as an input precompile in the Rust engine; the Rust engine captures O's set() calls, which
// the Go side then replays through the real WithFieldSetter precompile. No Rust->Go callback.
func TestRustEngineOPCMParity(t *testing.T) {
	bin := buildEngine(t)
	art, err := filepath.Abs(opcmArtifactsRel)
	require.NoError(t, err)
	logw := testWriter{t}

	input := OPCMExampleInput{
		Owner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Blob:  []byte("hello-opcm-parity"),
	}
	target := common.HexToAddress("0x0000000000000000000000000000000000C0FFEE")

	goHost := func(t *testing.T) *script.Host {
		gh := script.NewHost(testlog.Logger(t, log.LevelError),
			foundry.OpenArtifactsDir(opcmArtifactsRel), nil, script.DefaultContext)
		require.NoError(t, gh.EnableCheats())
		return gh
	}

	// RunScriptSingle: exercises input getter-snapshot + output setter-capture + WithFieldSetter
	// replay (the full unidirectional design). Leg A is the real opcm.RunScriptSingle.
	t.Run("single", func(t *testing.T) {
		gh := goHost(t)
		goOutput, err := opcm.RunScriptSingle[OPCMExampleInput, OPCMExampleOutput](
			gh, input, "OPCMExample.s.sol", "OPCMExample")
		require.NoError(t, err, "go RunScriptSingle")
		goDump, err := gh.StateDump()
		require.NoError(t, err)

		re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337}, logw)
		require.NoError(t, err)
		defer re.Close()
		rustOutput, err := RunScriptSingle[OPCMExampleInput, OPCMExampleOutput](
			re, input, "OPCMExample.s.sol", "OPCMExample", script.DefaultContext.Origin)
		require.NoError(t, err, "rust RunScriptSingle")
		rustDump, err := re.StateDump()
		require.NoError(t, err)

		require.Equal(t, target, goOutput.Result, "sanity: go output populated via WithFieldSetter")
		require.Equal(t, goOutput, rustOutput, "output struct parity")
		require.Contains(t, goDump.Accounts, target, "go dump must contain the mutated TARGET account")
		requireAllocsEqual(t, "opcm-single", goDump, rustDump)
		t.Logf("RunScriptSingle parity: output.Result=%s, %d dump accounts, byte-identical",
			rustOutput.Result, len(rustDump.Accounts))
	})

	// RunScriptVoid: input-only path (mirrors UpgradeOPChain / SetDisputeGameImpl). Leg A is the
	// real opcm.RunScriptVoid.
	t.Run("void", func(t *testing.T) {
		gh := goHost(t)
		require.NoError(t, opcm.RunScriptVoid[OPCMExampleInput](
			gh, input, "OPCMExample.s.sol", "OPCMExampleVoid"), "go RunScriptVoid")
		goDump, err := gh.StateDump()
		require.NoError(t, err)

		re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337}, logw)
		require.NoError(t, err)
		defer re.Close()
		require.NoError(t, RunScriptVoid[OPCMExampleInput](
			re, input, "OPCMExample.s.sol", "OPCMExampleVoid", script.DefaultContext.Origin),
			"rust RunScriptVoid")
		rustDump, err := re.StateDump()
		require.NoError(t, err)

		require.Contains(t, goDump.Accounts, target, "go dump must contain the mutated TARGET account")
		requireAllocsEqual(t, "opcm-void", goDump, rustDump)
		t.Logf("RunScriptVoid parity: %d dump accounts, byte-identical", len(rustDump.Accounts))
	})
}
