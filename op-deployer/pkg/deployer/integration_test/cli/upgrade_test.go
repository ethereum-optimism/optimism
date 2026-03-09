package cli

import (
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/opcmregistry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v2_0_0"
	v6_0_0 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v6_0_0"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestCLIUpgrade tests the upgrade CLI command for each standard opcm release
// - forks sepolia at a block before op-sepolia was upgraded
// - runs the upgrade CLI command using op-sepolia values to simulate its upgrade
func TestCLIUpgrade(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelDebug)

	// op-sepolia values
	l1ProxyAdminOwner := common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2")
	systemConfigProxy := common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538")
	proxyAdminImpl := common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc")

	testCases := []struct {
		contractTag string
		version     string
		forkBlock   uint64
	}{
		{
			contractTag: standard.ContractsV200Tag,
			version:     "v2.0.0",
			forkBlock:   7792843, // one block past the opcm deployment block
		},
		{
			contractTag: standard.ContractsV300Tag,
			version:     "v3.0.0",
			forkBlock:   8092886, // one block before op-sepolia was upgraded
		},
		{
			contractTag: standard.ContractsV400Tag,
			version:     "v4.0.0",
			forkBlock:   8577263, // one block past the opcm deployment block
		},
		{
			contractTag: standard.ContractsV410Tag,
			version:     "v4.1.0",
			forkBlock:   9165154, // one block past the opcm deployment block
		},
		{
			contractTag: standard.ContractsV500Tag,
			version:     "v5.0.0",
			forkBlock:   9629972, // one block past the opcm deployment block
		},
		{
			contractTag: standard.ContractsV600Tag,
			version:     "v6.0.0-rc.2",
			forkBlock:   10101510, // one block past the opcm deployment block
		},
	}

	for _, tc := range testCases {
		t.Run(tc.contractTag, func(t *testing.T) {
			forkedL1, stopL1, err := devnet.NewForkedSepoliaFromBlock(lgr, tc.forkBlock)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, stopL1())
			})

			runner := NewCLITestRunnerWithNetwork(t, WithL1RPC(forkedL1.RPCUrl()))
			workDir := runner.GetWorkDir()

			opcm, err := standard.OPCMImplAddressFor(11155111, tc.contractTag)
			require.NoError(t, err)

			versionStr := strings.TrimPrefix(tc.version, "v") // Remove "v" prefix for parsing
			version, err := opcmregistry.ParseSemver(versionStr)
			require.NoError(t, err, "failed to parse version %s", versionStr)

			v6Semver := opcmregistry.Semver{Major: 6, Minor: 0, Patch: 0}
			var configData []byte
			if version.Compare(v6Semver) >= 0 {
				// v6.0.0+ uses a different input structure
				testConfig := v6_0_0.UpgradeOPChainInput{
					Prank: l1ProxyAdminOwner,
					Opcm:  opcm,
					EncodedChainConfigs: []v6_0_0.OPChainConfig{
						{
							SystemConfigProxy:  systemConfigProxy,
							CannonPrestate:     common.HexToHash("0x0abc"),
							CannonKonaPrestate: common.HexToHash("0x0def"),
						},
					},
				}
				configData, err = json.MarshalIndent(testConfig, "", "  ")
			} else {
				// Older versions use v2_0_0 structure
				testConfig := v2_0_0.UpgradeOPChainInput{
					Prank: l1ProxyAdminOwner,
					Opcm:  opcm,
					EncodedChainConfigs: []v2_0_0.OPChainConfig{
						{
							SystemConfigProxy: systemConfigProxy,
							ProxyAdmin:        proxyAdminImpl,
							AbsolutePrestate:  common.HexToHash("0x0abc"),
						},
					},
				}
				configData, err = json.MarshalIndent(testConfig, "", "  ")
			}
			require.NoError(t, err)

			configFile := filepath.Join(workDir, "upgrade_config_"+tc.version+".json")
			outputFile := filepath.Join(workDir, "upgrade_output_"+tc.version+".json")
			require.NoError(t, os.WriteFile(configFile, configData, 0o644))

			// Run full cli command to write calldata to outfile
			output := runner.ExpectSuccess(t, []string{
				"upgrade", tc.version,
				"--config", configFile,
				"--l1-rpc-url", runner.l1RPC,
				"--outfile", outputFile,
			}, nil)

			t.Logf("Command output (logs):\n%s", output)

			// Read and parse calldata from outfile
			require.FileExists(t, outputFile)
			data, err := os.ReadFile(outputFile)
			require.NoError(t, err)

			var dump []broadcaster.CalldataDump
			require.NoError(t, json.Unmarshal(data, &dump))

			t.Logf("Upgrade %s generated calldata: %s", tc.version, string(data))

			// Verify the calldata
			require.Len(t, dump, 1)
			require.Equal(t, l1ProxyAdminOwner.Hex(), dump[0].To.Hex())
			dataHex := hex.EncodeToString(dump[0].Data)

			// v6.0.0+ uses a different function signature: upgrade((address,bytes32,bytes32)[])
			// Older versions use: upgrade((address,address,bytes32)[])
			var expectedSelector string
			if version.Compare(v6Semver) >= 0 {
				expectedSelector = "cbeda5a7" // upgrade((address,bytes32,bytes32)[])
			} else {
				expectedSelector = "ff2dd5a1" // upgrade((address,address,bytes32)[])
			}
			require.True(t, strings.HasPrefix(dataHex, expectedSelector),
				"calldata should have opcm.upgrade fcn selector %s, got: %s", expectedSelector, dataHex[:8])
		})
	}
}
