package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// generates the standard-versions-<network>.toml file content for a given release tag, to facilitate updates to the
// Superchain Registry.
//
// Usage:
// go run ./scripts/autogen/standard-versions \
//   --release <release tag> \
//   --opcm <OPCM_ADDRESS> \
//   --rpc-url <RPC_URL>
func main() {
	release := flag.String("release", "", "Release version (e.g., op-contracts/v5.0.0-rc.2)")
	opcm := flag.String("opcm", "", "OPCM contract address")
	rpcURL := flag.String("rpc-url", "", "RPC URL")
	flag.Parse()

	if *release == "" || *opcm == "" || *rpcURL == "" {
		fmt.Fprintf(os.Stderr, "Error: --release, --opcm, and --rpc-url are required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Verify that the current git commit matches the release tag
	if err := verifyGitTag(*release); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 1. First output the release header
	fmt.Printf("[\"%s\"]\n", *release)

	// 2. Get OPCM version
	opcmVersion, err := getVersion(*opcm, *rpcURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting OPCM version: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("op_contracts_manager = { version = \"%s\", address = \"%s\" }\n", opcmVersion, strings.ToLower(*opcm))

	// 3. Get implementations and their versions
	implementations, err := getImplementations(*opcm, *rpcURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting implementations: %v\n", err)
		os.Exit(1)
	}
	if err := outputImplementations(implementations, *rpcURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting implementations: %v\n", err)
		os.Exit(1)
	}

	// 4. Get the Dispute game versions from the source code (these contracts are deployed via blueprint, so the versions
	// cannot easily be read for the OPCM's implementations struct).
	faultVersion, err := extractVersionFromSource("src/dispute/FaultDisputeGame.sol")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting FaultDisputeGame version: %v\n", err)
		os.Exit(1)
	}
	permissionedVersion, err := extractVersionFromSource("src/dispute/PermissionedDisputeGame.sol")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting PermissionedDisputeGame version: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("fault_dispute_game = { version = \"%s\" }\n", faultVersion)
	fmt.Printf("permissioned_dispute_game = { version = \"%s\" }\n", permissionedVersion)
}

// verifyGitTag checks that the current git commit matches the specified tag
func verifyGitTag(tag string) error {
	// Get the current commit hash
	currentCmd := exec.Command("git", "rev-parse", "HEAD")
	currentOutput, err := currentCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}
	currentCommit := strings.TrimSpace(string(currentOutput))

	// Get the commit hash for the tag
	tagCmd := exec.Command("git", "rev-list", "-n", "1", tag)
	tagOutput, err := tagCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit for tag %s: %w (is the tag valid?)", tag, err)
	}
	tagCommit := strings.TrimSpace(string(tagOutput))

	// Compare the commits
	if currentCommit != tagCommit {
		return fmt.Errorf("current commit (%s) does not match tag %s (%s)", currentCommit[:8], tag, tagCommit[:8])
	}

	return nil
}

// getVersion calls version() on a contract and returns the version string
func getVersion(address, rpcURL string) (string, error) {
	// version() function signature: 54fd4d50
	cmd := exec.Command("cast", "call", address, "version()(string)", "--rpc-url", rpcURL)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("cast call failed: %w", err)
	}

	version := strings.TrimSpace(string(output))
	// Remove surrounding quotes if present
	version = strings.Trim(version, "\"")
	return version, nil
}

// extractVersionFromSource extracts the version string from a Solidity source file
func extractVersionFromSource(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Look for: function version() public pure returns (string memory) { return "X.Y.Z"; }
	functionPattern := regexp.MustCompile(`function\s+version\(\).*?\{[^}]*return\s+"([^"]+)"`)
	if matches := functionPattern.FindStringSubmatch(string(content)); len(matches) > 1 {
		return matches[1], nil
	}

	// Look for: string constant version = "X.Y.Z";
	constantPattern := regexp.MustCompile(`string\s+.*?\s+version\s*=\s*"([^"]+)"`)
	if matches := constantPattern.FindStringSubmatch(string(content)); len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("version not found in %s", filePath)
}

// Implementation represents an implementation contract
type Implementation struct {
	Name    string
	Address string
}

// ABIComponent represents a component in an ABI struct
type ABIComponent struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ABIOutput represents an output in an ABI function
type ABIOutput struct {
	Components []ABIComponent `json:"components"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
}

// ABIFunction represents a function in an ABI
type ABIFunction struct {
	Name            string      `json:"name"`
	Outputs         []ABIOutput `json:"outputs"`
	StateMutability string      `json:"stateMutability"`
	Type            string      `json:"type"`
}

// getImplementationsABI reads the ABI and returns field names and count
func getImplementationsABI() ([]string, int, error) {
	abiPath := "snapshots/abi/OPContractsManagerContractsContainer.json"
	data, err := os.ReadFile(abiPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read ABI: %w", err)
	}

	var abi []ABIFunction
	if err := json.Unmarshal(data, &abi); err != nil {
		return nil, 0, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Find the implementations() function
	for _, fn := range abi {
		if fn.Name == "implementations" && fn.Type == "function" {
			if len(fn.Outputs) == 0 {
				return nil, 0, fmt.Errorf("implementations() has no outputs")
			}
			components := fn.Outputs[0].Components
			if len(components) == 0 {
				return nil, 0, fmt.Errorf("implementations() output has no components")
			}

			fieldNames := make([]string, len(components))
			for i, comp := range components {
				fieldNames[i] = comp.Name
			}
			return fieldNames, len(components), nil
		}
	}

	return nil, 0, fmt.Errorf("implementations() function not found in ABI")
}

// getImplementations calls implementations() on OPCM and returns the struct of addresses
func getImplementations(opcm, rpcURL string) ([]Implementation, error) {
	// Get field names and count from ABI
	fieldNames, count, err := getImplementationsABI()
	if err != nil {
		return nil, fmt.Errorf("failed to get ABI info: %w", err)
	}

	// Call implementations() with dynamic array size
	signature := fmt.Sprintf("implementations()(address[%d])", count)
	cmd := exec.Command("cast", "call", opcm, signature, "--rpc-url", rpcURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cast call failed: %w", err)
	}

	// Parse the output - cast returns addresses as an array: [0x..., 0x..., ...]
	outputStr := strings.TrimSpace(string(output))

	// Remove the brackets and split by comma
	outputStr = strings.TrimPrefix(outputStr, "[")
	outputStr = strings.TrimSuffix(outputStr, "]")

	addresses := strings.Split(outputStr, ",")
	if len(addresses) != count {
		return nil, fmt.Errorf("expected %d addresses, got %d", count, len(addresses))
	}

	implementations := make([]Implementation, 0, len(fieldNames))
	for i, addr := range addresses {
		address := strings.TrimSpace(addr)

		// Check if address is zero - warn but skip
		if isZeroAddress(address) {
			fmt.Fprintf(os.Stderr, "Warning: implementation %s has zero address, skipping\n", fieldNames[i])
			continue
		}

		implementations = append(implementations, Implementation{
			Name:    fieldNames[i],
			Address: strings.ToLower(address),
		})
	}

	return implementations, nil
}

// isZeroAddress checks if an address is 0x0000...
func isZeroAddress(addr string) bool {
	addr = strings.TrimPrefix(strings.ToLower(addr), "0x")
	return addr == "" || regexp.MustCompile(`^0+$`).MatchString(addr)
}

// nameMapping maps ABI field names to TOML keys
var nameMapping = map[string]string{
	"superchainConfigImpl":              "superchain_config",
	"protocolVersionsImpl":              "protocol_versions",
	"l1ERC721BridgeImpl":                "l1_erc721_bridge",
	"optimismPortalImpl":                "optimism_portal",
	"optimismPortalInteropImpl":         "optimism_portal_interop",
	"ethLockboxImpl":                    "eth_lockbox",
	"systemConfigImpl":                  "system_config",
	"optimismMintableERC20FactoryImpl":  "optimism_mintable_erc20_factory",
	"l1CrossDomainMessengerImpl":        "l1_cross_domain_messenger",
	"l1StandardBridgeImpl":              "l1_standard_bridge",
	"disputeGameFactoryImpl":            "dispute_game_factory",
	"anchorStateRegistryImpl":           "anchor_state_registry",
	"delayedWETHImpl":                   "delayed_weth",
	"mipsImpl":                          "mips",
	"preimageOracleImpl":                "preimage_oracle",
	"faultDisputeGameV2Impl":            "fault_dispute_game_v2",
	"permissionedDisputeGameV2Impl":     "permissioned_dispute_game_v2",
}

// tomlKeyFromABIName converts an ABI field name to a TOML key
func tomlKeyFromABIName(abiName string) (string, error) {
	tomlKey, ok := nameMapping[abiName]
	if !ok {
		return "", fmt.Errorf("unknown ABI field name: %s (needs to be added to nameMapping)", abiName)
	}
	return tomlKey, nil
}

// outputImplementations outputs each implementation with its version in TOML format
func outputImplementations(implementations []Implementation, rpcURL string) error {
	// Special contracts that use "address" instead of "implementation_address"
	nonProxied := map[string]bool{
		"preimageOracleImpl": true,
		"mipsImpl":           true,
	}

	for _, impl := range implementations {
		version, err := getVersion(impl.Address, rpcURL)
		if err != nil {
			return fmt.Errorf("failed to get version for %s at %s: %w", impl.Name, impl.Address, err)
		}

		tomlKey, err := tomlKeyFromABIName(impl.Name)
		if err != nil {
			return err
		}

		// Determine which address field to use
		addressField := "implementation_address"
		if nonProxied[impl.Name] {
			addressField = "address"
		}

		fmt.Printf("%s = { version = \"%s\", %s = \"%s\" }\n", tomlKey, version, addressField, impl.Address)
	}

	return nil
}
