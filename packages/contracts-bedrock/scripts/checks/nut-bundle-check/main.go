package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	DEPOSITOR_ACCOUNT    = "0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001"
	PROXY_ADMIN          = "0x4200000000000000000000000000000000000018"
	CONDITIONAL_DEPLOYER = "0x420000000000000000000000000000000000002C"
	ZERO_ADDRESS         = "0x0000000000000000000000000000000000000000"
)

type BundleMetadata struct {
	Version string `json:"version"`
}

type NetworkUpgradeTxn struct {
	Data     string `json:"data"`
	From     string `json:"from"`
	GasLimit uint64 `json:"gasLimit"`
	Intent   string `json:"intent"`
	To       string `json:"to"`
}

type UpgradeBundle struct {
	Metadata     BundleMetadata      `json:"metadata"`
	Transactions []NetworkUpgradeTxn `json:"transactions"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌  %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Construct bundle path dynamically
	bundlePath := "snapshots/upgrades/current-upgrade-bundle.json"
	fmt.Printf("Bundle path: %s\n", bundlePath)

	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return fmt.Errorf("bundle file not found at %s", bundlePath)
	}

	// Read the committed bundle
	committedData, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to read bundle file: %w", err)
	}

	var committedBundle UpgradeBundle
	if err := json.Unmarshal(committedData, &committedBundle); err != nil {
		return fmt.Errorf("failed to parse bundle JSON: %w", err)
	}

	fmt.Printf("Found %d transactions in bundle (version: %s)\n",
		len(committedBundle.Transactions), committedBundle.Metadata.Version)

	// 1. Bundle Structure Validation
	fmt.Println("\nValidating bundle structure...")
	if err := validateBundleStructure(&committedBundle); err != nil {
		return fmt.Errorf("bundle structure validation failed: %w", err)
	}
	fmt.Println("  ✅ Bundle structure is valid")

	// 2. Transaction Validation
	fmt.Println("\nValidating transactions...")
	if err := validateTransactions(committedBundle.Transactions); err != nil {
		return fmt.Errorf("transaction validation failed: %w", err)
	}
	fmt.Printf("  ✅ All %d transactions are valid\n", len(committedBundle.Transactions))

	// 3. Address Validation
	fmt.Println("\nValidating addresses...")
	if err := validateAddresses(committedBundle.Transactions); err != nil {
		return fmt.Errorf("address validation failed: %w", err)
	}
	fmt.Println("  ✅ All addresses are valid")

	// 4. Bundle Up-to-Date Check
	fmt.Println("\nChecking if bundle is up-to-date...")
	fmt.Println("   (generating fresh bundle to compare...)")
	if err := checkBundleUpToDate(committedData, bundlePath); err != nil {
		return fmt.Errorf("bundle up-to-date check failed: %w", err)
	}
	fmt.Println("  ✅ Bundle is up-to-date")

	fmt.Println("\n✅  NUT bundle validation passed")
	return nil
}

func validateBundleStructure(bundle *UpgradeBundle) error {
	// Check metadata exists and has version
	if bundle.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is empty")
	}

	// Validate version format (must be valid semver: X.Y.Z with optional pre-release/build metadata)
	semverPattern := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z\-\.]+)?(\+[0-9A-Za-z\-\.]+)?$`)
	if !semverPattern.MatchString(bundle.Metadata.Version) {
		return fmt.Errorf("metadata.version has invalid semver format: %s (expected format: X.Y.Z[-prerelease][+build])", bundle.Metadata.Version)
	}

	// Check transactions array exists
	if bundle.Transactions == nil {
		return fmt.Errorf("transactions array is nil")
	}

	if len(bundle.Transactions) == 0 {
		return fmt.Errorf("transactions array is empty")
	}

	return nil
}

func validateTransactions(txns []NetworkUpgradeTxn) error {
	for i, txn := range txns {
		// Check all required fields are present
		if txn.Data == "" {
			return fmt.Errorf("transaction %d (%s): data field is empty", i, txn.Intent)
		}

		// Validate data is hex string
		if !strings.HasPrefix(txn.Data, "0x") {
			return fmt.Errorf("transaction %d (%s): data is not a hex string", i, txn.Intent)
		}

		if _, err := hex.DecodeString(txn.Data[2:]); err != nil {
			return fmt.Errorf("transaction %d (%s): data is not valid hex: %w", i, txn.Intent, err)
		}

		if txn.To == "" {
			return fmt.Errorf("transaction %d (%s): to address is empty", i, txn.Intent)
		}

		if txn.From == "" {
			return fmt.Errorf("transaction %d (%s): from address is empty", i, txn.Intent)
		}

		if txn.GasLimit == 0 {
			return fmt.Errorf("transaction %d (%s): gasLimit is zero", i, txn.Intent)
		}

		if txn.Intent == "" {
			return fmt.Errorf("transaction %d: intent is empty", i)
		}

		// Intent should be descriptive (at least 5 characters)
		if len(txn.Intent) < 5 {
			return fmt.Errorf("transaction %d: intent too short: %s", i, txn.Intent)
		}
	}

	return nil
}

func validateAddresses(txns []NetworkUpgradeTxn) error {
	for i, txn := range txns {
		// Validate 'to' address
		if !isValidAddressFormat(txn.To) {
			return fmt.Errorf("transaction %d (%s): invalid 'to' address format: %s", i, txn.Intent, txn.To)
		}

		// All the deployments are done via ConditionalDeployer predeploy, so 'to' cannot be zero address.
		if strings.EqualFold(txn.To, ZERO_ADDRESS) {
			return fmt.Errorf("transaction %d (%s): 'to' address is zero address", i, txn.Intent)
		}

		// Validate 'from' address
		if !isValidAddressFormat(txn.From) {
			return fmt.Errorf("transaction %d (%s): invalid 'from' address format: %s", i, txn.Intent, txn.From)
		}
	}

	return nil
}

func isValidAddressFormat(addr string) bool {
	if !strings.HasPrefix(addr, "0x") {
		return false
	}

	if len(addr) != 42 { // 0x + 40 hex characters
		return false
	}

	// Check if all characters after 0x are valid hex
	_, err := hex.DecodeString(addr[2:])
	return err == nil
}

func checkBundleUpToDate(committedData []byte, bundlePath string) error {
	// Temporarily move the committed bundle
	originalPath := bundlePath
	backupPath := bundlePath + ".backup"

	if err := os.Rename(originalPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup bundle: %w", err)
	}

	// Ensure we restore the bundle even if generation fails
	defer func() {
		_ = os.Rename(backupPath, originalPath)
	}()

	// Generate new bundle using forge script
	cmd := exec.Command("forge", "script",
		"scripts/upgrade/GenerateNUTBundle.s.sol:GenerateNUTBundle",
		"--sig", "run()")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate bundle:\nstdout: %s\nstderr: %s\nerror: %w",
			stdout.String(), stderr.String(), err)
	}

	// Read the newly generated bundle
	generatedData, err := os.ReadFile(originalPath)
	if err != nil {
		return fmt.Errorf("failed to read generated bundle: %w", err)
	}

	// Restore the original bundle
	if err := os.Rename(backupPath, originalPath); err != nil {
		return fmt.Errorf("failed to restore original bundle: %w", err)
	}

	// Compare the two bundles
	if !bytes.Equal(committedData, generatedData) {
		// Pretty print the difference hint
		return fmt.Errorf("bundle is out of date\n\nThe committed bundle differs from the generated bundle.\nRun 'just generate-nut-bundle' to update it")
	}

	return nil
}
