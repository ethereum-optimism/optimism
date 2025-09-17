package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Helper to create test artifact
func createTestArtifact(methods map[string]string, compilationTarget map[string]string) *solc.ForgeArtifact {
	abiMethods := make(map[string]abi.Method)
	for name := range methods {
		abiMethods[name] = abi.Method{Name: name}
	}

	return &solc.ForgeArtifact{
		Abi: solc.AbiType{Parsed: abi.ABI{Methods: abiMethods}},
		Metadata: solc.ForgeCompilerMetadata{
			Settings: solc.CompilerSettings{CompilationTarget: compilationTarget},
		},
	}
}

func TestCheckTestName(t *testing.T) {
	tests := []struct {
		name      string
		testName  string
		wantError bool
	}{
		{"valid test", "test_something_succeeds", false},
		{"valid fuzz test", "testFuzz_something_works", false},
		{"valid failure test", "test_something_reason_fails", false},
		{"invalid uppercase", "test_Something_succeeds", true},
		{"invalid parts count", "test_fails", true},
		{"invalid prefix", "testing_something_succeeds", true},
		{"invalid suffix", "test_something_invalid", true},
		{"failure without reason", "test_something_fails", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTestName(tt.testName)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("checkTestName(%q) error = %v, wantError %v", tt.testName, err, tt.wantError)
			}
		})
	}
}

func TestExtractTestNames(t *testing.T) {
	artifact := createTestArtifact(map[string]string{
		"IS_TEST":                 "",
		"test_something_succeeds": "",
		"testFuzz_other_works":    "",
		"not_a_test":              "",
	}, nil)

	got := extractTestNames(artifact)
	want := []string{"test_something_succeeds", "testFuzz_other_works"}

	if len(got) != len(want) {
		t.Errorf("extractTestNames() = %v, want %v", got, want)
	}
}

func TestCheckTestStructure(t *testing.T) {
	// Setup exclusions for test
	originalExcludedTests := excludedTests
	defer func() { excludedTests = originalExcludedTests }()
	excludedTests = []string{"ExcludedTest"}

	tests := []struct {
		name         string
		contractName string
		wantErrors   int
	}{
		{"valid TestInit", "Contract_TestInit", 0},
		{"valid Harness", "Contract_Harness", 0},
		{"valid Uncategorized", "Contract_Uncategorized_Test", 0},
		{"excluded test", "ExcludedTest", 0},
		{"invalid pattern", "InvalidPattern", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := createTestArtifact(nil, map[string]string{
				"test/Contract.t.sol": tt.contractName,
			})

			errors := checkTestStructure(artifact)
			if len(errors) != tt.wantErrors {
				t.Errorf("checkTestStructure() errors = %d, want %d", len(errors), tt.wantErrors)
			}
		})
	}
}

func TestLoadExclusions(t *testing.T) {
	// Create temp TOML file
	content := `[excluded_paths]
src_validation = ["test/legacy/"]
contract_name_validation = ["test/deprecated/"]
function_name_validation = ["test/libs/"]

[excluded_tests]
contracts = ["LegacyTest"]`

	tempFile := filepath.Join(t.TempDir(), "test.toml")
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Clear and load
	excludedPaths = nil
	excludedTests = nil

	if err := loadExclusions(tempFile); err != nil {
		t.Errorf("loadExclusions() error = %v", err)
	}

	if len(excludedPaths) != 3 {
		t.Errorf("Expected 3 excluded paths, got %d", len(excludedPaths))
	}
	if len(excludedTests) != 1 || excludedTests[0] != "LegacyTest" {
		t.Errorf("Expected [LegacyTest], got %v", excludedTests)
	}
}

func TestIsExcluded(t *testing.T) {
	excludedPaths = []string{"test/legacy/", "test/deprecated/"}

	tests := []struct {
		path string
		want bool
	}{
		{"test/legacy/Contract.t.sol", true},
		{"test/L1/Contract.t.sol", false},
	}

	for _, tt := range tests {
		if got := isExcluded(tt.path); got != tt.want {
			t.Errorf("isExcluded(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsExcludedTest(t *testing.T) {
	excludedTests = []string{"LegacyTest"}

	tests := []struct {
		name string
		want bool
	}{
		{"LegacyTest", true},
		{"RegularTest", false},
	}

	for _, tt := range tests {
		if got := isExcludedTest(tt.name); got != tt.want {
			t.Errorf("isExcludedTest(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// Test the validation check functions
func TestValidationChecks(t *testing.T) {
	tests := []struct {
		checkName string
		parts     []string
		want      bool
	}{
		// camelCase tests
		{"camelCase", []string{"test", "something"}, true},
		{"camelCase", []string{"Test", "something"}, false},

		// partsCount tests
		{"partsCount", []string{"test", "something", "succeeds"}, true},
		{"partsCount", []string{"test", "fails"}, false},

		// prefix tests
		{"prefix", []string{"test", "something", "succeeds"}, true},
		{"prefix", []string{"testing", "something", "succeeds"}, false},

		// suffix tests
		{"suffix", []string{"test", "something", "succeeds"}, true},
		{"suffix", []string{"test", "something", "invalid"}, false},

		// failureParts tests
		{"failureParts", []string{"test", "something", "reason", "fails"}, true},
		{"failureParts", []string{"test", "something", "fails"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.checkName+"_"+tt.parts[len(tt.parts)-1], func(t *testing.T) {
			if got := checks[tt.checkName].check(tt.parts); got != tt.want {
				t.Errorf("%s check(%v) = %v, want %v", tt.checkName, tt.parts, got, tt.want)
			}
		})
	}
}
