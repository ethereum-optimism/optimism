package main

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/stretchr/testify/assert"
)

// mockDirEntry implements os.DirEntry for testing
type mockDirEntry struct {
	name string
}

func (m mockDirEntry) Name() string {
	return m.name
}

func (m mockDirEntry) IsDir() bool {
	return false
}

func (m mockDirEntry) Type() os.FileMode {
	return 0
}

func (m mockDirEntry) Info() (os.FileInfo, error) {
	return nil, nil
}

func TestFilterFilesAndDeriveArtifactPath(t *testing.T) {
	tests := []struct {
		name                  string
		files                 []os.DirEntry
		excludedFiles         []string
		expectedArtifactPaths []string
	}{
		{
			name: "no excluded files",
			files: []os.DirEntry{
				&mockDirEntry{name: "Opcm.sol"},
				&mockDirEntry{name: "Opcm2.sol"},
			},
			excludedFiles:         []string{},
			expectedArtifactPaths: []string{"forge-artifacts/Opcm.sol/*.json", "forge-artifacts/Opcm2.sol/*.json"},
		},
		{
			name: "one excluded file",
			files: []os.DirEntry{
				&mockDirEntry{name: "Opcm.sol"},
				&mockDirEntry{name: "Opcm2.sol"},
			},
			excludedFiles:         []string{"Opcm.sol"},
			expectedArtifactPaths: []string{"forge-artifacts/Opcm2.sol/*.json"},
		},
		{
			name: "multiple excluded files",
			files: []os.DirEntry{
				&mockDirEntry{name: "Opcm.sol"},
				&mockDirEntry{name: "Opcm2.sol"},
			},
			excludedFiles:         []string{"Opcm.sol", "Opcm2.sol"},
			expectedArtifactPaths: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifactPaths := filterFilesAndDeriveArtifactPath(test.files, test.excludedFiles)
			assert.Equal(t, test.expectedArtifactPaths, artifactPaths)
		})
	}
}

func TestGetOpcmUpgradeFunctionAst(t *testing.T) {
	tests := []struct {
		name         string
		opcmArtifact *solc.ForgeArtifact
		expectedAst  *solc.AstNode
	}{
		{
			name: "With upgrade function",
			opcmArtifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{
							NodeType: "ContractDefinition",
							Nodes: []solc.AstNode{
								{
									NodeType:         "FunctionDefinition",
									Name:             "upgrade",
									Visibility:       "external",
									FunctionSelector: opcmUpgradeFunctionSelector,
									Nodes: []solc.AstNode{
										{
											NodeType: "UniqueNonExistentNodeType",
										},
									},
								},
							},
							Name: "OPContractsManagerUpgrader",
						},
					},
				},
			},
			expectedAst: &solc.AstNode{
				NodeType:         "FunctionDefinition",
				Name:             "upgrade",
				Visibility:       "external",
				FunctionSelector: opcmUpgradeFunctionSelector,
				Nodes: []solc.AstNode{
					{
						NodeType: "UniqueNonExistentNodeType",
					},
				},
			},
		},
		{
			name: "With an upgrade function but not the right visibility",
			opcmArtifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{
							NodeType: "ContractDefinition",
							Nodes: []solc.AstNode{
								{
									NodeType:         "FunctionDefinition",
									Name:             "upgrade",
									Visibility:       "public",
									FunctionSelector: opcmUpgradeFunctionSelector,
									Nodes: []solc.AstNode{
										{
											NodeType: "UniqueNonExistentNodeType",
										},
									},
								},
							},
							Name: "OPContractsManagerUpgrader",
						},
					},
				},
			},
			expectedAst: &solc.AstNode{},
		},
		{
			name: "With an upgrade function but not the right function selector",
			opcmArtifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{
							NodeType: "ContractDefinition",
							Nodes: []solc.AstNode{
								{
									NodeType:         "FunctionDefinition",
									Name:             "upgrade",
									Visibility:       "external",
									FunctionSelector: "aabbccdd",
									Nodes: []solc.AstNode{
										{
											NodeType: "UniqueNonExistentNodeType",
										},
									},
								},
							},
							Name: "OPContractsManagerUpgrader",
						},
					},
				},
			},
			expectedAst: &solc.AstNode{},
		},
		{
			name: "With no upgrade function",
			opcmArtifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{
							NodeType: "ContractDefinition",
							Nodes: []solc.AstNode{
								{
									NodeType:         "FunctionDefinition",
									Name:             "randomFunctionName",
									Visibility:       "external",
									FunctionSelector: opcmUpgradeFunctionSelector,
									Nodes: []solc.AstNode{
										{
											NodeType: "UniqueNonExistentNodeType",
										},
									},
								},
							},
							Name: "OPContractsManagerUpgrader",
						},
					},
				},
			},
			expectedAst: &solc.AstNode{},
		},
		{
			name: "With no contract definition",
			opcmArtifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{},
				},
			},
			expectedAst: &solc.AstNode{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ast := getOpcmUpgradeFunctionAst(test.opcmArtifact)
			assert.Equal(t, test.expectedAst, ast)
		})
	}
}

func TestGetNumberOfUpgradeFunctions(t *testing.T) {
	tests := []struct {
		name        string
		artifact    *solc.ForgeArtifact
		expectedNum int
	}{
		{
			name: "With an external upgrade function",
			artifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{
							NodeType: "ContractDefinition",
							Nodes: []solc.AstNode{
								{
									NodeType:   "FunctionDefinition",
									Name:       "upgrade",
									Visibility: "external",
								},
							},
						},
					},
				},
			},
			expectedNum: 1,
		},
		{
			name: "With a public upgrade function",
			artifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{
							NodeType: "ContractDefinition",
							Nodes: []solc.AstNode{
								{
									NodeType:   "FunctionDefinition",
									Name:       "upgrade",
									Visibility: "public",
								},
							},
						},
					},
				},
			},
			expectedNum: 1,
		},
		{
			name: "With multiple upgrade functions",
			artifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{
							NodeType: "ContractDefinition",
							Nodes: []solc.AstNode{
								{
									NodeType:   "FunctionDefinition",
									Name:       "upgrade",
									Visibility: "external",
								},
							},
						},
						{
							NodeType: "ContractDefinition",
							Nodes: []solc.AstNode{
								{
									NodeType:   "FunctionDefinition",
									Name:       "upgrade",
									Visibility: "public",
								},
							},
						},
					},
				},
			},
			expectedNum: 2,
		},
		{
			name: "With no upgrade functions",
			artifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{},
				},
			},
			expectedNum: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			num := getNumberOfUpgradeFunctions(test.artifact)
			assert.Equal(t, test.expectedNum, num)
		})
	}
}

func TestUpgradesContract(t *testing.T) {
	tests := []struct {
		name           string
		node           []solc.AstNode
		typeName       string
		expectedOutput bool
	}{
		{
			name: "With a top level external upgrade function call",
			node: []solc.AstNode{
				{
					NodeType: "ExpressionStatement",
					Expression: &solc.Expression{
						Expression: &solc.Expression{
							MemberName: "upgrade",
							Expression: &solc.Expression{
								TypeDescriptions: &solc.AstTypeDescriptions{
									TypeString: "contract RandomToBeUpgradedContract",
								},
							},
						},
					},
				},
			},
			typeName:       "contract RandomToBeUpgradedContract",
			expectedOutput: true,
		},
		{
			name: "With an external upgrade function call within a block",
			node: []solc.AstNode{
				{
					NodeType: "Block",
					Statements: &[]solc.AstNode{
						{
							NodeType: "ExpressionStatement",
							Expression: &solc.Expression{
								Expression: &solc.Expression{
									MemberName: "upgrade",
									Expression: &solc.Expression{
										TypeDescriptions: &solc.AstTypeDescriptions{
											TypeString: "contract RandomToBeUpgradedContract",
										},
									},
								},
							},
						},
					},
				},
			},
			typeName:       "contract RandomToBeUpgradedContract",
			expectedOutput: true,
		},
		{
			name:           "With NO external upgrade function call",
			node:           []solc.AstNode{},
			typeName:       "contract RandomToBeUpgradedContract",
			expectedOutput: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := upgradesContract(test.node, test.typeName)
			assert.Equal(t, test.expectedOutput, output)
		})
	}
}
