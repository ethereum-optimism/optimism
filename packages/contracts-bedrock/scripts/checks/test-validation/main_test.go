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

func TestProcessFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(tmpFile, []byte(`{"abi":[{"name":"IS_TEST"}],"metadata":{"settings":{"compilationTarget":{"test.sol":"Test"}}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, errors := processFile(tmpFile)
	if len(errors) == 0 {
		t.Error("expected error for invalid test name")
	}
}

func TestValidateTestName(t *testing.T) {
	artifact := &solc.ForgeArtifact{
		Abi: solc.AbiType{
			Parsed: abi.ABI{
				Methods: map[string]abi.Method{
					"IS_TEST":             {Name: "IS_TEST"},
					"test_valid_succeeds": {Name: "test_valid_succeeds"},
					"test_invalid_bad":    {Name: "test_invalid_bad"},
				},
			},
		},
	}

	errors := validateTestName(artifact)
	if len(errors) != 1 {
		t.Errorf("validateTestName() expected 1 error, got %d", len(errors))
	}
}

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

func TestValidateTestStructure(t *testing.T) {
	excludedPaths = []string{"test/excluded/"}
	defer func() { excludedPaths = nil }()
	artifact := &solc.ForgeArtifact{Metadata: solc.ForgeCompilerMetadata{Settings: solc.CompilerSettings{CompilationTarget: map[string]string{"test/excluded/Contract.t.sol": "Contract_Test"}}}}
	if errors := validateTestStructure(artifact); len(errors) != 0 {
		t.Errorf("expected no errors for excluded path, got %d", len(errors))
	}
}

func TestCheckTestStructure(t *testing.T) {
	valid := &solc.ForgeArtifact{Metadata: solc.ForgeCompilerMetadata{Settings: solc.CompilerSettings{CompilationTarget: map[string]string{"test.sol": "Contract_TestInit"}}}}
	invalid := &solc.ForgeArtifact{Metadata: solc.ForgeCompilerMetadata{Settings: solc.CompilerSettings{CompilationTarget: map[string]string{"test.sol": "Invalid_Pattern"}}}}
	if len(checkTestStructure(valid)) > 0 {
		t.Error("valid pattern should not error")
	}
	if len(checkTestStructure(invalid)) == 0 {
		t.Error("invalid pattern should error")
	}
}

func TestGetCompilationTarget(t *testing.T) {
	tests := []struct {
		name         string
		artifact     *solc.ForgeArtifact
		wantPath     string
		wantContract string
		wantErr      bool
	}{
		{
			name: "single target",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{"path/file.sol": "Contract"},
					},
				},
			},
			wantPath:     "path/file.sol",
			wantContract: "Contract",
			wantErr:      false,
		},
		{
			name: "no targets",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotContract, err := getCompilationTarget(tt.artifact)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCompilationTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPath != tt.wantPath || gotContract != tt.wantContract {
				t.Errorf("getCompilationTarget() = (%v, %v), want (%v, %v)", gotPath, gotContract, tt.wantPath, tt.wantContract)
			}
		})
	}
}

func TestCheckSrcPath(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "Contract.sol"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	oldWd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Error(err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	valid := &solc.ForgeArtifact{Metadata: solc.ForgeCompilerMetadata{Settings: solc.CompilerSettings{CompilationTarget: map[string]string{"test/Contract.t.sol": "Contract_Test"}}}}
	invalid := &solc.ForgeArtifact{Metadata: solc.ForgeCompilerMetadata{Settings: solc.CompilerSettings{CompilationTarget: map[string]string{"test/Missing.t.sol": "Missing_Test"}}}}

	if !checkSrcPath(valid) {
		t.Error("valid src path should return true")
	}
	if checkSrcPath(invalid) {
		t.Error("invalid src path should return false")
	}
}

func TestCheckContractNameFilePath(t *testing.T) {
	tests := []struct {
		name     string
		artifact *solc.ForgeArtifact
		want     bool
	}{
		{
			name: "matching name",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{"test/Contract.t.sol": "Contract_Test"},
					},
				},
			},
			want: true,
		},
		{
			name: "non-matching name",
			artifact: &solc.ForgeArtifact{
				Metadata: solc.ForgeCompilerMetadata{
					Settings: solc.CompilerSettings{
						CompilationTarget: map[string]string{"test/Contract.t.sol": "Other_Test"},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkContractNameFilePath(tt.artifact); got != tt.want {
				t.Errorf("checkContractNameFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindArtifactPath(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "forge-artifacts", "Contract.sol"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "forge-artifacts", "Contract.sol", "Contract.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	oldWd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Error(err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if _, err := findArtifactPath("Contract.sol", "Contract"); err != nil {
		t.Error("existing contract should not error")
	}
	if _, err := findArtifactPath("Missing.sol", "Missing"); err == nil {
		t.Error("missing contract should error")
	}
}

func TestCheckFunctionExists(t *testing.T) {
	artifact := &solc.ForgeArtifact{Metadata: solc.ForgeCompilerMetadata{Settings: solc.CompilerSettings{CompilationTarget: map[string]string{"test/Contract.t.sol": "Contract_Test"}}}}
	if !checkFunctionExists(artifact, "constructor") {
		t.Error("constructor should always exist")
	}
	if checkFunctionExists(artifact, "nonexistent") {
		t.Error("nonexistent function should not exist")
	}
}

func TestLoadExclusions(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.toml")
	if err := os.WriteFile(tmpFile, []byte(`[excluded_paths]
src_validation = ["path1"]
[excluded_tests]
contracts = ["Test1"]`), 0644); err != nil {
		t.Fatal(err)
	}

	excludedPaths, excludedTests = nil, nil
	defer func() { excludedPaths, excludedTests = nil, nil }()

	if err := loadExclusions(tmpFile); err != nil {
		t.Error("loadExclusions should not error")
	}
	if len(excludedPaths) != 1 || len(excludedTests) != 1 {
		t.Error("expected 1 excluded path and 1 excluded test")
	}
}

func TestIsExcluded(t *testing.T) {
	excludedPaths = []string{"test/excluded/", "other/path/"}
	defer func() { excludedPaths = nil }()

	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{"excluded path", "test/excluded/file.sol", true},
		{"other excluded", "other/path/file.sol", true},
		{"not excluded", "test/normal/file.sol", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExcluded(tt.filePath); got != tt.want {
				t.Errorf("isExcluded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExcludedTest(t *testing.T) {
	excludedTests = []string{"ExcludedContract", "AnotherExcluded"}
	defer func() { excludedTests = nil }()

	tests := []struct {
		name         string
		contractName string
		want         bool
	}{
		{"excluded contract", "ExcludedContract", true},
		{"another excluded", "AnotherExcluded", true},
		{"not excluded", "NormalContract", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExcludedTest(tt.contractName); got != tt.want {
				t.Errorf("isExcludedTest() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
		{"valid no empty", []string{"test", "something", "succeeds"}, true},
		{"invalid empty part", []string{"test", "", "succeeds"}, false},
		{"invalid multiple empty", []string{"test", "", "", "succeeds"}, false},
		{"empty parts", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checks["doubleUnderscores"].check(tt.parts); got != tt.expected {
				t.Errorf("doubleUnderscores check for %v = %v, want %v", tt.parts, got, tt.expected)
			}
		})
	}
}
