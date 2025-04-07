package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
)

var opcmArtifactPath = "forge-artifacts/OPContractsManager.sol/OPContractsManagerUpgrader.json"
var opcmAst *solc.ForgeArtifact
var opcmUpgradeFunctionSelector = "ff2dd5a1"

func main() {
	var err error

	// Get OPCM's AST.
	opcmAst, err = common.ReadForgeArtifact(opcmArtifactPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Process.
	if _, err := common.ProcessFilesGlob(
		[]string{"forge-artifacts/**/*.json"},
		[]string{"forge-artifacts/OPContractsManager.sol/*.json"},
		processFile,
	); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func processFile(artifactPath string) (*common.Void, []error) {
	// Get the artifact.
	artifact, err := common.ReadForgeArtifact(artifactPath)
	if err != nil {
		return nil, []error{err}
	}

	// If the absolute path is not src/L1, return early.
	if !strings.HasPrefix(artifact.Ast.AbsolutePath, "src/L1") {
		return nil, nil
	}

	// Find if it contains any upgrade function and if there are no upgradeFunctions, return early.
	if getNumberOfUpgradeFunctions(artifact) == 0 {
		return nil, nil
	}

	// Get the AST of OPCM's upgrade function.
	opcmUpgradeAst := getOpcmUpgradeFunctionAst(opcmAst)

	// Check that there is a call to contract.upgrade.
	contractName := strings.Split(filepath.Base(artifactPath), ".")[0]
	typeName := "contract I" + contractName
	if !upgradesContract(opcmUpgradeAst.Body.Statements, typeName) {
		return nil, []error{fmt.Errorf("OPCM upgrade function does not call %v.upgrade", contractName)}
	}

	return nil, nil
}

// We want to ensure that:
// - Top level external upgrade calls can be identified
// - External upgrade calls within in a block i.e `{ }` can be identified
// - External upgrade calls within a for, while, do loop can be identified
// - External upgrade calls within the true/false block of if/else-if/else statements can be identified
// - External upgrade calls within a try or catch path
// - External upgrade calls within the true/false block of ternary statements can be identified
// - Any combination of the aforementioned can be identified
func upgradesContract(opcmUpgradeAst []solc.AstNode, typeName string) bool {
	// Loop through all statements finding any external call to an upgrade function with a contract type of `typeName`
	for _, node := range opcmUpgradeAst {
		// To support nested statements or blocks.
		if node.Statements != nil {
			found := upgradesContract(*node.Statements, typeName)
			if found {
				return found
			}
		}

		// For if / else-if / else statements
		if node.TrueBody != nil {
			found := upgradesContract([]solc.AstNode{*node.TrueBody}, typeName)
			if found {
				return found
			}
		}
		if node.FalseBody != nil {
			found := upgradesContract([]solc.AstNode{*node.FalseBody}, typeName)
			if found {
				return found
			}
		}

		// For tenary statement
		if node.Expression != nil && node.Expression.NodeType == "Conditional" {
			if node.Expression.TrueExpression != nil {
				found := upgradesContract([]solc.AstNode{*node.Expression.TrueExpression}, typeName)
				if found {
					return found
				}
			}
			if node.Expression.FalseExpression != nil {
				found := upgradesContract([]solc.AstNode{*node.Expression.FalseExpression}, typeName)
				if found {
					return found
				}
			}
		}

		// For nested tenary statement
		if node.TrueExpression != nil {
			found := upgradesContract([]solc.AstNode{*node.TrueExpression}, typeName)
			if found {
				return found
			}
		}
		if node.FalseExpression != nil {
			found := upgradesContract([]solc.AstNode{*node.FalseExpression}, typeName)
			if found {
				return found
			}
		}

		// To support loops.
		if node.Body != nil && node.Body.Statements != nil {
			found := upgradesContract(node.Body.Statements, typeName)
			if found {
				return found
			}
		}

		// To support try/catch blocks.
		// Try part
		if node.NodeType == "TryStatement" && node.ExternalCall != nil {
			found := upgradesContract([]solc.AstNode{*node.ExternalCall}, typeName)
			if found {
				return found
			}
		}
		// Catch part
		if node.Clauses != nil {
			for _, clause := range node.Clauses {
				if clause.Block != nil && clause.Block.Statements != nil {
					found := upgradesContract(clause.Block.Statements, typeName)
					if found {
						return found
					}
				}
			}
		}

		// If not nested, check if the statement is an external call to an upgrade function with a contract type of `typeName`
		if node.NodeType == "ExpressionStatement" {
			if node.Expression.Expression != nil && node.Expression.Expression.Expression != nil {
				if node.Expression.Expression.MemberName == "upgrade" && node.Expression.Expression.Expression.TypeDescriptions.TypeString == typeName {
					return true
				}
			}
		}

		// To support try external calls and external calls within tenary statements.
		if node.NodeType == "FunctionCall" {
			// Try branch.
			if node.Expression != nil && node.Expression.Expression != nil {
				if node.Expression.MemberName == "upgrade" && node.Expression.Expression.TypeDescriptions.TypeString == typeName {
					return true
				}
			}
		}
	}

	// Else return false.
	return false
}

// Get the AST of OPCM's upgrade function.
func getOpcmUpgradeFunctionAst(opcmArtifact *solc.ForgeArtifact) *solc.AstNode {
	opcmUpgradeAst := solc.AstNode{}
	for _, astNode := range opcmArtifact.Ast.Nodes {
		if astNode.NodeType == "ContractDefinition" && astNode.Name == "OPContractsManagerUpgrader" {
			for _, node := range astNode.Nodes {
				if node.NodeType == "FunctionDefinition" &&
					node.Name == "upgrade" &&
					node.Visibility == "external" &&
					node.FunctionSelector == opcmUpgradeFunctionSelector {
					opcmUpgradeAst = node
					break
				}
			}
		}
	}

	return &opcmUpgradeAst
}

// Get the first upgrade function from the input artifact.
func getNumberOfUpgradeFunctions(artifact *solc.ForgeArtifact) int {
	upgradeFunctions := []solc.AstNode{}
	for _, astNode := range artifact.Ast.Nodes {
		if astNode.NodeType == "ContractDefinition" {
			for _, node := range astNode.Nodes {
				if node.NodeType == "FunctionDefinition" &&
					node.Name == "upgrade" &&
					(node.Visibility == "external" || node.Visibility == "public") {
					upgradeFunctions = append(upgradeFunctions, node)
				}
			}
		}
	}

	return len(upgradeFunctions)
}
