package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
)

// Main entry point

// Validates test function naming conventions and structure in Forge test artifacts
func main() {
	if _, err := common.ProcessFilesGlob(
		[]string{"forge-artifacts/**/*.t.sol/*.json"},
		[]string{},
		processFile,
	); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// Processes a single test artifact file and runs all validations
func processFile(path string) (*common.Void, []error) {
	// Read and parse the Forge artifact file
	artifact, err := common.ReadForgeArtifact(path)
	if err != nil {
		return nil, []error{err}
	}

	var errors []error

	// Validate test function naming conventions
	testNameErrors := validateTestName(artifact)
	errors = append(errors, testNameErrors...)

	// Validate test function structure and organization
	structureErrors := validateTestStructure(artifact)
	errors = append(errors, structureErrors...)

	return nil, errors
}

// Test name validation

// Validates that test function names follow the expected naming conventions
func validateTestName(artifact *solc.ForgeArtifact) []error {
	var errors []error

	// Extract all test function names from the artifact
	names := extractTestNames(artifact)

	// Check each test name against naming conventions
	for _, name := range names {
		if err := checkTestName(name); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// Extracts all test function names from the artifact
func extractTestNames(artifact *solc.ForgeArtifact) []string {
	// Check if this is a test contract by looking for IS_TEST method
	isTest := false
	for _, entry := range artifact.Abi.Parsed.Methods {
		if entry.Name == "IS_TEST" {
			isTest = true
			break
		}
	}
	// Skip non-test contracts
	if !isTest {
		return nil
	}

	// Collect all method names that start with "test"
	names := []string{}
	for _, entry := range artifact.Abi.Parsed.Methods {
		if !strings.HasPrefix(entry.Name, "test") {
			continue
		}
		names = append(names, entry.Name)
	}

	return names
}

// Validates a single test name against all naming rules
func checkTestName(name string) error {
	// Split the test name into parts
	parts := strings.Split(name, "_")

	// Check each part against the defined validation rules
	for _, check := range checks {
		if !check.check(parts) {
			return fmt.Errorf("%s: %s", name, check.error)
		}
	}
	return nil
}

// Test structure validation

// Validates the overall structure and organization of test contracts
func validateTestStructure(artifact *solc.ForgeArtifact) []error {
	var errors []error

	// Extract file path from compilation target
	filePath, _, err := getCompilationTarget(artifact)
	if err != nil {
		errors = append(errors, err)
	}

	// Skip validation for excluded files
	if isExcluded(filePath) {
		return nil
	}

	// Validate test file path matches corresponding src path
	if !checkSrcPath(artifact) {
		errors = append(errors, fmt.Errorf("test file path does not match src path"))
	}

	// Validate contract name matches file path
	if !checkContractNameFilePath(artifact) {
		errors = append(errors, fmt.Errorf("contract name does not match file path"))
	}

	// Validate contract naming pattern structure
	structureErrors := checkTestStructure(artifact)
	errors = append(errors, structureErrors...)

	return errors
}

// Validates the contract naming pattern structure
func checkTestStructure(artifact *solc.ForgeArtifact) []error {
	var errors []error

	// Validate each contract name in the compilation target
	for _, contractName := range artifact.Metadata.Settings.CompilationTarget {
		contractParts := strings.Split(contractName, "_")

		// Check for initialization test pattern
		if len(contractParts) == 2 && contractParts[1] == "TestInit" {
			// Pattern: <ContractName>_TestInit
			continue
		} else if len(contractParts) == 2 && contractParts[1] == "Harness" {
			// Pattern: <ContractName>_Harness
			continue
		} else if len(contractParts) == 3 && contractParts[2] == "Test" {
			// Check for uncategorized test pattern
			if contractParts[1] == "Uncategorized" || contractParts[1] == "Unclassified" {
				// Pattern: <ContractName>_Uncategorized_Test
				continue
			} else {
				// Pattern: <ContractName>_<FunctionName>_Test - validate function exists
				functionName := contractParts[1]
				if !checkFunctionExists(artifact, functionName) {
					// Convert to camelCase for error message
					camelCaseFunctionName := strings.ToLower(functionName[:1]) + functionName[1:]
					errors = append(errors, fmt.Errorf("contract '%s': function '%s' does not exist in source contract", contractName, camelCaseFunctionName))
				}
			}
		} else if len(contractParts) == 3 && contractParts[2] == "Harness" {
			// Pattern: <ContractName>_<Descriptor>_Harness
			// (e.g., OPContractsManager_Upgrade_Harness)
			continue
		} else {
			// Invalid naming pattern
			errors = append(errors, fmt.Errorf("contract '%s': invalid naming pattern. Expected patterns: <ContractName>_TestInit, <ContractName>_<FunctionName>_Test, or <ContractName>_Uncategorized_Test", contractName))
		}
	}

	return errors
}

// Artifact and path validation helpers

// Extracts the compilation target from the artifact
func getCompilationTarget(artifact *solc.ForgeArtifact) (string, string, error) {
	// Return the first (and expected only) compilation target
	for filePath, contractName := range artifact.Metadata.Settings.CompilationTarget {
		return filePath, contractName, nil
	}

	// This should never happen since we already returned if we found a target,
	// but verify there's exactly one compilation target as expected
	if len(artifact.Metadata.Settings.CompilationTarget) != 1 {
		return "", "", fmt.Errorf("expected 1 compilation target, got %d", len(artifact.Metadata.Settings.CompilationTarget))
	}

	return "", "", nil
}

// Validates that the test file has a corresponding source file
func checkSrcPath(artifact *solc.ForgeArtifact) bool {
	for testFilePath := range artifact.Metadata.Settings.CompilationTarget {
		// Ensure file is in test directory with .t.sol extension
		if !strings.HasPrefix(testFilePath, "test/") || !strings.HasSuffix(testFilePath, ".t.sol") {
			return false
		}

		// Convert test path to corresponding source path
		srcPath := "src/" + strings.TrimPrefix(testFilePath, "test/")
		srcPath = strings.TrimSuffix(srcPath, ".t.sol") + ".sol"

		// Check if source file exists
		_, err := os.Stat(srcPath)
		if err != nil {
			return false
		}
	}

	return true
}

// Validates that contract name matches the file path
func checkContractNameFilePath(artifact *solc.ForgeArtifact) bool {
	for filePath, contractName := range artifact.Metadata.Settings.CompilationTarget {
		// Split contract name to get the base contract name (before first underscore)
		contractParts := strings.Split(contractName, "_")
		// Split file path to get individual path components
		filePathParts := strings.Split(filePath, "/")

		// Extract filename without .t.sol extension
		fileName := strings.TrimSuffix(filePathParts[len(filePathParts)-1], ".t.sol")
		// Check if filename matches the base contract name
		if fileName != contractParts[0] {
			return false
		}
	}
	return true
}

// Finds the artifact file path for a given contract
func findArtifactPath(contractFileName, contractName string) (string, error) {
	// Construct the pattern to match the artifact file
	pattern := "forge-artifacts/" + contractFileName + "/" + contractName + "*.json"

	// Use filepath.Glob to find matching files
	files, err := filepath.Glob(pattern)

	// If no files found or there was an error, return an error
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no artifact found for %s", contractName)
	}
	return files[0], nil
}

// Validates that a function exists in the source contract
func checkFunctionExists(artifact *solc.ForgeArtifact, functionName string) bool {
	// Special functions always exist
	if strings.EqualFold(functionName, "constructor") || strings.EqualFold(functionName, "receive") || strings.EqualFold(functionName, "fallback") {
		return true
	}

	// Check each compilation target for the function
	for filePath := range artifact.Metadata.Settings.CompilationTarget {
		// Convert test path to source path
		srcPath := strings.TrimPrefix(filePath, "test/")
		srcPath = strings.TrimSuffix(srcPath, ".t.sol") + ".sol"

		// Extract contract name from file path
		pathParts := strings.Split(srcPath, "/")
		contractFileName := pathParts[len(pathParts)-1]
		contractName := strings.TrimSuffix(contractFileName, ".sol")

		// Find the corresponding source artifact
		srcArtifactPath, err := findArtifactPath(contractFileName, contractName)
		if err != nil {
			return false
		}

		// Read the source contract artifact
		srcArtifact, err := common.ReadForgeArtifact(srcArtifactPath)
		if err != nil {
			fmt.Printf("Failed to read artifact: %s, error: %v\n", srcArtifactPath, err)
			return false
		}

		// Check if function exists in the ABI
		for _, method := range srcArtifact.Abi.Parsed.Methods {
			if strings.EqualFold(method.Name, functionName) {
				return true
			}
		}
	}
	return false
}

// Exclusion configuration

// Checks if a file path should be excluded from validation
func isExcluded(filePath string) bool {
	for _, excluded := range excludedPaths {
		if strings.HasPrefix(filePath, excluded) {
			return true
		}
	}
	return false
}

// Defines the list of paths that should be excluded from validation
var excludedPaths = []string{
	// PATHS EXCLUDED FROM SRC VALIDATION:
	// These paths are excluded because they don't follow the standard naming convention
	// where test files (*.t.sol) have corresponding source files (*.sol) in the src/ directory.
	// Instead, they follow alternative naming conventions or serve specialized purposes:
	// - Some are utility/infrastructure tests that don't test specific contracts
	// - Some test external libraries or vendor code that exists elsewhere
	// - Some are integration tests that test multiple contracts together
	// - Some are specialized test types (invariants, formal verification, etc.)
	//
	// Resolving these naming inconsistencies is outside the current scope, but they
	// are documented here to avoid false validation failures while maintaining
	// the validation rules for standard contract tests.
	"test/invariants/",                    // Invariant testing framework - no direct src counterpart
	"test/kontrol/",                       // Formal verification tests - specialized tooling
	"test/opcm/",                          // OP Chain Manager tests - may have different structure
	"test/scripts/",                       // Script tests - test deployment/utility scripts, not contracts
	"test/integration/",                   // Integration tests - test multiple contracts together
	"test/setup/",                         // Test setup utilities - infrastructure, not contract tests
	"test/cannon/MIPS64Memory.t.sol",      // Tests external MIPS implementation
	"test/dispute/lib/LibClock.t.sol",     // Tests library utilities
	"test/dispute/lib/LibGameId.t.sol",    // Tests library utilities
	"test/universal/BenchmarkTest.t.sol",  // Performance benchmarking tests
	"test/universal/ExtendedPause.t.sol",  // Tests extended functionality
	"test/vendor/Initializable.t.sol",     // Tests external vendor code
	"test/vendor/InitializableOZv5.t.sol", // Tests external vendor code

	// PATHS EXCLUDED FROM CONTRACT NAME FILE PATH VALIDATION:
	// These paths are excluded because they don't follow the standard naming convention
	// where the contract name matches the file name pattern: <FileName>_<Function>_Test.
	// Instead, these files contain contracts with names like <AnotherName>_<Function>_Test,
	// where the base contract name doesn't match the file name.
	//
	// This typically occurs when:
	// - The test file contains helper contracts or alternative implementations
	// - The test file tests multiple related contracts or contract variants
	// - The test file uses a different naming strategy for organizational purposes
	// - The contracts being tested have complex inheritance or composition patterns
	//
	// These naming inconsistencies may indicate the presence of specialized test
	// infrastructure beyond standard harnesses or different setup contracts patterns.
	"test/dispute/FaultDisputeGame.t.sol",       // Contains contracts not matching FaultDisputeGame base name
	"test/dispute/SuperFaultDisputeGame.t.sol",  // Contains contracts not matching SuperFaultDisputeGame base name
	"test/L1/ResourceMetering.t.sol",            // Contains contracts not matching ResourceMetering base name
	"test/L1/StandardValidator.t.sol",           // Contains contracts not matching StandardValidator base name
	"test/L2/CrossDomainOwnable.t.sol",          // Contains contracts not matching CrossDomainOwnable base name
	"test/L2/CrossDomainOwnable2.t.sol",         // Contains contracts not matching CrossDomainOwnable2 base name
	"test/L2/CrossDomainOwnable3.t.sol",         // Contains contracts not matching CrossDomainOwnable3 base name
	"test/L2/GasPriceOracle.t.sol",              // Contains contracts not matching GasPriceOracle base name
	"test/legacy/L1ChugSplashProxy.t.sol",       // Contains contracts not matching L1ChugSplashProxy base name
	"test/legacy/ResolvedDelegateProxy.t.sol",   // Contains contracts not matching ResolvedDelegateProxy base name
	"test/libraries/Blueprint.t.sol",            // Contains contracts not matching Blueprint base name
	"test/libraries/SafeCall.t.sol",             // Contains contracts not matching SafeCall base name
	"test/periphery/drippie/Drippie.t.sol",      // Contains contracts not matching Drippie base name
	"test/safe/LivenessGuard.t.sol",             // Contains contracts not matching LivenessGuard base name
	"test/universal/CrossDomainMessenger.t.sol", // Contains contracts not matching CrossDomainMessenger base name
	"test/universal/Proxy.t.sol",                // Contains contracts not matching Proxy base name
	"test/universal/StandardBridge.t.sol",       // Contains contracts not matching StandardBridge base name

	// PATHS EXCLUDED FROM FUNCTION NAME VALIDATION:
	// These paths are excluded because they don't pass the function name validation,
	// which checks that the function in the <FileName>_<Function>_Test pattern
	// actually exists in the source contract's ABI.
	//
	// Common reasons for exclusion:
	// - Libraries: Have different artifact structures that the validation system
	//   doesn't currently support, making function name lookup impossible
	// - Internal/Private functions: Some contracts test internal functions that
	//   aren't exposed in the public ABI, so they can't be validated
	// - Misspelled/Incorrect function names: Test contracts may have typos or
	//   incorrect function names that don't match the actual source contract
	//
	// Resolving these issues requires either:
	// - Enhancing the validation system to support libraries and complex structures
	// - Fixing misspelled function names in test contracts
	// - Restructuring tests to match actual function signatures
	"test/libraries",                     // Libraries have different artifact structure, unsupported
	"test/dispute/lib/LibPosition.t.sol", // Library testing - artifact structure issues
	"test/L1/ProxyAdminOwnedBase.t.sol",  // Tests internal functions not in ABI
	"test/L1/SystemConfig.t.sol",         // Tests internal functions not in ABI
	"test/safe/SafeSigners.t.sol",        // Function name validation issues

	// PATHS EXCLUDED FROM TEST PATTERN VALIDATION:
	// These paths are excluded because they don't follow the standard test contract
	// naming patterns defined in the validation system. The expected patterns are:
	// - <ContractName>_TestInit: for initialization/setup contracts
	// - <ContractName>_<FunctionName>_Test: for testing specific functions
	// - <ContractName>_<Descriptor>_Harness: for harness contracts
	// - <ContractName>_Uncategorized_Test: for miscellaneous tests
	//
	// Common reasons for exclusion:
	// - Multiple test contracts for the same function: When testing the same function
	//   requires different setups or contexts, multiple test contracts are needed
	//   (e.g., testing convert() with different token types requires separate contracts)
	// - Legacy naming patterns: Some contracts use older naming conventions like
	//   <ContractName>_<Function>_TestFail that need to be updated to _Test
	//
	// Resolving these issues requires either:
	// - Refactoring tests to fit standard patterns where possible
	// - Updating legacy TestFail patterns to use standard _Test naming
	// - Enhancing the validation system to support legitimate multiple-contract scenarios
	"test/L2/L2StandardBridgeInterop.t.sol", // Multiple contracts test same convert() function with different setups
}

// Test name validation rules

// Defines the signature for test name validation functions
type CheckFunc func(parts []string) bool

// Contains validation logic and error message for a specific rule
type CheckInfo struct {
	error string
	check CheckFunc
}

// Defines all the validation rules for test names
var checks = map[string]CheckInfo{
	// Ensure no empty parts between underscores (e.g., "test__name" is invalid)
	"doubleUnderscores": {
		error: "test names cannot have double underscores",
		check: func(parts []string) bool {
			for _, part := range parts {
				if len(strings.TrimSpace(part)) == 0 {
					return false
				}
			}
			return true
		},
	},
	// Each part should start with lowercase letter (camelCase within parts is allowed)
	"camelCase": {
		error: "test name parts should be in camelCase",
		check: func(parts []string) bool {
			for _, part := range parts {
				if len(part) > 0 && unicode.IsUpper(rune(part[0])) {
					return false
				}
			}
			return true
		},
	},
	// Test names must have exactly 3 or 4 underscore-separated parts
	"partsCount": {
		error: "test names should have either 3 or 4 parts, each separated by underscores",
		check: func(parts []string) bool {
			return len(parts) == 3 || len(parts) == 4
		},
	},
	// First part must be one of the allowed test prefixes
	"prefix": {
		error: "test names should begin with 'test', 'testFuzz', or 'testDiff'",
		check: func(parts []string) bool {
			return len(parts) > 0 && (parts[0] == "test" || parts[0] == "testFuzz" || parts[0] == "testDiff")
		},
	},
	// Last part must indicate test outcome or be a benchmark
	"suffix": {
		error: "test names should end with either 'succeeds', 'reverts', 'fails', 'works', or 'benchmark[_num]'",
		check: func(parts []string) bool {
			if len(parts) == 0 {
				return false
			}
			last := parts[len(parts)-1]
			if last == "succeeds" || last == "reverts" || last == "fails" || last == "works" {
				return true
			}
			// Handle benchmark_<number> pattern
			if len(parts) >= 2 && parts[len(parts)-2] == "benchmark" {
				_, err := strconv.Atoi(last)
				return err == nil
			}
			return last == "benchmark"
		},
	},
	// Failure tests (ending with "reverts" or "fails") must have 4 parts to include failure reason
	"failureParts": {
		error: "failure tests should have 4 parts, third part should indicate the reason for failure",
		check: func(parts []string) bool {
			if len(parts) == 0 {
				return false
			}
			last := parts[len(parts)-1]
			return len(parts) == 4 || (last != "reverts" && last != "fails")
		},
	},
}
