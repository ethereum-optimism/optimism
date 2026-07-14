package scriptbackend

import (
	"math/big"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
)

// forgeArtifactsRel is the on-disk forge-artifacts directory relative to this package.
const forgeArtifactsRel = "../../../../packages/contracts-bedrock/forge-artifacts"

// provisionEngineBin resolves the op-script-engine binary the same way the other gates do: a
// pre-built binary named by RUST_BINARY_PATH_OP_SCRIPT_ENGINE (CI), else a local cargo build. Under
// REQUIRE_RUST_ENGINE (CI) it fails when neither is available so a provisioning break can't skip the
// gate silently; local dev skips.
func provisionEngineBin(t *testing.T) string {
	if p, ok := rustengine.PrebuiltEngineBinary(); ok {
		return p
	}
	if _, err := exec.LookPath("cargo"); err != nil {
		msg := "no pre-built op-script-engine (RUST_BINARY_PATH_OP_SCRIPT_ENGINE) and cargo unavailable"
		if rustengine.RequireEngine() {
			t.Fatal(msg + " (" + rustengine.RequireEngineEnv + " is set)")
		}
		t.Skip(msg)
	}
	root, err := filepath.Abs("../../../../rust")
	require.NoError(t, err)
	cmd := exec.Command("cargo", "build", "-p", "op-script-engine", "--bin", "op-script-engine")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cargo build op-script-engine failed: %v\n%s", err, out)
	}
	return filepath.Join(root, "target", "debug", "op-script-engine")
}

type engineLogWriter struct{ t *testing.T }

func (w engineLogWriter) Write(p []byte) (int, error) {
	w.t.Logf("[engine] %s", strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// TestNewEngineScripts is the production-path replacement for the deleted opcm.NewScripts host smoke
// tests (opcm/scripts_test.go, superchain_test.go, mips_test.go, ...): it builds the OPCM Scripts
// bundle on the Rust engine via NewEngineScripts (the same construction apply.go uses), asserts every
// typed script loaded (which fails if a Go I/O struct diverges from the Solidity ABI), and runs the
// two dependency-free scripts end-to-end to certify they execute on the engine.
func TestNewEngineScripts(t *testing.T) {
	bin := provisionEngineBin(t)

	artifactsPath, err := filepath.Abs(forgeArtifactsRel)
	require.NoError(t, err)
	fa := foundry.OpenArtifactsDir(artifactsPath)

	eng, err := rustengine.Spawn(bin, rustengine.SpawnOpts{
		ArtifactsDir:    artifactsPath,
		ChainID:         1337,
		Create2Deployer: true,
	}, engineLogWriter{t})
	require.NoError(t, err)
	defer eng.Close()

	origin := func() common.Address { return common.BigToAddress(big.NewInt(6)) }
	scripts, err := NewEngineScripts(eng, fa, origin)
	require.NoError(t, err)

	// The full bundle must load — an ABI/Go-type mismatch on any script would fail here.
	require.NotNil(t, scripts.DeployImplementations)
	require.NotNil(t, scripts.DeploySuperchain)
	require.NotNil(t, scripts.DeployAlphabetVM)
	require.NotNil(t, scripts.DeployAltDA)
	require.NotNil(t, scripts.DeployDisputeGame)
	require.NotNil(t, scripts.DeployMIPS)
	require.NotNil(t, scripts.DeployOPChain)

	// DeploySuperchain runs without external dependencies; certify it executes on the engine.
	superOut, err := scripts.DeploySuperchain.Run(opcm.DeploySuperchainInput{
		Guardian:                  common.BigToAddress(big.NewInt(1)),
		SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(3)),
		Paused:                    true,
	})
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, superOut.SuperchainConfigImpl)

	// DeployMIPS runs against a dummy PreimageOracle immutable (never dereferenced at deploy time).
	mipsOut, err := scripts.DeployMIPS.Run(opcm.DeployMIPSInput{
		PreimageOracle: common.Address{'P'},
		MipsVersion:    big.NewInt(int64(standard.MIPSVersion)),
	})
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, mipsOut.MipsSingleton)

	// DeployDisputeGame stores the VM (IBigStepper) as an immutable; run it against the MIPS singleton
	// just deployed. This is the migration of the deleted opcm/dispute_game_test.go, which built the VM
	// by hand on the Go host — here the production DeployMIPS engine script supplies it, in the same
	// engine (the deployed singleton persists across script runs).
	dgOut, err := scripts.DeployDisputeGame.Run(opcm.DeployDisputeGameInput{
		Release:                  "dev",
		VmAddress:                mipsOut.MipsSingleton,
		GameKind:                 "PermissionedDisputeGame",
		GameType:                 1,
		AbsolutePrestate:         common.Hash{'A'},
		MaxGameDepth:             big.NewInt(int64(standard.DisputeMaxGameDepth)),
		SplitDepth:               big.NewInt(int64(standard.DisputeSplitDepth)),
		ClockExtension:           standard.DisputeClockExtension,
		MaxClockDuration:         standard.DisputeMaxClockDuration,
		DelayedWethProxy:         common.Address{'D'},
		AnchorStateRegistryProxy: common.Address{'A'},
		L2ChainId:                big.NewInt(69),
		Proposer:                 common.Address{'P'},
		Challenger:               common.Address{'C'},
	})
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, dgOut.DisputeGameImpl)
	code, err := eng.GetCode(dgOut.DisputeGameImpl)
	require.NoError(t, err)
	require.NotEmpty(t, code, "deployed dispute game implementation must have code")
}
