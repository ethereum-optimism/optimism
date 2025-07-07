package integration_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// Helper function for waiting for regular Ethereum transactions in tests
func waitForTransaction(t *testing.T, ctx context.Context, ethClient *ethclient.Client, txHash common.Hash) *types.Receipt {
	for {
		receipt, err := ethClient.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt
		}
		if !errors.Is(err, ethereum.NotFound) {
			require.NoError(t, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func deploySafeContracts(t *testing.T, rpcUrl string, privateKey string) (common.Address, common.Address) {
	// Get the directory where this test file is located
	_, testFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "Failed to get test file location")

	testDir := filepath.Dir(testFile) // op-service/gnosis/integration_test

	// Navigate to the correct paths from test file location
	contractsDir := filepath.Join(testDir, "..", "contracts")
	sharedLibDir := filepath.Join(testDir, "..", "..", "..", "packages", "contracts-bedrock", "lib")

	// Verify paths exist
	contractsDirAbs, err := filepath.Abs(contractsDir)
	require.NoError(t, err)
	sharedLibDirAbs, err := filepath.Abs(sharedLibDir)
	require.NoError(t, err)

	t.Logf("Contracts directory: %s", contractsDirAbs)
	t.Logf("Shared lib directory: %s", sharedLibDirAbs)

	// Check if shared dependencies are available
	if _, err := os.Stat(sharedLibDirAbs); os.IsNotExist(err) {
		t.Fatalf("Shared contracts dependencies not found at: %s", sharedLibDirAbs)
	}

	// Check if forge-std is available in shared lib
	forgeStdDir := filepath.Join(sharedLibDirAbs, "forge-std")
	if _, err := os.Stat(forgeStdDir); os.IsNotExist(err) {
		t.Fatalf("forge-std not found in shared lib at: %s", forgeStdDir)
	}

	// Check if safe-contracts is available in shared lib
	safeContractsDir := filepath.Join(sharedLibDirAbs, "safe-contracts")
	if _, err := os.Stat(safeContractsDir); os.IsNotExist(err) {
		t.Fatalf("safe-contracts not found in shared lib at: %s.\n Try running 'forge install' in packages/contracts-bedrock first.", safeContractsDir)
	}

	// Change to contracts directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore original directory: %v", err)
		}
	}()

	err = os.Chdir(contractsDirAbs)
	require.NoError(t, err)

	t.Log("Building contracts...")
	buildCmd := exec.Command("forge", "build")
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build contracts: %s", string(output))
	t.Log("Contracts built successfully")

	// Run forge script to deploy contracts
	cmd := exec.Command("forge", "script", "script/DeploySafe.s.sol:DeploySafe",
		"--rpc-url", rpcUrl,
		"--broadcast",
		"--private-key", privateKey)

	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to deploy Safe contracts: %s", string(output))

	// Parse the output to extract deployed addresses
	outputStr := string(output)
	safeAddrStr := extractAddressFromOutput(outputStr, "Safe instance deployed at:")
	require.NotEmpty(t, safeAddrStr, "Failed to extract Safe address from deployment output")

	// Extract TestDelegateCall address
	testContractAddrStr := extractAddressFromOutput(outputStr, "TestDelegateCall deployed at:")
	require.NotEmpty(t, testContractAddrStr, "Failed to extract TestDelegateCall address from deployment output")

	return common.HexToAddress(safeAddrStr), common.HexToAddress(testContractAddrStr)
}

// extractAddressFromOutput extracts an address from forge script output
func extractAddressFromOutput(output, prefix string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, prefix) {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	return ""
}
