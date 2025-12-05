package cli

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/verify"
	"github.com/stretchr/testify/require"
)

// TestCLIVerify consolidates all verification tests to reuse deployed contracts
func TestCLIVerify(t *testing.T) {
	l1ChainID := uint64(31337)
	l1ChainIDBig := big.NewInt(int64(l1ChainID))

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	superchainProxyAdminOwner := shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainIDBig))
	protocolVersionsOwner := shared.AddrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainIDBig))
	guardian := shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
	challenger := shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

	// Shared setup: deploy contracts ONCE
	runner := NewCLITestRunnerWithNetwork(t)
	workDir := runner.GetWorkDir()
	mockServer := setupMockBlockscout(t)

	superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain.json")
	implsOutputFile := filepath.Join(workDir, "bootstrap_implementations.json")

	// Deploy superchain contracts (once for all subtests)
	t.Logf("Deploying superchain contracts (shared setup)...")
	runner.ExpectSuccessWithNetwork(t, []string{
		"bootstrap", "superchain",
		"--outfile", superchainOutputFile,
		"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
		"--protocol-versions-owner", protocolVersionsOwner.Hex(),
		"--guardian", guardian.Hex(),
	}, nil)

	var superchainOutput opcm.DeploySuperchainOutput
	data, err := os.ReadFile(superchainOutputFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, &superchainOutput)
	require.NoError(t, err)
	require.NoError(t, addresses.CheckNoZeroAddresses(superchainOutput))

	// Deploy implementations contracts (once for relevant subtests)
	t.Logf("Deploying implementations contracts (shared setup)...")
	runner.ExpectSuccessWithNetwork(t, []string{
		"bootstrap", "implementations",
		"--outfile", implsOutputFile,
		"--mips-version", strconv.Itoa(int(standard.MIPSVersion)),
		"--protocol-versions-proxy", superchainOutput.ProtocolVersionsProxy.Hex(),
		"--superchain-config-proxy", superchainOutput.SuperchainConfigProxy.Hex(),
		"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
		"--superchain-proxy-admin", superchainOutput.SuperchainProxyAdmin.Hex(),
		"--challenger", challenger.Hex(),
	}, nil)

	var implsOutput opcm.DeployImplementationsOutput
	data, err = os.ReadFile(implsOutputFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, &implsOutput)
	require.NoError(t, err)

	// Now run all verification tests using the same deployed contracts
	t.Run("manual verify superchain", func(t *testing.T) {
		output := runner.ExpectSuccess(t, []string{
			"verify",
			"--l1-rpc-url", runner.l1RPC,
			"--input-file", superchainOutputFile,
			"--verifier", "blockscout",
			"--verifier-url", mockServer + "/api",
			"--artifacts-locator", "embedded",
		}, nil)

		assertVerificationSuccess(t, output)
	})

	t.Run("manual verify implementations", func(t *testing.T) {
		output := runner.ExpectSuccess(t, []string{
			"verify",
			"--l1-rpc-url", runner.l1RPC,
			"--input-file", implsOutputFile,
			"--verifier", "blockscout",
			"--verifier-url", mockServer + "/api",
			"--artifacts-locator", "embedded",
		}, nil)

		assertVerificationSuccess(t, output)
	})

	t.Run("verify single contract", func(t *testing.T) {
		output := runner.ExpectSuccess(t, []string{
			"verify",
			"--l1-rpc-url", runner.l1RPC,
			"--input-file", superchainOutputFile,
			"--contract-name", "superchainConfigProxyAddress",
			"--verifier", "blockscout",
			"--verifier-url", mockServer + "/api",
			"--artifacts-locator", "embedded",
		}, nil)

		require.Contains(t, output, "Contract already verified")
		require.Contains(t, output, "superchainConfigProxyAddress")
	})

	t.Run("auto-verify with bootstrap", func(t *testing.T) {
		// Test the --verify flag by deploying a fresh set to a new output file
		autoVerifyOutputFile := filepath.Join(workDir, "bootstrap_superchain_autoverify.json")

		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", autoVerifyOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--verify",
			"--verifier", "blockscout",
			"--verifier-url", mockServer + "/api",
		}, nil)

		require.Contains(t, output, "Starting automatic contract verification")
		require.Contains(t, output, "Verification Summary")
		require.Contains(t, output, "verified=0")
		require.Contains(t, output, "skipped=5")
		require.Contains(t, output, "failed=0")
	})

	t.Run("state file contract name mapping", func(t *testing.T) {
		// Test the artifact path mapping for underscore contract names

		testCases := map[string]string{
			"superchain_superchain_config_proxy": "Proxy.sol/Proxy.json",
			"superchain_protocol_versions_proxy": "Proxy.sol/Proxy.json",
			"superchain_superchain_config_impl":  "SuperchainConfig.sol/SuperchainConfig.json",
			"implementations_opcm_impl":          "OPContractsManager.sol/OPContractsManager.json",
			"regular_contract_name":              "RegularContractName.sol/RegularContractName.json",
		}

		for contractName, expectedPath := range testCases {
			actualPath := verify.GetArtifactPath(contractName)
			require.Equal(t, expectedPath, actualPath,
				"contract name %q should map to %q, got %q", contractName, expectedPath, actualPath)
		}
	})
}

func setupMockBlockscout(t *testing.T) string {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasPrefix(r.URL.Path, "/api/v2/smart-contracts/") {
			response := map[string]interface{}{
				"message":                         "OK",
				"is_verified":                     true,
				"is_fully_verified":               true,
				"is_partially_verified":           false,
				"is_verified_via_eth_bytecode_db": false,
				"is_verified_via_sourcify":        false,
			}
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		var action string
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse form", http.StatusBadRequest)
				return
			}
			action = r.FormValue("module")
			if action == "contract" {
				action = r.FormValue("action")
			}
		} else {
			action = r.URL.Query().Get("action")
		}

		switch action {
		case "getabi":
			response := map[string]interface{}{
				"status":  "0",
				"message": "NOTOK",
				"result":  "Contract source code not verified",
			}
			_ = json.NewEncoder(w).Encode(response)

		case "verifysourcecode":
			response := map[string]interface{}{
				"status":  "1",
				"message": "OK",
				"result":  "verification_guid_123",
			}
			_ = json.NewEncoder(w).Encode(response)

		case "checkverifystatus":
			response := map[string]interface{}{
				"status":  "1",
				"message": "OK",
				"result":  "Pass - Verified",
			}
			_ = json.NewEncoder(w).Encode(response)

		case "getcontractcreation":
			// Forge uses this with --guess-constructor-args
			// Return success with empty array so forge proceeds without constructor args
			response := map[string]interface{}{
				"status":  "1",
				"message": "OK",
				"result":  []interface{}{},
			}
			_ = json.NewEncoder(w).Encode(response)

		default:
			if r.Method == http.MethodPost {
				response := map[string]interface{}{
					"status":  "1",
					"message": "OK",
					"result":  "verification_guid_123",
				}
				_ = json.NewEncoder(w).Encode(response)
			} else {
				// Return JSON error instead of plain text
				response := map[string]interface{}{
					"status":  "0",
					"message": "NOTOK",
					"result":  "Unknown action",
				}
				_ = json.NewEncoder(w).Encode(response)
			}
		}
	}))

	t.Cleanup(server.Close)
	return server.URL
}

func parseVerifyOutput(output string) (verified, skipped, partiallyVerified, failed int, err error) {
	re := regexp.MustCompile(`(?:Results|Verification complete).*verified=(\d+).*skipped=(\d+).*partially_verified=(\d+).*failed=(\d+)`)
	matches := re.FindStringSubmatch(output)

	if len(matches) != 5 {
		return 0, 0, 0, 0, fmt.Errorf("could not parse verification output: expected 'Results' or 'Verification complete' line with verified, skipped, partially_verified, and failed counts")
	}

	verified, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse numVerified: %w", err)
	}

	skipped, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse numSkipped: %w", err)
	}

	partiallyVerified, err = strconv.Atoi(matches[3])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse numPartiallyVerified: %w", err)
	}

	failed, err = strconv.Atoi(matches[4])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse numFailed: %w", err)
	}

	return verified, skipped, partiallyVerified, failed, nil
}

func assertVerificationSuccess(t *testing.T, output string) {
	verified, skipped, partiallyVerified, failed, err := parseVerifyOutput(output)
	require.NoError(t, err, "Failed to parse verification output")
	require.GreaterOrEqual(t, verified+skipped+partiallyVerified, 1, "At least one contract should be verified, skipped, or partially verified")
	require.Equal(t, 0, failed, "No contracts should fail verification")

	lowerOutput := strings.ToLower(output)
	require.True(t, strings.Contains(lowerOutput, "successfully") || strings.Contains(lowerOutput, "complete"), "Output should indicate completion")
}
