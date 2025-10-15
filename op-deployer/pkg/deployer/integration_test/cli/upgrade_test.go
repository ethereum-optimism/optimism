package cli

import (
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v2_0_0"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestCLIUpgrade tests the upgrade CLI command for each standard opcm release
// - forks sepolia at the block immediately after the opcm deployment block
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
		forkBlock   uint64 // one block number past the opcm deployment block
	}{
		{
			contractTag: standard.ContractsV200Tag,
			version:     "v2.0.0",
			forkBlock:   7792843,
		},
		//{
		//contractTag: standard.ContractsV300Tag,
		//version:     "v3.0.0",
		//forkBlock:   7853303,
		//},
		{
			contractTag: standard.ContractsV400Tag,
			version:     "v4.0.0",
			forkBlock:   8577263,
		},
		{
			contractTag: standard.ContractsV410Tag,
			version:     "v4.1.0",
			forkBlock:   9165154,
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

			configFile := filepath.Join(workDir, "upgrade_config_"+tc.version+".json")
			outputFile := filepath.Join(workDir, "upgrade_output_"+tc.version+".json")

			configData, err := json.MarshalIndent(testConfig, "", "  ")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(configFile, configData, 0644))

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
			require.True(t, strings.HasPrefix(dataHex, "ff2dd5a1"),
				"calldata should have opcm.upgrade fcn selector ff2dd5a1, got: %s", dataHex[:8])

		})
	}
}
