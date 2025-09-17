package main

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Tests for processFile function
func TestProcessFile(t *testing.T) {
	// Create a temporary directory with test artifacts
	tempDir := t.TempDir()

	// Create artifact file
	artifactPath := filepath.Join(tempDir, "valid.json")
	if err := os.WriteFile(artifactPath, []byte(`{"abi":[],"metadata":{"settings":{"compilationTarget":{}}}}`), 0644); err != nil {
		t.Fatalf("Failed to create artifact file: %v", err)
	}

	// Test with invalid JSON
	invalidPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte(`invalid json`), 0644); err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	tests := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{"invalid JSON file", invalidPath, true},
		{"non-existent file", filepath.Join(tempDir, "missing.json"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := processFile(tt.filePath)
			hasError := len(errors) > 0
			if hasError != tt.expectError {
				t.Errorf("processFile() hasError = %v, expectError %v", hasError, tt.expectError)
			}
		})
	}
}

// Tests for validateTestName function
func TestValidateTestName(t *testing.T) {
	tests := []struct {
		name         string
		artifact     *solc.ForgeArtifact
		expectErrors int
	}{
		{
			name: "valid test contract with valid names",
			artifact: &solc.ForgeArtifact{
				Abi: solc.AbiType{
					Parsed: abi.ABI{
						Methods: map[string]abi.Method{
							"IS_TEST":                 {Name: "IS_TEST"},
							"test_something_succeeds": {Name: "test_something_succeeds"},
							"testFuzz_other_works":    {Name: "testFuzz_other_works"},
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "test contract with invalid names",
			artifact: &solc.ForgeArtifact{
				Abi: solc.AbiType{
					Parsed: abi.ABI{
						Methods: map[string]abi.Method{
							"IS_TEST":            {Name: "IS_TEST"},
							"test_Invalid_fails": {Name: "test_Invalid_fails"},
							"test_bad":           {Name: "test_bad"},
						},
					},
				},
			},
			expectErrors: 2,
		},
		{
			name: "non-test contract",
			artifact: &solc.ForgeArtifact{
				Abi: solc.AbiType{
					Parsed: abi.ABI{
						Methods: map[string]abi.Method{
							"someMethod": {Name: "someMethod"},
						},
					},
				},
			},
			expectErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateTestName(tt.artifact)
			if len(errors) != tt.expectErrors {
				t.Errorf("validateTestName() returned %d errors, expected %d", len(errors), tt.expectErrors)
			}
		})
	}
}

// Tests for extractTestNames function
func TestExtractTestNames(t *testing.T) {
	tests := []struct {
		name     string
		artifact *solc.ForgeArtifact
		want     []string
	}{
		{
			name: "valid test contract",
			artifact: &solc.ForgeArtifact{
				Abi: solc.AbiType{
					Parsed: abi.ABI{
						Methods: map[string]abi.Method{
							"IS_TEST":                  {Name: "IS_TEST"},
							"test_something_succeeds":  {Name: "test_something_succeeds"},
							"test_other_fails":         {Name: "test_other_fails"},
							"not_a_test":               {Name: "not_a_test"},
							"testFuzz_something_works": {Name: "testFuzz_something_works"},
						},
					},
				},
			},
			want: []string{
				"test_something_succeeds",
				"test_other_fails",
				"testFuzz_something_works",
			},
		},
		{
			name: "non-test contract",
			artifact: &solc.ForgeArtifact{
				Abi: solc.AbiType{
					Parsed: abi.ABI{
						Methods: map[string]abi.Method{
							"test_something_succeeds": {Name: "test_something_succeeds"},
							"not_a_test":              {Name: "not_a_test"},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "empty contract",
			artifact: &solc.ForgeArtifact{
				Abi: solc.AbiType{
					Parsed: abi.ABI{
						Methods: map[string]abi.Method{},
					},
				},
			},
			want: nil,
		},
		{
			name: "test contract with no test methods",
			artifact: &solc.ForgeArtifact{
				Abi: solc.AbiType{
					Parsed: abi.ABI{
						Methods: map[string]abi.Method{
							"IS_TEST":        {Name: "IS_TEST"},
							"not_a_test":     {Name: "not_a_test"},
							"another_method": {Name: "another_method"},
						},
					},
				},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTestNames(tt.artifact)
			slices.Sort(got)
			slices.Sort(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractTestNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Tests for checkTestName function
func TestCheckTestName(t *testing.T) {
	tests := []struct {
		name          string
		testName      string
		shouldSucceed bool
	}{
		// Valid test names - Basic patterns
		{"valid basic test succeeds", "test_something_succeeds", true},
		{"valid basic test fails with reason", "test_something_reason_fails", true},
		{"valid basic test reverts with reason", "test_something_reason_reverts", true},
		{"valid basic test works", "test_something_works", true},

		// Valid test names - Fuzz variants
		{"valid fuzz test succeeds", "testFuzz_something_succeeds", true},
		{"valid fuzz test fails with reason", "testFuzz_something_reason_fails", true},
		{"valid fuzz test reverts with reason", "testFuzz_something_reason_reverts", true},
		{"valid fuzz test works", "testFuzz_something_works", true},

		// Valid test names - Diff variants
		{"valid diff test succeeds", "testDiff_something_succeeds", true},
		{"valid diff test fails with reason", "testDiff_something_reason_fails", true},
		{"valid diff test reverts with reason", "testDiff_something_reason_reverts", true},
		{"valid diff test works", "testDiff_something_works", true},

		// Valid test names - Benchmark variants
		{"valid benchmark test", "test_something_benchmark", true},
		{"valid benchmark with number", "test_something_benchmark_123", true},
		{"valid benchmark with large number", "test_something_benchmark_999999", true},
		{"valid benchmark with zero", "test_something_benchmark_0", true},

		// Valid test names - Complex middle parts
		{"valid complex middle part", "test_complexOperation_succeeds", true},
		{"valid multiple word middle", "test_veryComplexOperation_succeeds", true},
		{"valid numbers in middle", "test_operation123_succeeds", true},
		{"valid special case", "test_specialCase_reason_fails", true},

		// Invalid test names - Prefix issues
		{"invalid empty string", "", false},
		{"invalid prefix Test", "Test_something_succeeds", false},
		{"invalid prefix testing", "testing_something_succeeds", false},
		{"invalid prefix testfuzz", "testfuzz_something_succeeds", false},
		{"invalid prefix testdiff", "testdiff_something_succeeds", false},
		{"invalid prefix TEST", "TEST_something_succeeds", false},

		// Invalid test names - Suffix issues
		{"invalid suffix succeed", "test_something_succeed", false},
		{"invalid suffix revert", "test_something_revert", false},
		{"invalid suffix fail", "test_something_fail", false},
		{"invalid suffix work", "test_something_work", false},
		{"invalid suffix benchmarks", "test_something_benchmarks", false},
		{"invalid benchmark suffix text", "test_something_benchmark_abc", false},
		{"invalid benchmark suffix special", "test_something_benchmark_123abc", false},

		// Invalid test names - Case issues
		{"invalid uppercase middle", "test_Something_succeeds", false},
		{"invalid multiple uppercase", "test_SomethingHere_succeeds", false},
		{"invalid all caps middle", "test_SOMETHING_succeeds", false},
		{"invalid mixed case suffix", "test_something_Succeeds", false},

		// Invalid test names - Structure issues
		{"invalid single part", "test", false},
		{"invalid two parts", "test_succeeds", false},
		{"invalid five parts", "test_this_that_those_succeeds", false},
		{"invalid six parts", "test_this_that_those_these_succeeds", false},
		{"invalid failure without reason", "test_something_fails", false},
		{"invalid revert without reason", "test_something_reverts", false},

		// Invalid test names - Special cases
		{"invalid empty parts", "test__succeeds", false},
		{"invalid multiple underscores", "test___succeeds", false},
		{"invalid trailing underscore", "test_something_succeeds_", false},
		{"invalid leading underscore", "_test_something_succeeds", false},
		{"invalid benchmark no number", "test_something_benchmark_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTestName(tt.testName)
			if (err != nil) == tt.shouldSucceed {
				t.Errorf("checkTestName(%q) error = %v, shouldSucceed %v", tt.testName, err, tt.shouldSucceed)
			}
		})
	}
}

// Tests for validateTestStructure function
func TestValidateTestStructure(t *testing.T) {
	// Setup test exclusions
	excludedPaths = []string{"test/excluded/"}
	excludedTests = []string{"ExcludedTest"}

	// Create temporary directory structure for file system tests
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	// Create a source file
	srcFile := filepath.Join(srcDir, "Contract.sol")
	if err := os.WriteFile(srcFile, []byte("// contract"), 0644); err != nil {
		t.Fatalf("Failed to create src file: %v", err)
	}

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	tests := []struct {
		name         string
		artifact     *solc.ForgeArtifact
		expectErrors int
	}{
		{
			name: "valid test structure",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "Contract_Test",
						},
					},
				},
			},
			expectErrors: 1, // Error expected due to contract name/file path validation
		},
		{
			name: "excluded path - no errors",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/excluded/Contract.t.sol": "Contract_Test",
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "invalid contract name pattern",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "InvalidPattern",
						},
					},
				},
			},
			expectErrors: 2, // Multiple errors: src path check + contract name check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateTestStructure(tt.artifact)
			if len(errors) != tt.expectErrors {
				t.Errorf("validateTestStructure() returned %d errors, expected %d", len(errors), tt.expectErrors)
			}
		})
	}
}

// Tests for checkTestStructure function
func TestCheckTestStructure(t *testing.T) {
	// Setup exclusions
	excludedTests = []string{"ExcludedTest"}

	tests := []struct {
		name         string
		artifact     *solc.ForgeArtifact
		expectErrors int
	}{
		{
			name: "valid TestInit pattern",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "Contract_TestInit",
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "valid Harness pattern",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "Contract_Harness",
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "valid Uncategorized test pattern",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "Contract_Uncategorized_Test",
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "valid descriptor harness pattern",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "Contract_Upgrade_Harness",
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "excluded test - no validation",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "ExcludedTest",
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "invalid naming pattern",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "InvalidPattern",
						},
					},
				},
			},
			expectErrors: 1,
		},
		{
			name: "function test with non-existent function",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "Contract_NonExistentFunction_Test",
						},
					},
				},
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := checkTestStructure(tt.artifact)
			if len(errors) != tt.expectErrors {
				t.Errorf("checkTestStructure() returned %d errors, expected %d. Errors: %v", len(errors), tt.expectErrors, errors)
			}
		})
	}
}

// Tests for getCompilationTarget function
func TestGetCompilationTarget(t *testing.T) {
	tests := []struct {
		name             string
		artifact         *solc.ForgeArtifact
		expectError      bool
		expectedPath     string
		expectedContract string
	}{
		{
			name: "single compilation target",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "ContractTest",
						},
					},
				},
			},
			expectError:      false,
			expectedPath:     "test/Contract.t.sol",
			expectedContract: "ContractTest",
		},
		{
			name: "no compilation targets",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{},
					},
				},
			},
			expectError: true,
		},
		{
			name: "multiple compilation targets",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract1.t.sol": "Contract1Test",
							"test/Contract2.t.sol": "Contract2Test",
						},
					},
				},
			},
			expectError:      false,
			expectedPath:     "test/Contract1.t.sol",
			expectedContract: "Contract1Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath, contractName, err := getCompilationTarget(tt.artifact)
			if (err != nil) != tt.expectError {
				t.Errorf("getCompilationTarget() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError {
				if filePath != tt.expectedPath {
					t.Errorf("getCompilationTarget() filePath = %v, expected %v", filePath, tt.expectedPath)
				}
				if contractName != tt.expectedContract {
					t.Errorf("getCompilationTarget() contractName = %v, expected %v", contractName, tt.expectedContract)
				}
			}
		})
	}
}

// Tests for checkSrcPath function
func TestCheckSrcPath(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	testDir := filepath.Join(tempDir, "test")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Create a source file
	srcFile := filepath.Join(srcDir, "Contract.sol")
	if err := os.WriteFile(srcFile, []byte("// contract"), 0644); err != nil {
		t.Fatalf("Failed to create src file: %v", err)
	}

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	tests := []struct {
		name     string
		artifact *solc.ForgeArtifact
		expected bool
	}{
		{
			name: "valid test with existing source",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "ContractTest",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "test with non-existing source",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/NonExistent.t.sol": "NonExistentTest",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "invalid test path (not in test directory)",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"src/Contract.sol": "Contract",
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkSrcPath(tt.artifact)
			if got != tt.expected {
				t.Errorf("checkSrcPath() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// Tests for checkContractNameFilePath function
func TestCheckContractNameFilePath(t *testing.T) {
	tests := []struct {
		name     string
		artifact *solc.ForgeArtifact
		expected bool
	}{
		{
			name: "matching contract and file name",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "Contract_Test",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "non-matching contract and file name",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/Contract.t.sol": "DifferentContract_Test",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "complex path matching",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{
							"test/path/to/MyContract.t.sol": "MyContract_Function_Test",
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkContractNameFilePath(tt.artifact)
			if got != tt.expected {
				t.Errorf("checkContractNameFilePath() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// Tests for loadExclusions function
func TestLoadExclusions(t *testing.T) {
	// Create temporary TOML file
	tomlContent := `[excluded_paths]
src_validation = ["test/legacy/", "test/old/"]
contract_name_validation = ["test/deprecated/"]
function_name_validation = ["test/libs/"]

[excluded_tests]
contracts = ["LegacyTest", "DeprecatedTest"]
`
	tempFile := filepath.Join(t.TempDir(), "test_exclusions.toml")
	if err := os.WriteFile(tempFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Clear existing exclusions
	excludedPaths = nil
	excludedTests = nil

	err := loadExclusions(tempFile)
	if err != nil {
		t.Errorf("loadExclusions() error = %v", err)
	}

	// Check excluded paths were loaded
	expectedPaths := []string{"test/legacy/", "test/old/", "test/deprecated/", "test/libs/"}
	if len(excludedPaths) != len(expectedPaths) {
		t.Errorf("Expected %d excluded paths, got %d", len(expectedPaths), len(excludedPaths))
	}

	// Check excluded tests were loaded
	expectedTests := []string{"LegacyTest", "DeprecatedTest"}
	if !reflect.DeepEqual(excludedTests, expectedTests) {
		t.Errorf("Expected excluded tests %v, got %v", expectedTests, excludedTests)
	}
}

// Tests for isExcluded function
func TestIsExcluded(t *testing.T) {
	// Setup exclusions
	excludedPaths = []string{"test/legacy/", "test/deprecated/"}

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"excluded legacy path", "test/legacy/Contract.t.sol", true},
		{"excluded deprecated path", "test/deprecated/Old.t.sol", true},
		{"non-excluded path", "test/L1/Contract.t.sol", false},
		{"partial match not excluded", "test/leg/Contract.t.sol", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExcluded(tt.filePath)
			if got != tt.expected {
				t.Errorf("isExcluded(%q) = %v, expected %v", tt.filePath, got, tt.expected)
			}
		})
	}
}

// Tests for isExcludedTest function
func TestIsExcludedTest(t *testing.T) {
	// Setup exclusions
	excludedTests = []string{"LegacyTest", "DeprecatedTest"}

	tests := []struct {
		name         string
		contractName string
		expected     bool
	}{
		{"excluded legacy test", "LegacyTest", true},
		{"excluded deprecated test", "DeprecatedTest", true},
		{"non-excluded test", "RegularTest", false},
		{"partial match not excluded", "Legacy", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExcludedTest(tt.contractName)
			if got != tt.expected {
				t.Errorf("isExcludedTest(%q) = %v, expected %v", tt.contractName, got, tt.expected)
			}
		})
	}
}

// Tests for individual validation check functions
func TestCamelCaseCheck(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected bool
	}{
		{"valid single part", []string{"test"}, true},
		{"valid multiple parts", []string{"test", "something", "succeeds"}, true},
		{"invalid uppercase", []string{"Test"}, false},
		{"invalid middle uppercase", []string{"test", "Something", "succeeds"}, false},
		{"empty parts", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checks["camelCase"].check(tt.parts); got != tt.expected {
				t.Errorf("checkCamelCase error for %v = %v, want %v", tt.parts, got, tt.expected)
			}
		})
	}
}

func TestPartsCountCheck(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected bool
	}{
		{"three parts", []string{"test", "something", "succeeds"}, true},
		{"four parts", []string{"test", "something", "reason", "fails"}, true},
		{"too few parts", []string{"test", "fails"}, false},
		{"too many parts", []string{"test", "a", "b", "c", "fails"}, false},
		{"empty parts", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checks["partsCount"].check(tt.parts); got != tt.expected {
				t.Errorf("checkPartsCount error for %v = %v, want %v", tt.parts, got, tt.expected)
			}
		})
	}
}

func TestPrefixCheck(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected bool
	}{
		{"valid test", []string{"test", "something", "succeeds"}, true},
		{"valid testFuzz", []string{"testFuzz", "something", "succeeds"}, true},
		{"valid testDiff", []string{"testDiff", "something", "succeeds"}, true},
		{"invalid prefix", []string{"testing", "something", "succeeds"}, false},
		{"empty parts", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checks["prefix"].check(tt.parts); got != tt.expected {
				t.Errorf("checkPrefix error for %v = %v, want %v", tt.parts, got, tt.expected)
			}
		})
	}
}

func TestSuffixCheck(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected bool
	}{
		{"valid succeeds", []string{"test", "something", "succeeds"}, true},
		{"valid reverts", []string{"test", "something", "reverts"}, true},
		{"valid fails", []string{"test", "something", "fails"}, true},
		{"valid works", []string{"test", "something", "works"}, true},
		{"valid benchmark", []string{"test", "something", "benchmark"}, true},
		{"valid benchmark_num", []string{"test", "something", "benchmark", "123"}, true},
		{"invalid suffix", []string{"test", "something", "invalid"}, false},
		{"invalid benchmark_text", []string{"test", "something", "benchmark", "abc"}, false},
		{"empty parts", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checks["suffix"].check(tt.parts); got != tt.expected {
				t.Errorf("checkSuffix error for %v = %v, want %v", tt.parts, got, tt.expected)
			}
		})
	}
}

func TestFailurePartsCheck(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected bool
	}{
		{"valid failure with reason", []string{"test", "something", "reason", "fails"}, true},
		{"valid failure with reason", []string{"test", "something", "reason", "reverts"}, true},
		{"invalid failure without reason", []string{"test", "something", "fails"}, false},
		{"invalid failure without reason", []string{"test", "something", "reverts"}, false},
		{"valid non-failure with three parts", []string{"test", "something", "succeeds"}, true},
		{"empty parts", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checks["failureParts"].check(tt.parts); got != tt.expected {
				t.Errorf("checkFailureParts error for %v = %v, want %v", tt.parts, got, tt.expected)
			}
		})
	}
}

func TestDoubleUnderscoresCheck(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected bool
	}{
		{"valid parts", []string{"test", "something", "succeeds"}, true},
		{"empty part (double underscore)", []string{"test", "", "succeeds"}, false},
		{"whitespace only part", []string{"test", "   ", "succeeds"}, false},
		{"multiple empty parts", []string{"test", "", "", "succeeds"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checks["doubleUnderscores"].check(tt.parts); got != tt.expected {
				t.Errorf("doubleUnderscores check for %v = %v, want %v", tt.parts, got, tt.expected)
			}
		})
	}
}
