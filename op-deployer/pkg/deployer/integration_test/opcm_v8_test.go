package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/opcmregistry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"
)

func readOPCMVersion(t *testing.T, host *script.Host, addr common.Address, method string) string {
	t.Helper()
	fn := w3.MustNewFunc(method+"()", "string")
	data, err := fn.EncodeArgs()
	require.NoError(t, err)
	result, err := (&shared.HostCaller{Host: host}).Call(addr, data)
	require.NoError(t, err)
	var version string
	require.NoError(t, fn.DecodeReturns(result, &version))
	return version
}

// TODO(#22906): Use released v8 artifacts, then remove this helper once RunPastUpgrades covers v8.
// stageOPCMV8 applies the development v8 manager using the current contract implementations.
func stageOPCMV8(t *testing.T, host *script.Host, impls opcm.DeployImplementationsOutput, input embedded.UpgradeOPChainInput) {
	t.Helper()
	lastVersion, err := opcmregistry.ParseSemver(readOPCMVersion(t, host, input.UpgradeInputV2.SystemConfig, "lastUsedOPCMVersion"))
	require.NoError(t, err)
	if lastVersion.Major >= 8 {
		return
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)
	root, err := op_service.FindMonorepoRoot(cwd)
	require.NoError(t, err)
	contractsDir := filepath.Join(root, "packages", "contracts-bedrock")
	artifactPath := os.Getenv("OPCM_V8_ARTIFACT")
	if artifactPath == "" {
		cmd := exec.CommandContext(t.Context(), "just", "prepare-opcm-v8-artifact")
		cmd.Dir = contractsDir
		var stderr strings.Builder
		cmd.Stderr = &stderr
		output, err := cmd.Output()
		require.NoError(t, err, "build v8 OPCM: %s", stderr.String())
		artifactPath = strings.TrimSpace(string(output))
	}
	if !filepath.IsAbs(artifactPath) {
		artifactPath = filepath.Join(contractsDir, artifactPath)
	}
	artifact, err := foundry.ReadArtifact(artifactPath)
	require.NoError(t, err)
	args, err := artifact.ABI.Pack("", impls.OpcmStandardValidator, impls.OpcmMigrator, impls.OpcmUtils)
	require.NoError(t, err)
	v8, err := host.Create(host.TxOrigin(), append(artifact.Bytecode.Object, args...))
	require.NoError(t, err)
	v8Version := readOPCMVersion(t, host, v8, "version")
	version, err := opcmregistry.ParseSemver(v8Version)
	require.NoError(t, err)
	require.EqualValues(t, 8, version.Major, "expected a v8 OPCM artifact")

	input.Opcm = v8
	encoded, err := json.Marshal(input)
	require.NoError(t, err)
	require.NoError(t, embedded.DefaultUpgrader.Upgrade(host, encoded), "v8 upgrade should succeed")
	require.Equal(t, v8Version, readOPCMVersion(t, host, input.UpgradeInputV2.SystemConfig, "lastUsedOPCMVersion"))
}
