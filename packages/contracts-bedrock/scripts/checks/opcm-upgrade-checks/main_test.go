package main

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
	"github.com/stretchr/testify/assert"
)

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
	// To add tests for this, create a contract with one or a combination of solidity statements and an optional upgrade function call within it to a contract type of IUpgradeable.
	// Then create a constant bool variable EXPECTED_OUTPUT and set it to true if the upgrade function call is expected to be found and false otherwise.
	// See opcm_upgrade_checks_mocks.sol for already existing mock contracts used for testing.

	artifact, err := common.ReadForgeArtifact("../../../forge-artifacts/OPCMUpgradeChecksMocks.sol/IUpgradeable.json")
	if err != nil {
		t.Fatalf("Failed to load artifact: %v", err)
	}

	type test struct {
		name           string
		upgradeAst     []solc.AstNode
		typeName       string
		expectedOutput bool
	}

	tests := []test{}

	for _, node := range artifact.Ast.Nodes {
		if node.NodeType == "ContractDefinition" && node.Name != "IUpgradeable" {
			upgradeAst := solc.AstNode{}
			expectedOutput := false
			for _, astNode := range node.Nodes {
				if astNode.NodeType == "FunctionDefinition" && astNode.Name == "upgrade" {
					if upgradeAst.NodeType != "" {
						t.Fatalf("Expected only one upgrade function")
					}
					upgradeAst = astNode
				}

				if astNode.NodeType == "VariableDeclaration" &&
					astNode.Name == "EXPECTED_OUTPUT" &&
					astNode.Mutability == "constant" {

					value, ok := astNode.Value.(map[string]interface{})
					if !ok {
						t.Fatalf("Expected value to be a map: %v", astNode.Value)
					}

					if value["kind"] != "bool" {
						continue
					}

					if value["value"] == "true" {
						expectedOutput = true
					} else if value["value"] == "false" {
						expectedOutput = false
					} else {
						t.Fatalf("Expected output is not a boolean: %s", astNode.Value)
					}
				}
			}

			tests = append(tests, struct {
				name           string
				upgradeAst     []solc.AstNode
				typeName       string
				expectedOutput bool
			}{node.Name, []solc.AstNode{upgradeAst}, "contract IUpgradeable", expectedOutput})
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := upgradesContract(test.upgradeAst, test.typeName)
			assert.Equal(t, test.expectedOutput, output)
		})
	}
}
