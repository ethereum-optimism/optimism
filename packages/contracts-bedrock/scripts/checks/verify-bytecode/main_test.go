package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Helper to create a basic ForgeArtifact for testing
func newTestArtifact(opts ...func(*solc.ForgeArtifact)) *solc.ForgeArtifact {
	artifact := &solc.ForgeArtifact{
		Abi:              solc.AbiType{}, // Usually not needed for bytecode verification tests
		Bytecode:         solc.CompilerOutputBytecode{Object: "0x"},
		DeployedBytecode: solc.CompilerOutputBytecode{Object: "0x"},
		Ast:              solc.Ast{Nodes: []solc.AstNode{}},
	}
	for _, opt := range opts {
		opt(artifact)
	}
	return artifact
}

// Option to set deployed bytecode
func withDeployedBytecode(code string) func(*solc.ForgeArtifact) {
	return func(a *solc.ForgeArtifact) {
		a.DeployedBytecode.Object = code
	}
}

// Option to set creation bytecode
func withCreationBytecode(code string) func(*solc.ForgeArtifact) {
	return func(a *solc.ForgeArtifact) {
		a.Bytecode.Object = code
	}
}

// Option to add immutable references
func withImmutableRefs(refs map[string][]solc.ImmutableReference) func(*solc.ForgeArtifact) {
	return func(a *solc.ForgeArtifact) {
		if a.DeployedBytecode.ImmutableReferences == nil {
			a.DeployedBytecode.ImmutableReferences = make(map[string][]solc.ImmutableReference)
		}
		for k, v := range refs {
			a.DeployedBytecode.ImmutableReferences[k] = v
		}
	}
}

// Option to add AST nodes
func withAstNodes(nodes []solc.AstNode) func(*solc.ForgeArtifact) {
	return func(a *solc.ForgeArtifact) {
		a.Ast.Nodes = nodes
	}
}

// Helper to create AST nodes for immutable tests
func createTestAstNodes() []solc.AstNode {
	return []solc.AstNode{
		{ // Contract Definition
			Id:       10,
			NodeType: "ContractDefinition",
			Name:     "MyContract",
			Nodes: []solc.AstNode{
				{ // State Variable 1 (immutable)
					Id:               5,
					NodeType:         "VariableDeclaration",
					Name:             "IMMUTABLE_VAR_1",
					StateVariable:    true,
					Mutability:       "immutable",
					Constant:         false,
					TypeDescriptions: &solc.AstTypeDescriptions{TypeString: "uint256"},
				},
				{ // State Variable 2 (regular)
					Id:               6,
					NodeType:         "VariableDeclaration",
					Name:             "regularVar",
					StateVariable:    true,
					TypeDescriptions: &solc.AstTypeDescriptions{TypeString: "bool"},
				},
				{ // Function Definition
					Id:       8,
					NodeType: "FunctionDefinition",
					Name:     "doSomething",
					Body: &solc.AstBlock{
						NodeType: "Block",
						Id:       9,
						Statements: []solc.AstNode{
							{ // Local Variable (shouldn't be found by ID 5)
								Id:       7,
								NodeType: "ExpressionStatement",
								Src:      "placeholder;",
							},
						},
					},
				},
				{ // State Variable 3 (immutable, nested struct type not important for name lookup)
					Id:               15,
					NodeType:         "VariableDeclaration",
					Name:             "IMMUTABLE_VAR_2",
					StateVariable:    true,
					Mutability:       "immutable",
					TypeDescriptions: &solc.AstTypeDescriptions{TypeString: "struct MyStruct"},
				},
			},
		},
		{ // Another top-level node (e.g., ImportDirective, ErrorDefinition)
			Id:       11,
			NodeType: "ImportDirective",
		},
		{ // Struct Definition (containing a node with ID 5, but wrong type)
			Id:       12,
			NodeType: "StructDefinition",
			Name:     "MyStruct",
			Nodes: []solc.AstNode{
				{
					Id:               5, // Duplicate ID, but wrong node type
					NodeType:         "MemberAccess",
					Name:             "", // Not a declaration name
					TypeDescriptions: &solc.AstTypeDescriptions{TypeString: "uint"},
				},
			},
		},
	}
}

func TestGetImmutableName(t *testing.T) {
	astNodes := createTestAstNodes()
	artifact := newTestArtifact(withAstNodes(astNodes))

	tests := []struct {
		name     string
		refKey   string
		artifact *solc.ForgeArtifact
		wantName string
	}{
		{
			name:     "Valid ID simple",
			refKey:   "5",
			artifact: artifact,
			wantName: "IMMUTABLE_VAR_1",
		},
		{
			name:     "Valid ID with type prefix",
			refKey:   "t_struct:MyStruct:15",
			artifact: artifact,
			wantName: "IMMUTABLE_VAR_2",
		},
		{
			name:     "ID exists but not VariableDeclaration",
			refKey:   "7",
			artifact: artifact,
			wantName: "",
		},
		{
			name:     "ID not found",
			refKey:   "999",
			artifact: artifact,
			wantName: "",
		},
		{
			name:     "Invalid refKey format",
			refKey:   "invalid-key",
			artifact: artifact,
			wantName: "",
		},
		{
			name:     "Nil artifact",
			refKey:   "5",
			artifact: nil,
			wantName: "",
		},
		{
			name:     "Artifact with no AST",
			refKey:   "5",
			artifact: newTestArtifact(),
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We don't directly test findAstNodeNameByID as it's an internal helper.
			// Its behavior is tested via getImmutableName.
			gotName := getImmutableName(tt.artifact, tt.refKey)
			assert.Equal(t, tt.wantName, gotName)
		})
	}
}

func TestCompareBytecode_ExactMatch(t *testing.T) {
	artifact := newTestArtifact() // No immutables needed
	expected := "0x12345678"
	actual := "0x12345678"

	diffs, infos, err := compareBytecode(artifact, true, expected, actual)
	require.NoError(t, err)
	assert.Empty(t, diffs, "Should be no differences")
	assert.Empty(t, infos, "Should be no immutable info")
}

func TestCompareBytecode_SimpleMismatch(t *testing.T) {
	artifact := newTestArtifact()
	expected := "0x12345678"
	actual := "0x1234ff78" // Mismatch at byte 2 (0-indexed)

	diffs, infos, err := compareBytecode(artifact, true, expected, actual)
	require.NoError(t, err)
	assert.Empty(t, infos)
	require.Len(t, diffs, 1)

	diff := diffs[0]
	assert.Equal(t, 2, diff.Start)
	assert.Equal(t, 1, diff.Length)
	assert.Equal(t, "56", diff.Expected)
	assert.Equal(t, "ff", diff.Actual)
	assert.False(t, diff.InImmutable)
	assert.Equal(t, "", diff.ImmutableName)
}

func TestCompareBytecode_DifferentLengths(t *testing.T) {
	artifact := newTestArtifact()
	expected := "0x12345678"
	actualShort := "0x123456"
	actualLong := "0x1234567890"

	_, _, err := compareBytecode(artifact, true, expected, actualShort)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytecode length mismatch")

	_, _, err = compareBytecode(artifact, true, expected, actualLong)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytecode length mismatch")

	_, _, err = compareBytecode(artifact, true, actualShort, expected)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytecode length mismatch")
}

func TestCompareBytecode_PrefixHandling(t *testing.T) {
	artifact := newTestArtifact()
	expected := "0x1234"
	actualNoPrefix := "1234"
	actualWithPrefix := "0x1234"

	// Expected has prefix, actual does not
	diffs, infos, err := compareBytecode(artifact, true, expected, actualNoPrefix)
	require.NoError(t, err)
	assert.Empty(t, diffs)
	assert.Empty(t, infos)

	// Expected does not have prefix, actual does
	diffs, infos, err = compareBytecode(artifact, true, actualNoPrefix, actualWithPrefix)
	require.NoError(t, err)
	assert.Empty(t, diffs)
	assert.Empty(t, infos)
}

func TestCompareBytecode_EmptyBytecode(t *testing.T) {
	artifact := newTestArtifact()

	// Both empty with prefix
	diffs, infos, err := compareBytecode(artifact, true, "0x", "0x")
	require.NoError(t, err)
	assert.Empty(t, diffs)
	assert.Empty(t, infos)

	// Both empty without prefix
	diffs, infos, err = compareBytecode(artifact, true, "", "")
	require.NoError(t, err)
	assert.Empty(t, diffs)
	assert.Empty(t, infos)

	// One empty, one not (should fail length check)
	_, _, err = compareBytecode(artifact, true, "0x12", "0x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytecode length mismatch")

	_, _, err = compareBytecode(artifact, true, "", "12")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytecode length mismatch")
}

func TestCompareBytecode_InvalidHex(t *testing.T) {
	artifact := newTestArtifact()
	valid := "0x1234"
	invalid := "0x123G" // Invalid character 'G'
	oddLen := "0x123"   // Odd length

	_, _, err := compareBytecode(artifact, true, invalid, valid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode expected bytecode")

	_, _, err = compareBytecode(artifact, true, valid, invalid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode actual bytecode")

	_, _, err = compareBytecode(artifact, true, oddLen, valid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expected bytecode hex length")

	_, _, err = compareBytecode(artifact, true, valid, oddLen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode actual bytecode")
	assert.Contains(t, err.Error(), "odd length hex string") // Specific error from hex pkg
}

func TestCompareBytecode_Immutables(t *testing.T) {
	tests := []struct {
		name       string
		expected   string
		actual     string
		checkImmut bool
		immutables solc.ImmutableReferences
		wantDiffs  []BytecodeDifference
		wantInfos  []ImmutableValueInfo
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "Match",
			expected:   "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			actual:     "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			checkImmut: true,
			immutables: solc.ImmutableReferences{
				"5": {
					{Start: 2, Length: 3},
				},
				"15": {
					{Start: 8, Length: 2},
					{Start: 12, Length: 1},
				},
			},
			wantDiffs: []BytecodeDifference{},
			wantInfos: []ImmutableValueInfo{
				{Name: "IMMUTABLE_VAR_1", Offset: 2, Length: 3, Value: "0xaaaa00"},
				{Name: "IMMUTABLE_VAR_2", Offset: 8, Length: 2, Value: "0xbbbb"},
				{Name: "IMMUTABLE_VAR_2", Offset: 12, Length: 1, Value: "0xcc"},
			},
			wantErr: false,
		},
		{
			name:       "Diff inside immutable only (checkImmut=false)",
			expected:   "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			actual:     "0xAAAAAAAA112233BBBBBBBBCCCCDDDDDD",
			checkImmut: false,
			immutables: solc.ImmutableReferences{},
			wantDiffs: []BytecodeDifference{
				{Start: 4, Length: 3, Expected: "000000", Actual: "112233", InImmutable: false, ImmutableName: ""},
			},
			wantInfos: []ImmutableValueInfo{},
			wantErr:   false,
		},
		{
			name:       "Diff outside immutable only",
			expected:   "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			actual:     "0xAAAAAAAA000000BBBBBBBBCCCCDDDDFF",
			checkImmut: true,
			immutables: solc.ImmutableReferences{
				"5": {
					{Start: 2, Length: 3},
				},
				"15": {
					{Start: 8, Length: 2},
					{Start: 12, Length: 1},
				},
			},
			wantDiffs: []BytecodeDifference{
				{Start: 15, Length: 1, Expected: "dd", Actual: "ff", InImmutable: false, ImmutableName: ""},
			},
			wantInfos: []ImmutableValueInfo{
				{Name: "IMMUTABLE_VAR_1", Offset: 2, Length: 3, Value: "0xaaaa00"},
				{Name: "IMMUTABLE_VAR_2", Offset: 8, Length: 2, Value: "0xbbbb"},
				{Name: "IMMUTABLE_VAR_2", Offset: 12, Length: 1, Value: "0xcc"},
			},
			wantErr: false,
		},
		{
			name:       "Diffs inside and outside immutable",
			expected:   "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			actual:     "0xAAAA1122330000BBBBBBBBCCCCDDDDFF",
			checkImmut: true,
			immutables: solc.ImmutableReferences{
				"5": {
					{Start: 2, Length: 3},
				},
				"15": {
					{Start: 8, Length: 2},
					{Start: 12, Length: 1},
				},
			},
			wantDiffs: []BytecodeDifference{
				{Start: 2, Length: 3, Expected: "aaaa00", Actual: "112233", InImmutable: true, ImmutableName: "IMMUTABLE_VAR_1"},
				{Start: 15, Length: 1, Expected: "dd", Actual: "ff", InImmutable: false, ImmutableName: ""},
			},
			wantInfos: []ImmutableValueInfo{
				{Name: "IMMUTABLE_VAR_1", Offset: 2, Length: 3, Value: "0x112233"},
				{Name: "IMMUTABLE_VAR_2", Offset: 8, Length: 2, Value: "0xbbbb"},
				{Name: "IMMUTABLE_VAR_2", Offset: 12, Length: 1, Value: "0xcc"},
			},
			wantErr: false,
		},
		{
			name:       "Diff spanning immutable boundary",
			expected:   "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			actual:     "0xAAAAAAAA112200BBBBBBBBCCCCDDDDDD",
			checkImmut: true,
			immutables: solc.ImmutableReferences{
				"5": {
					{Start: 2, Length: 3},
				},
				"15": {
					{Start: 8, Length: 2},
					{Start: 12, Length: 1},
				},
			},
			wantDiffs: []BytecodeDifference{
				{Start: 4, Length: 1, Expected: "00", Actual: "11", InImmutable: true, ImmutableName: "IMMUTABLE_VAR_1"},
				{Start: 5, Length: 1, Expected: "00", Actual: "22", InImmutable: false, ImmutableName: ""},
			},
			wantInfos: []ImmutableValueInfo{
				{Name: "IMMUTABLE_VAR_1", Offset: 2, Length: 3, Value: "0xaaaa11"},
				{Name: "IMMUTABLE_VAR_2", Offset: 8, Length: 2, Value: "0xbbbb"},
				{Name: "IMMUTABLE_VAR_2", Offset: 12, Length: 1, Value: "0xcc"},
			},
			wantErr: false,
		},
		{
			name:       "Immutable ref out of bounds for actual bytecode",
			expected:   "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			actual:     "0xAAAAAAAA000000BBBBBBBBCCCC",
			checkImmut: true,
			immutables: solc.ImmutableReferences{
				"5": {
					{Start: 2, Length: 3},
				},
				"15": {
					{Start: 8, Length: 2},
					{Start: 12, Length: 1},
				},
			},
			wantDiffs:  nil,
			wantInfos:  nil,
			wantErr:    true,
			wantErrMsg: "bytecode length mismatch",
		},
		{
			name:       "Immutable ref has invalid length (zero)",
			expected:   "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			actual:     "0xAAAAAAAA000000BBBBBBBBCCCCDDDDDD",
			checkImmut: true,
			immutables: solc.ImmutableReferences{
				"5": {
					{Start: 2, Length: 0},
				},
			},
			wantDiffs:  nil,
			wantInfos:  nil,
			wantErr:    true,
			wantErrMsg: "immutable 'IMMUTABLE_VAR_1' location (offset 2, length 0) has invalid length 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			astNodes := createTestAstNodes()
			art := newTestArtifact(withAstNodes(astNodes), withImmutableRefs(tt.immutables))
			diffs, infos, err := compareBytecode(art, tt.checkImmut, tt.expected, tt.actual)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDiffs, diffs, "Differences mismatch")
				assert.ElementsMatch(t, tt.wantInfos, infos, "Immutable infos mismatch")
			}
		})
	}
}

func TestCompareBytecode_DifferenceGrouping(t *testing.T) {
	artifact := newTestArtifact()
	expected := "0x112233445566"
	actual := "0x11aabbcc5566" // Differs @ 1,2,3 (0x223344 -> 0xaabbcc)

	diffs, infos, err := compareBytecode(artifact, true, expected, actual)
	require.NoError(t, err)
	assert.Empty(t, infos)
	require.Len(t, diffs, 1, "Differences should be grouped")

	diff := diffs[0]
	assert.Equal(t, 1, diff.Start)
	assert.Equal(t, 3, diff.Length)
	assert.Equal(t, "223344", diff.Expected)
	assert.Equal(t, "aabbcc", diff.Actual)
	assert.False(t, diff.InImmutable)
}

func TestCategorizeDifferences(t *testing.T) {
	tests := []struct {
		name          string
		resultType    VerificationType
		allDiffs      []BytecodeDifference
		wantCodeDiffs []BytecodeDifference
		wantImmDiffs  []BytecodeDifference
		wantHasCode   bool
	}{
		{
			name:          "No diffs",
			resultType:    DeployedContract,
			allDiffs:      []BytecodeDifference{},
			wantCodeDiffs: []BytecodeDifference{},
			wantImmDiffs:  []BytecodeDifference{},
			wantHasCode:   false,
		},
		{
			name:       "Only code diffs",
			resultType: DeployedContract,
			allDiffs: []BytecodeDifference{
				{Start: 1, Length: 1, Expected: "aa", Actual: "bb", InImmutable: false},
				{Start: 5, Length: 2, Expected: "cccc", Actual: "dddd", InImmutable: false},
			},
			wantCodeDiffs: []BytecodeDifference{
				{Start: 1, Length: 1, Expected: "aa", Actual: "bb", InImmutable: false},
				{Start: 5, Length: 2, Expected: "cccc", Actual: "dddd", InImmutable: false},
			},
			wantImmDiffs: []BytecodeDifference{},
			wantHasCode:  true,
		},
		{
			name:       "Only immutable diffs (DeployedContract)",
			resultType: DeployedContract,
			allDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
				{Start: 20, Length: 1, Expected: "ee", Actual: "ff", InImmutable: true, ImmutableName: "VarB"},
			},
			wantCodeDiffs: []BytecodeDifference{},
			wantImmDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
				{Start: 20, Length: 1, Expected: "ee", Actual: "ff", InImmutable: true, ImmutableName: "VarB"},
			},
			wantHasCode: false,
		},
		{
			name:       "Mixed diffs (DeployedContract)",
			resultType: DeployedContract,
			allDiffs: []BytecodeDifference{
				{Start: 1, Length: 1, Expected: "aa", Actual: "bb", InImmutable: false},
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
				{Start: 5, Length: 2, Expected: "cccc", Actual: "dddd", InImmutable: false},
				{Start: 20, Length: 1, Expected: "ee", Actual: "ff", InImmutable: true, ImmutableName: "VarB"},
			},
			wantCodeDiffs: []BytecodeDifference{
				{Start: 1, Length: 1, Expected: "aa", Actual: "bb", InImmutable: false},
				{Start: 5, Length: 2, Expected: "cccc", Actual: "dddd", InImmutable: false},
			},
			wantImmDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
				{Start: 20, Length: 1, Expected: "ee", Actual: "ff", InImmutable: true, ImmutableName: "VarB"},
			},
			wantHasCode: true,
		},
		{
			name:       "Only immutable diffs (Implementation)",
			resultType: Implementation, // Also checks immutables
			allDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
			},
			wantCodeDiffs: []BytecodeDifference{},
			wantImmDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
			},
			wantHasCode: false,
		},
		{
			name:       "Only immutable diffs (OPContractsManager)",
			resultType: OPContractsManager, // Also checks immutables
			allDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
			},
			wantCodeDiffs: []BytecodeDifference{},
			wantImmDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
			},
			wantHasCode: false,
		},
		{
			name:       "Only immutable diffs (Blueprint)",
			resultType: Blueprint, // Does NOT check immutables
			allDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
			},
			wantCodeDiffs: []BytecodeDifference{
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
			}, // Immutable diffs treated as code diffs
			wantImmDiffs: []BytecodeDifference{},
			wantHasCode:  true,
		},
		{
			name:       "Mixed diffs (SplitBlueprintPart1)",
			resultType: SplitBlueprintPart1, // Does NOT check immutables
			allDiffs: []BytecodeDifference{
				{Start: 1, Length: 1, Expected: "aa", Actual: "bb", InImmutable: false},
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
			},
			wantCodeDiffs: []BytecodeDifference{
				{Start: 1, Length: 1, Expected: "aa", Actual: "bb", InImmutable: false},
				{Start: 10, Length: 4, Expected: "00000000", Actual: "11111111", InImmutable: true, ImmutableName: "VarA"},
			}, // Immutable diffs treated as code diffs
			wantImmDiffs: []BytecodeDifference{},
			wantHasCode:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &VerificationResult{
				Type:        tt.resultType,
				Differences: tt.allDiffs,
			}
			gotCodeDiffs, gotImmDiffs := categorizeDifferences(result)
			assert.Equal(t, tt.wantCodeDiffs, gotCodeDiffs, "Code differences mismatch")
			assert.Equal(t, tt.wantImmDiffs, gotImmDiffs, "Immutable differences mismatch")

			// Test hasCodeDifferences as well
			assert.Equal(t, tt.wantHasCode, hasCodeDifferences(result), "hasCodeDifferences mismatch")
		})
	}
}

// --- Tests for verify*Logic functions ---
// Note: These tests focus on the orchestration logic, assuming dependencies like
// getOnchainBytecode, ccom.ReadForgeArtifact, and compareBytecode work correctly (or are mocked implicitly).
// They primarily check the structure and fields of the returned VerificationResult.

// Mock implementations (replace with actual mocking library if needed)
var mockBytecodeStore = make(map[string]string)
var mockArtifactStore = make(map[string]*solc.ForgeArtifact)
var mockReadArtifactError error
var mockGetBytecodeError error

// setupMocks resets mock state and installs mock implementations for the current test.
func setupMocks(t *testing.T) {
	t.Helper()

	mockBytecodeStore = make(map[string]string)
	mockArtifactStore = make(map[string]*solc.ForgeArtifact) // Although ReadForgeArtifact isn't directly mocked here now
	mockReadArtifactError = nil
	mockGetBytecodeError = nil

	// Keep track of the original implementation
	originalGetBytecode := getOnchainBytecodeImpl

	// Define the mock implementation
	getOnchainBytecodeImpl = func(client *ethclient.Client, addr common.Address) (string, error) {
		if mockGetBytecodeError != nil {
			// Check if the error is specific to this address (optional enhancement)
			// For now, any error applies globally.
			return "", mockGetBytecodeError
		}
		code, ok := mockBytecodeStore[addr.Hex()]
		if !ok {
			// Return 0x for unknown addresses to simulate no code found, common case
			return "0x", fmt.Errorf("no code found at address (mock)")
		}
		return code, nil
	}

	// Use t.Cleanup to restore the original implementation after the test
	t.Cleanup(func() {
		getOnchainBytecodeImpl = originalGetBytecode
	})
}

func TestVerifyDeployedContractLogic(t *testing.T) {
	artifactPath := "/mock/MyContract.json" // Path is still used for metadata
	contractName := "MyContract"
	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	expectedCode := "0x6080604052348015600f57600080fd5b50604051602080606f8339810160405280600a5f5260005f60005f5151f3fe"
	actualCodeMatch := expectedCode
	actualCodeMismatch := "0x6080604052348015600f57600080fd5b50604051602080606f8339810160405280ffff5f5260005f60005f5151f3fe" // Mismatch '600a' -> 'ffff' (same length)

	tests := []struct {
		name           string
		artifact       *solc.ForgeArtifact
		mockSetup      func()
		wantErr        bool
		wantErrContent string
		wantDiffs      bool
		wantImmutables bool
	}{
		{
			name:     "Match",
			artifact: newTestArtifact(withDeployedBytecode(expectedCode)),
			mockSetup: func() {
				mockBytecodeStore[address.Hex()] = actualCodeMatch
			},
			wantErr:        false,
			wantDiffs:      false,
			wantImmutables: false,
		},
		{
			name:     "Mismatch",
			artifact: newTestArtifact(withDeployedBytecode(expectedCode)),
			mockSetup: func() {
				mockBytecodeStore[address.Hex()] = actualCodeMismatch
			},
			wantErr:        false,
			wantDiffs:      true,
			wantImmutables: false,
		},
		{
			name:     "Get bytecode error",
			artifact: newTestArtifact(withDeployedBytecode(expectedCode)),
			mockSetup: func() {
				mockGetBytecodeError = fmt.Errorf("rpc is down")
			},
			wantErr:        true,
			wantErrContent: "getting onchain bytecode: rpc is down",
		},
		{
			name:     "No code at address",
			artifact: newTestArtifact(withDeployedBytecode(expectedCode)),
			mockSetup: func() {
				// No entry in mockBytecodeStore triggers the mock's error
			},
			wantErr:        true,
			wantErrContent: "no code found at address (mock)",
		},
		{
			name: "Match with immutables",
			artifact: newTestArtifact(
				withDeployedBytecode("0xAAAABBBBCCCCDDDD"), // Expected
				withAstNodes(createTestAstNodes()),
				withImmutableRefs(solc.ImmutableReferences{
					"5": {{Start: 2, Length: 2}}, // BBBBB
				}),
			),
			mockSetup: func() {
				mockBytecodeStore[address.Hex()] = "0xAAAA1122CCCCDDDD" // Actual (diff only in immutable)
			},
			wantErr:        false,
			wantDiffs:      true, // compareBytecode returns diffs, but categorizeDifferences handles it
			wantImmutables: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.mockSetup()

			// Pass nil client because getOnchainBytecodeImpl is mocked
			result := verifyDeployedContractLogic(nil, tt.artifact, contractName, artifactPath, address)

			if tt.wantErr {
				require.Error(t, result.ProcessError)
				if tt.wantErrContent != "" {
					assert.Contains(t, result.ProcessError.Error(), tt.wantErrContent)
				}
			} else {
				require.NoError(t, result.ProcessError)
				assert.Equal(t, DeployedContract, result.Type)
				assert.Equal(t, address.Hex(), result.Address)
				assert.Equal(t, artifactPath, result.ArtifactPath)
				assert.Equal(t, contractName, result.ContractName)
				if tt.wantDiffs {
					assert.NotEmpty(t, result.Differences)
				} else {
					assert.Empty(t, result.Differences)
				}
				if tt.wantImmutables {
					assert.NotEmpty(t, result.ImmutableInfos)
				} else {
					assert.Empty(t, result.ImmutableInfos)
				}
				// Check categorization for the immutable case
				if tt.name == "Match with immutables" {
					assert.False(t, hasCodeDifferences(result), "Should have no *code* differences")
				}
			}
		})
	}
}

func TestVerifyBlueprintLogic(t *testing.T) {
	targetArtifactPath := "/mock/TargetContract.json"
	targetContractName := "TargetContract"
	blueprintAddress := common.HexToAddress("0xABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCD")
	blueprintFieldName := "TheBlueprint"
	creationCode := "608060405234801561001057600080fd5b5061015ff3"
	expectedBlueprintCode := blueprintPreamble + creationCode
	actualCodeMatch := expectedBlueprintCode
	actualCodeMismatch := blueprintPreamble + "ffffff" + creationCode[6:] // Mismatch after preamble

	tests := []struct {
		name           string
		artifact       *solc.ForgeArtifact
		mockSetup      func()
		wantErr        bool
		wantErrContent string
		wantDiffs      bool
	}{
		{
			name:     "Match",
			artifact: newTestArtifact(withCreationBytecode("0x" + creationCode)),
			mockSetup: func() {
				mockBytecodeStore[blueprintAddress.Hex()] = actualCodeMatch
			},
			wantErr:   false,
			wantDiffs: false,
		},
		{
			name:     "Mismatch",
			artifact: newTestArtifact(withCreationBytecode("0x" + creationCode)),
			mockSetup: func() {
				mockBytecodeStore[blueprintAddress.Hex()] = actualCodeMismatch
			},
			wantErr:   false,
			wantDiffs: true,
		},
		{
			name:     "Get blueprint bytecode error",
			artifact: newTestArtifact(withCreationBytecode("0x" + creationCode)),
			mockSetup: func() {
				mockGetBytecodeError = fmt.Errorf("rpc is down")
			},
			wantErr:        true,
			wantErrContent: "getting onchain bytecode for blueprint",
		},
		{
			name:     "No code at address",
			artifact: newTestArtifact(withCreationBytecode("0x" + creationCode)),
			mockSetup: func() {
				// No entry in mockBytecodeStore
			},
			wantErr:        true,
			wantErrContent: "no code found at address (mock)",
		},
		{
			name:           "No creation code in artifact",
			artifact:       newTestArtifact(withCreationBytecode("0x")), // Empty creation code
			mockSetup:      func() {},                                   // Bytecode doesn't matter here
			wantErr:        true,
			wantErrContent: "no creation bytecode found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.mockSetup()

			// Pass nil client because getOnchainBytecodeImpl is mocked
			result := verifyBlueprintLogic(nil, tt.artifact, targetContractName, targetArtifactPath, blueprintAddress, blueprintFieldName)

			if tt.wantErr {
				require.Error(t, result.ProcessError)
				if tt.wantErrContent != "" {
					assert.Contains(t, result.ProcessError.Error(), tt.wantErrContent)
				}
			} else {
				require.NoError(t, result.ProcessError)
				assert.Equal(t, Blueprint, result.Type)
				assert.Equal(t, blueprintAddress.Hex(), result.Address)
				assert.Equal(t, targetArtifactPath, result.ArtifactPath)
				assert.Equal(t, blueprintFieldName, result.FieldName)
				assert.Equal(t, targetContractName, result.TargetContract)
				assert.Equal(t, fmt.Sprintf("Blueprint for %s", targetContractName), result.ContractName)
				if tt.wantDiffs {
					assert.NotEmpty(t, result.Differences)
					assert.True(t, hasCodeDifferences(result)) // Any blueprint diff is a code diff
				} else {
					assert.Empty(t, result.Differences)
				}
			}
		})
	}
}

func TestVerifySplitBlueprintLogic(t *testing.T) {
	targetArtifactPath := "/mock/SplitTarget.json"
	targetContractName := "SplitTarget"
	address1 := common.HexToAddress("0xAAAAAAAAAAAAAAAABBBBBBBBBBBBBBBBBBBB")
	address2 := common.HexToAddress("0xCCCCCCCCCCCCCCCCDDDDDDDDDDDDDDDDDDDD")
	fieldName1 := "SplitBP1"
	fieldName2 := "SplitBP2"

	// Create creation code longer than maxInitCodeSize
	part1Hex := strings.Repeat("11", maxInitCodeSize)
	part2Hex := strings.Repeat("22", 10)
	fullCreationCode := part1Hex + part2Hex
	artifactLong := newTestArtifact(withCreationBytecode("0x" + fullCreationCode))
	expectedBP1 := blueprintPreamble + part1Hex
	expectedBP2 := blueprintPreamble + part2Hex

	// Create creation code shorter than maxInitCodeSize
	shortCodeHex := strings.Repeat("33", 100)
	artifactShort := newTestArtifact(withCreationBytecode("0x" + shortCodeHex))
	expectedShortBP1 := blueprintPreamble + shortCodeHex
	expectedShortBP2 := blueprintPreamble // Empty part 2

	tests := []struct {
		name         string
		artifact     *solc.ForgeArtifact
		mockSetup    func()
		wantErr1     bool
		wantErr2     bool
		wantErr1Cont string
		wantErr2Cont string
		wantDiffs1   bool
		wantDiffs2   bool
	}{
		{
			name:     "Match Long Code",
			artifact: artifactLong,
			mockSetup: func() {
				mockBytecodeStore[address1.Hex()] = expectedBP1
				mockBytecodeStore[address2.Hex()] = expectedBP2
			},
			wantErr1:   false,
			wantErr2:   false,
			wantDiffs1: false,
			wantDiffs2: false,
		},
		{
			name:     "Match Short Code (part 2 is empty)",
			artifact: artifactShort,
			mockSetup: func() {
				mockBytecodeStore[address1.Hex()] = expectedShortBP1
				mockBytecodeStore[address2.Hex()] = expectedShortBP2 // Expect preamble only for empty code
			},
			wantErr1:   false,
			wantErr2:   false,
			wantDiffs1: false,
			wantDiffs2: false,
		},
		{
			name:     "Mismatch Part 1",
			artifact: artifactLong,
			mockSetup: func() {
				mockBytecodeStore[address1.Hex()] = blueprintPreamble + "ff" + part1Hex[2:]
				mockBytecodeStore[address2.Hex()] = expectedBP2
			},
			wantErr1:   false,
			wantErr2:   false,
			wantDiffs1: true,
			wantDiffs2: false,
		},
		{
			name:     "Mismatch Part 2",
			artifact: artifactLong,
			mockSetup: func() {
				mockBytecodeStore[address1.Hex()] = expectedBP1
				mockBytecodeStore[address2.Hex()] = blueprintPreamble + strings.Repeat("ff", 10) // Match length of part2Hex
			},
			wantErr1:   false,
			wantErr2:   false,
			wantDiffs1: false,
			wantDiffs2: true,
		},
		{
			name:     "Error Getting Part 1 Code",
			artifact: artifactLong,
			mockSetup: func() {
				mockGetBytecodeError = fmt.Errorf("rpc1 down")
				// Need more specific mock to only fail for address1, assume global for now
				mockBytecodeStore[address2.Hex()] = expectedBP2 // Set this so part 2 fetch succeeds
			},
			wantErr1:     true,
			wantErr1Cont: "getting onchain code for part 1",
			wantErr2:     true, // Error on addr1 implies error on addr2 too with global mock
			wantErr2Cont: "getting onchain code for part 2",
		},
		{
			name:     "Error Getting Part 2 Code",
			artifact: artifactLong,
			mockSetup: func() {
				var getBytecodeErr error = fmt.Errorf("rpc2 down")
				originalGetBytecode := getOnchainBytecodeImpl
				getOnchainBytecodeImpl = func(client *ethclient.Client, addr common.Address) (string, error) {
					if addr == address1 {
						return expectedBP1, nil
					} else if addr == address2 {
						return "", getBytecodeErr // Specific error for address 2
					}
					return "", fmt.Errorf("unexpected address in mock")
				}
				t.Cleanup(func() { getOnchainBytecodeImpl = originalGetBytecode })
			},
			wantErr1:     false, // Fetch for part 1 succeeds
			wantErr2:     true,
			wantErr2Cont: "rpc2 down", // Check for the core error message
		},
		{
			name:     "No creation code in artifact",
			artifact: newTestArtifact(withCreationBytecode("0x")), // Empty
			mockSetup: func() {
				// Bytecode fetch doesn't matter
			},
			wantErr1:     true,
			wantErr1Cont: "no creation bytecode found",
			wantErr2:     true,
			wantErr2Cont: "no creation bytecode found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t) // Base setup, might be overridden by tt.mockSetup
			tt.mockSetup()

			// Pass nil client because getOnchainBytecodeImpl is mocked
			res1, res2 := verifySplitBlueprintLogic(nil, tt.artifact, targetContractName, targetArtifactPath, address1, address2, fieldName1, fieldName2)

			// Check Result 1
			if tt.wantErr1 {
				require.Error(t, res1.ProcessError)
				if tt.wantErr1Cont != "" {
					assert.Contains(t, res1.ProcessError.Error(), tt.wantErr1Cont)
				}
			} else {
				require.NoError(t, res1.ProcessError)
				assert.Equal(t, SplitBlueprintPart1, res1.Type)
				assert.Equal(t, address1.Hex(), res1.Address)
				assert.Equal(t, targetArtifactPath, res1.ArtifactPath)
				assert.Equal(t, fieldName1, res1.FieldName)
				assert.Equal(t, targetContractName, res1.TargetContract)
				if tt.wantDiffs1 {
					assert.NotEmpty(t, res1.Differences)
				} else {
					assert.Empty(t, res1.Differences)
				}
			}

			// Check Result 2
			if tt.wantErr2 {
				require.Error(t, res2.ProcessError)
				if tt.wantErr2Cont != "" {
					assert.Contains(t, res2.ProcessError.Error(), tt.wantErr2Cont)
				}
			} else {
				require.NoError(t, res2.ProcessError)
				assert.Equal(t, SplitBlueprintPart2, res2.Type)
				assert.Equal(t, address2.Hex(), res2.Address)
				assert.Equal(t, targetArtifactPath, res2.ArtifactPath)
				assert.Equal(t, fieldName2, res2.FieldName)
				assert.Equal(t, targetContractName, res2.TargetContract)
				if tt.wantDiffs2 {
					assert.NotEmpty(t, res2.Differences)
				} else {
					assert.Empty(t, res2.Differences)
				}
			}
		})
	}
}
