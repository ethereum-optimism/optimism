package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
)

var excludeContracts = []string{
	// External dependencies
	"IERC20", "IERC721", "IERC5267", "IERC721Enumerable", "IERC721Upgradeable", "IERC721Metadata",
	"IERC165", "IERC165Upgradeable", "ERC721TokenReceiver", "ERC1155TokenReceiver",
	"ERC777TokensRecipient", "Guard", "IProxy", "Vm", "VmSafe", "IMulticall3",
	"IERC721TokenReceiver", "IProxyCreationCallback", "IBeacon", "IEIP712",

	// Generic interfaces
	"IHasSuperchainConfig",

	// EAS
	"IEAS", "ISchemaResolver", "ISchemaRegistry",

	// Misc stuff that can be ignored
	"IOPContractsManagerLegacyUpgrade",

	// TODO: Interfaces that need to be fixed
	"IInitializable", "IOptimismMintableERC20", "ILegacyMintableERC20",
	"KontrolCheatsBase", "IResolvedDelegateProxy",
}

// Contracts that don't need an interface
var excludeSourceContracts = []string{
	// Libraries don't need interfaces
	"Bytes", "Bytes32", "Encoding", "Hashing", "LibAddress", "LibBytesUtils", "LibMap", "LibMath", "LibSort", "LibString",

	// Test helpers and mocks
	"ERC20", "WETH9", "TestUtil", "TestERC20", "TestLib", "MockOVMCodeChecker", "MockERC20",
	"MIPS", "MIPSMemory", "PreimageOracle", "PreimageKeyLib",

	// Abstract contracts
	"CrossDomainMessenger", "Semver", "FeeVault", "StandardBridge", "OptimismPortal",

	// Special exclusions for specific reasons
	"SafeCall", "ResourceMetering", "Transactor", "AddressManager", "AddressResolver",
	"L1Block", "GasPriceOracle", "Burn",
	"AutoRedeem", "Initializable", "Predeploys", "Proxy",
	"SchemaRegistry", "AttestationStation", "EAS",

	// These are purely implementation contracts without interfaces needed
	"Ownable", "UUPSUpgradeable", "Clones", "Create2",
	"ProxyAdmin", "L1ChugSplashProxy", "ERC721Bridge", "ERC721BridgeLegacy",
	"CallContextHelper", "ImmutableProxy", "ResolvedDelegateProxy",

	// L1 contracts
	// NOTE: The following contracts should have interfaces and are intentionally NOT excluded:
	// - ETHLockbox
	// - DataAvailabilityChallenge
	// - L1CrossDomainMessenger
	// - L1ERC721Bridge
	// - L1StandardBridge
	"OPContractsManagerContractsContainer", "OPContractsManagerBase",
	"OPContractsManagerGameTypeAdder", "OPContractsManagerUpgrader", "OPContractsManagerDeployer",
	"OPContractsManagerInteropMigrator", "OPContractsManager", "OptimismPortal2",
	"ProtocolVersions", "ProxyAdminOwnedBase", "StandardValidatorBase", "StandardValidatorV180",
	"StandardValidatorV200", "StandardValidatorV300", "SuperchainConfig", "SystemConfig",

	// L2 contracts
	"BaseFeeVault", "CrossDomainOwnable", "CrossDomainOwnable2", "CrossDomainOwnable3",
	"CrossL2Inbox", "ETHLiquidity", "L1FeeVault", "L2CrossDomainMessenger", "L2ERC721Bridge",
	"L2StandardBridge", "L2StandardBridgeInterop", "L2ToL1MessagePasser", "L2ToL2CrossDomainMessenger",
	"OperatorFeeVault", "OptimismMintableERC721", "OptimismMintableERC721Factory",
	"OptimismSuperchainERC20", "OptimismSuperchainERC20Beacon", "OptimismSuperchainERC20Factory",
	"SequencerFeeVault", "SuperchainERC20", "SuperchainTokenBridge", "SuperchainWETH",

	// Cannon contracts
	"MIPS2", "MIPS64",

	// Dispute contracts
	"AnchorStateRegistry", "DelayedWETH", "DisputeGameFactory", "FaultDisputeGame",
	"PermissionedDisputeGame", "SuperFaultDisputeGame", "SuperPermissionedDisputeGame",

	// Governance contracts
	"GovernanceToken", "MintManager",

	// Integration contracts
	"EventLogger",

	// Legacy contracts
	"DeployerWhitelist", "L1BlockNumber", "LegacyMessagePasser", "LegacyMintableERC20",

	// Library-related contracts
	"Burner", "TransientReentrancyAware",

	// Periphery contracts
	"AssetReceiver", "TransferOnion", "Drippie", "CheckBalanceLow", "CheckSecrets",
	"CheckTrue", "Faucet", "AdminFaucetAuthModule", "DisputeMonitorHelper",

	// Safe contracts
	"DeputyGuardianModule", "DeputyPauseModule", "LivenessGuard", "LivenessModule",

	// Universal contracts
	"CrossDomainMessengerLegacySpacer0", "CrossDomainMessengerLegacySpacer1",
	"OptimismMintableERC20", "OptimismMintableERC20Factory", "ReinitializableBase",
	"SafeSend", "StorageSetter", "WETH98",

	// Vendor contracts
	"RISCV", "EIP1271Verifier", "SchemaResolver",
}

type ContractDefinition struct {
	ContractKind string `json:"contractKind"`
	Name         string `json:"name"`
}

type ASTNode struct {
	NodeType string   `json:"nodeType"`
	Literals []string `json:"literals,omitempty"`
	ContractDefinition
}

type ArtifactAST struct {
	Nodes []ASTNode `json:"nodes"`
}

type Artifact struct {
	AST ArtifactAST     `json:"ast"`
	ABI json.RawMessage `json:"abi"`
}

func main() {
	// Part 1: Check that all interfaces match their corresponding contracts
	if _, err := common.ProcessFilesGlob(
		[]string{"forge-artifacts/**/*.json"},
		[]string{},
		processFile,
	); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	// Part 2: Check that all contracts in src have a corresponding interface
	if err := verifyAllContractsHaveInterfaces(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func processFile(artifactPath string) (*common.Void, []error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, []error{fmt.Errorf("failed to get current working directory: %w", err)}
	}
	artifactsDir := filepath.Join(cwd, "forge-artifacts")

	contractName := strings.Split(filepath.Base(artifactPath), ".")[0]

	if isExcluded(contractName) {
		return nil, nil
	}

	artifact, err := readArtifact(artifactPath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read artifact: %w", err)}
	}

	contractDef := getContractDefinition(artifact, contractName)
	if contractDef == nil {
		return nil, nil // Skip processing if contract definition is not found
	}

	if contractDef.ContractKind != "interface" {
		return nil, nil
	}

	if !strings.HasPrefix(contractName, "I") {
		return nil, []error{fmt.Errorf("%s: Interface does not start with 'I'", contractName)}
	}

	semver, err := getContractSemver(artifact)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to get contract semver: %w", err)}
	}

	if semver != "solidity^0.8.0" {
		return nil, []error{fmt.Errorf("%s: Interface does not have correct compiler version (MUST be exactly solidity ^0.8.0)", contractName)}
	}

	contractBasename := contractName[1:]
	correspondingContractFile := filepath.Join(artifactsDir, contractBasename+".sol", contractBasename+".json")

	if _, err := os.Stat(correspondingContractFile); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	contractArtifact, err := readArtifact(correspondingContractFile)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read corresponding contract artifact: %w", err)}
	}

	interfaceABI := artifact.ABI
	contractABI := contractArtifact.ABI

	normalizedInterfaceABI, err := normalizeABI(interfaceABI)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to normalize interface ABI: %w", err)}
	}

	normalizedContractABI, err := normalizeABI(contractABI)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to normalize contract ABI: %w", err)}
	}

	match, err := compareABIs(normalizedInterfaceABI, normalizedContractABI)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to compare ABIs: %w", err)}
	}
	if !match {
		return nil, []error{fmt.Errorf("differences found")}
	}

	return nil, nil
}

func readArtifact(path string) (*Artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open artifact file: %w", err)
	}
	defer file.Close()

	var artifact Artifact
	if err := json.NewDecoder(file).Decode(&artifact); err != nil {
		return nil, fmt.Errorf("failed to parse artifact file: %w", err)
	}

	return &artifact, nil
}

func getContractDefinition(artifact *Artifact, contractName string) *ContractDefinition {
	for _, node := range artifact.AST.Nodes {
		if node.NodeType == "ContractDefinition" && node.Name == contractName {
			return &node.ContractDefinition
		}
	}
	return nil
}

func getContractSemver(artifact *Artifact) (string, error) {
	for _, node := range artifact.AST.Nodes {
		if node.NodeType == "PragmaDirective" {
			return strings.Join(node.Literals, ""), nil
		}
	}
	return "", errors.New("semver not found")
}

func normalizeABI(abi json.RawMessage) (json.RawMessage, error) {
	var abiData []map[string]interface{}
	if err := json.Unmarshal(abi, &abiData); err != nil {
		return nil, err
	}

	hasConstructor := false
	for i := range abiData {
		normalizeABIItem(abiData[i])
		if abiData[i]["type"] == "constructor" {
			hasConstructor = true
		}
	}

	// Add an empty constructor if it doesn't exist
	if !hasConstructor {
		emptyConstructor := map[string]interface{}{
			"type":            "constructor",
			"stateMutability": "nonpayable",
			"inputs":          []interface{}{},
		}
		abiData = append(abiData, emptyConstructor)
	}

	return json.Marshal(abiData)
}

func normalizeABIItem(item map[string]interface{}) {
	for key, value := range item {
		switch v := value.(type) {
		case string:
			if key == "internalType" {
				item[key] = normalizeInternalType(v)
			}
		case map[string]interface{}:
			normalizeABIItem(v)
		case []interface{}:
			for _, elem := range v {
				if elemMap, ok := elem.(map[string]interface{}); ok {
					normalizeABIItem(elemMap)
				}
			}
		}
	}

	if item["type"] == "function" && item["name"] == "__constructor__" {
		item["type"] = "constructor"
		delete(item, "name")
		delete(item, "outputs")
	}
}

func normalizeInternalType(internalType string) string {
	// Helper function to add 'I' prefix for non-interface types
	addIPrefix := func(match string) string {
		// Skip if it's already an interface pattern (I followed by uppercase)
		if len(match) > 1 && match[0] == 'I' && match[1] >= 'A' && match[1] <= 'Z' {
			return match
		}
		return "I" + match
	}

	// Replace patterns like "contract Something" with "contract ISomething"
	internalType = regexp.MustCompile(`(contract|struct|enum)\s+([^I]\w+|I[a-z]\w*)`).
		ReplaceAllStringFunc(internalType, func(s string) string {
			parts := strings.SplitN(s, " ", 2)
			return parts[0] + " " + addIPrefix(parts[1])
		})

	return internalType
}

func compareABIs(abi1, abi2 json.RawMessage) (bool, error) {
	var interfaceABI, contractABI []map[string]interface{}

	if err := json.Unmarshal(abi1, &interfaceABI); err != nil {
		return false, fmt.Errorf("error unmarshalling interface ABI: %w", err)
	}

	if err := json.Unmarshal(abi2, &contractABI); err != nil {
		return false, fmt.Errorf("error unmarshalling contract ABI: %w", err)
	}

	// Create maps for easier lookup
	interfaceItems := make(map[string]map[string]interface{})
	contractItems := make(map[string]map[string]interface{})

	// Helper to create a unique key for each ABI item
	makeKey := func(item map[string]interface{}) string {
		itemType := getString(item, "type")
		itemName := getString(item, "name")
		inputs, _ := json.Marshal(item["inputs"])
		outputs, _ := json.Marshal(item["outputs"])
		return fmt.Sprintf("%s_%s_%s_%s", itemType, itemName, inputs, outputs)
	}

	// Build lookup maps
	for _, item := range interfaceABI {
		key := makeKey(item)
		interfaceItems[key] = item
	}
	for _, item := range contractABI {
		key := makeKey(item)
		contractItems[key] = item
	}

	// Check for missing items in both directions
	isMatch := true

	// Check interface items exist in contract
	for key, item := range interfaceItems {
		if _, exists := contractItems[key]; !exists {
			itemType := getString(item, "type")
			signature := formatABIItem(item)
			log.Printf("REMOVE %s from interface: %s", itemType, signature)
			isMatch = false
		}
	}

	// Check contract items exist in interface
	for key, item := range contractItems {
		if _, exists := interfaceItems[key]; !exists {
			itemType := getString(item, "type")
			signature := formatABIItem(item)
			log.Printf("ADD %s to interface: %s", itemType, signature)
			isMatch = false
		}
	}

	return isMatch, nil
}

// Helper function to format ABI item into a readable signature
func formatABIItem(item map[string]interface{}) string {
	itemType := getString(item, "type")
	itemName := getString(item, "name")

	// Format inputs
	inputs, _ := item["inputs"].([]interface{})
	inputStr := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if inputMap, ok := input.(map[string]interface{}); ok {
			internalType := getString(inputMap, "internalType")
			paramType := internalType
			if parts := strings.Fields(internalType); len(parts) == 2 {
				paramType = parts[1]
			}
			paramName := getString(inputMap, "name")
			if paramName != "" {
				inputStr = append(inputStr, fmt.Sprintf("%s %s", paramType, paramName))
			} else {
				inputStr = append(inputStr, paramType)
			}
		}
	}

	// Format outputs
	outputs, _ := item["outputs"].([]interface{})
	outputStr := make([]string, 0, len(outputs))
	for _, output := range outputs {
		if outputMap, ok := output.(map[string]interface{}); ok {
			internalType := getString(outputMap, "internalType")
			paramType := internalType
			if parts := strings.Fields(internalType); len(parts) == 2 {
				paramType = parts[1]
			}
			paramName := getString(outputMap, "name")
			if paramName != "" {
				outputStr = append(outputStr, fmt.Sprintf("%s %s", paramType, paramName))
			} else {
				outputStr = append(outputStr, paramType)
			}
		}
	}

	// Build the signature based on the item type
	switch itemType {
	case "function":
		returnStr := ""
		if len(outputStr) > 0 {
			returnStr = fmt.Sprintf(" returns (%s)", strings.Join(outputStr, ", "))
		}
		return fmt.Sprintf("function %s(%s)%s", itemName, strings.Join(inputStr, ", "), returnStr)
	case "event":
		return fmt.Sprintf("event %s(%s)", itemName, strings.Join(inputStr, ", "))
	case "constructor":
		return fmt.Sprintf("constructor(%s)", strings.Join(inputStr, ", "))
	default:
		return fmt.Sprintf("%s %s(%s)", itemType, itemName, strings.Join(inputStr, ", "))
	}
}

func isExcluded(contractName string) bool {
	for _, exclude := range excludeContracts {
		if exclude == contractName {
			return true
		}
	}
	return false
}

// getString safely retrieves a string value from a map[string]interface{}
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Check if a source contract is in the exclude list
func isExcludedSourceContract(contractName string) bool {
	for _, excluded := range excludeSourceContracts {
		if excluded == contractName {
			return true
		}
	}
	return false
}

// Function to verify that all contracts in the src directory have corresponding interfaces
func verifyAllContractsHaveInterfaces() error {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Process contract files using common.ProcessFilesGlob
	processContract := func(path string) (*common.Void, []error) {
		// Read the file to determine if it's a contract
		file, err := os.ReadFile(path)
		if err != nil {
			return nil, []error{fmt.Errorf("failed to read file %s: %w", path, err)}
		}

		// Simple regex to find contract definitions
		contractRegex := regexp.MustCompile(`(?m)^\s*(contract|abstract contract)\s+(\w+)`)
		matches := contractRegex.FindAllStringSubmatch(string(file), -1)

		var errs []error
		for _, match := range matches {
			if len(match) >= 3 {
				contractName := match[2]

				// Skip contracts that are excluded
				if isExcludedSourceContract(contractName) {
					continue
				}

				// For contracts in src/L1, interfaces should be at interfaces/L1
				// Check if interface exists at the predictable location
				interfacePath := filepath.Join(cwd, "interfaces", "L1", "I"+contractName+".sol")
				_, err := os.Stat(interfacePath)

				if os.IsNotExist(err) {
					// Get relative paths for error message
					contractRelPath, _ := filepath.Rel(cwd, path)
					// Use the correct path structure for the error message
					interfaceRelPath := filepath.Join("interfaces", "L1", "I"+contractName+".sol")
					errs = append(errs, fmt.Errorf("Contract %s in %s does not have a corresponding interface at %s",
						contractName, contractRelPath, interfaceRelPath))
				}
			}
		}

		return nil, errs
	}

	// Find and process only .sol files in src/L1 directory as suggested by smartcontracts
	_, err = common.ProcessFilesGlob(
		[]string{"src/L1/**/*.sol"},
		[]string{},
		processContract,
	)

	if err != nil {
		return fmt.Errorf("error processing src/L1 directory: %w", err)
	}

	return nil
}
