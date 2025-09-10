package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
)

var OPCM_ARTIFACT_PATH = "forge-artifacts/OPContractsManager.sol/OPContractsManagerUpgrader.json"

type InternalUpgradeFunctionType struct {
	name     string
	typeName string
}

type CallType int

const (
	NOT_FOUND CallType = iota
	UPGRADE_EXTERNAL_CALL
	UPGRADE_INTERNAL_CALL
)

func main() {
	// Assert that the OPCM_BASE's upgradeToAndCall function has a call from IProxyAdmin.upgradeAndCall.
	res, err := assertOPCMBaseInternalUpgradeFunctionCallUpgrade("contract IProxyAdmin", "OPContractsManagerBase")
	if !res {
		fmt.Printf("error: %v\n", err)
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

func assertOPCMBaseInternalUpgradeFunctionCallUpgrade(upgraderContractTypeName string, upgradeContractsName string) (bool, error) {
	// First, get the OPCM's artifact.
	opcmArtifact, err := common.ReadForgeArtifact(OPCM_ARTIFACT_PATH)
	if err != nil {
		return false, fmt.Errorf("error: %w", err)
	}

	// Then get the OPCM Base's upgradeToAndCall internal functions AST.
	opcmBaseUpgradeToAndCallAst := solc.AstNode{}
	for _, node := range opcmArtifact.Ast.Nodes {
		if node.NodeType == "ContractDefinition" && node.Name == upgradeContractsName {
			for _, node := range node.Nodes {
				if node.NodeType == "FunctionDefinition" && node.Name == "upgradeToAndCall" && node.Visibility == "internal" && len(node.Parameters.Parameters) == 4 {
					opcmBaseUpgradeToAndCallAst = node
					break
				}
			}
		}
	}
	if opcmBaseUpgradeToAndCallAst.NodeType == "" {
		return false, fmt.Errorf("%v's upgradeToAndCall internal function not found", upgradeContractsName)
	}

	// Next, ensure that a call to IProxyAdmin.upgradeAndCall is found in the OPCM's upgradeToAndCall function.
	found := upgradesContract(opcmBaseUpgradeToAndCallAst.Body.Statements, "upgradeAndCall", upgraderContractTypeName, InternalUpgradeFunctionType{})
	if found == UPGRADE_EXTERNAL_CALL {
		return true, nil
	}

	return false, fmt.Errorf("%v's upgradeToAndCall internal function does not have a call from IProxyAdmin.upgradeAndCall", upgradeContractsName)
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

	// Find if it contains any upgrade function
	numOfUpgradeFunctions := getNumberOfUpgradeFunctions(artifact)

	// If there are no upgrade functions, return early.
	if numOfUpgradeFunctions == 0 {
		return nil, nil
	}

	// If there are more than 1 upgrade functions, return an error.
	if numOfUpgradeFunctions > 1 {
		return nil, []error{fmt.Errorf("expected 0 or 1 upgrade function, found %v", numOfUpgradeFunctions)}
	}

	// Get OPCM's AST.
	opcmAst, err := common.ReadForgeArtifact(OPCM_ARTIFACT_PATH)
	if err != nil {
		return nil, []error{err}
	}

	// Check that there is a call to contract.upgrade.
	contractName := strings.Split(filepath.Base(artifactPath), ".")[0]
	typeName := "contract I" + contractName

	var callType CallType
	if contractName == "SuperchainConfig" {
		// Get the AST of OPCM's upgradeSuperchainConfig function.
		opcmUpgradeSuperchainConfigAst, err := getOpcmUpgradeFunctionAst(opcmAst, "upgradeSuperchainConfig")
		if err != nil {
			return nil, []error{err}
		}

		callType = upgradesContract(opcmUpgradeSuperchainConfigAst.Body.Statements, "upgrade", typeName, InternalUpgradeFunctionType{
			name:     "upgradeToAndCall",
			typeName: "function (contract IProxyAdmin,address,address,bytes memory)",
		})
	} else {
		// Get the AST of OPCM's upgrade function.
		opcmUpgradeAst, err := getOpcmUpgradeFunctionAst(opcmAst, "_doChainUpgrade")
		if err != nil {
			return nil, []error{err}
		}

		callType = upgradesContract(opcmUpgradeAst.Body.Statements, "upgrade", typeName, InternalUpgradeFunctionType{
			name:     "upgradeToAndCall",
			typeName: "function (contract IProxyAdmin,address,address,bytes memory)",
		})
	}

	if callType == NOT_FOUND {
		return nil, []error{fmt.Errorf("OPCM upgrade function does not call %v.upgrade", contractName)}
	}

	return nil, nil
}

// We want to ensure that:
//   - Top level external upgrade calls call e.g `IContract.upgrade(...)` and
//     Internal ones called via `upgradeToAndCall(param: IProxyAdmin, address(param: IContract), param: address, abi.encodeCall(IContract.upgrade, (...)))` can be identified
//   - External upgrade calls within in a block i.e `{ }` can be identified
//   - External upgrade calls within a for, while, do loop can be identified
//   - External upgrade calls within the true/false block of if/else-if/else statements can be identified
//   - External upgrade calls within a try or catch path
//   - External upgrade calls within the true/false block of ternary statements can be identified
//   - Any combination of the aforementioned can be identified
func upgradesContract(opcmUpgradeAst []solc.AstNode, expectedExternalCallName string, typeName string, internalFunctionTypes InternalUpgradeFunctionType) CallType {
	// Loop through all statements finding any external call to an upgrade function with a contract type of `typeName`
	for _, node := range opcmUpgradeAst {
		// To support nested statements or blocks.
		if node.Statements != nil {
			found := upgradesContract(*node.Statements, expectedExternalCallName, typeName, internalFunctionTypes)
			if found != NOT_FOUND {
				return found
			}
		}

		// For if / else-if / else statements
		if node.TrueBody != nil {
			found := upgradesContract([]solc.AstNode{*node.TrueBody}, expectedExternalCallName, typeName, internalFunctionTypes)
			if found != NOT_FOUND {
				return found
			}
		}
		if node.FalseBody != nil {
			found := upgradesContract([]solc.AstNode{*node.FalseBody}, expectedExternalCallName, typeName, internalFunctionTypes)
			if found != NOT_FOUND {
				return found
			}
		}

		// For tenary statement
		if node.Expression != nil && node.Expression.NodeType == "Conditional" {
			if node.Expression.TrueExpression != nil {
				found := upgradesContract([]solc.AstNode{*node.Expression.TrueExpression}, expectedExternalCallName, typeName, internalFunctionTypes)
				if found != NOT_FOUND {
					return found
				}
			}
			if node.Expression.FalseExpression != nil {
				found := upgradesContract([]solc.AstNode{*node.Expression.FalseExpression}, expectedExternalCallName, typeName, internalFunctionTypes)
				if found != NOT_FOUND {
					return found
				}
			}
		}

		// For nested tenary statement
		if node.TrueExpression != nil {
			found := upgradesContract([]solc.AstNode{*node.TrueExpression}, expectedExternalCallName, typeName, internalFunctionTypes)
			if found != NOT_FOUND {
				return found
			}
		}
		if node.FalseExpression != nil {
			found := upgradesContract([]solc.AstNode{*node.FalseExpression}, expectedExternalCallName, typeName, internalFunctionTypes)
			if found != NOT_FOUND {
				return found
			}
		}

		// To support loops.
		if node.Body != nil && node.Body.Statements != nil {
			found := upgradesContract(node.Body.Statements, expectedExternalCallName, typeName, internalFunctionTypes)
			if found != NOT_FOUND {
				return found
			}
		}

		// To support try/catch blocks.
		// Try part
		if node.NodeType == "TryStatement" && node.ExternalCall != nil {
			found := upgradesContract([]solc.AstNode{*node.ExternalCall}, expectedExternalCallName, typeName, internalFunctionTypes)
			if found != NOT_FOUND {
				return found
			}
		}
		// Catch part
		if node.Clauses != nil {
			for _, clause := range node.Clauses {
				if clause.Block != nil && clause.Block.Statements != nil {
					found := upgradesContract(clause.Block.Statements, expectedExternalCallName, typeName, internalFunctionTypes)
					if found != NOT_FOUND {
						return found
					}
				}
			}
		}

		// If not nested, check if the statement is an external call to an upgrade function with a contract type of `typeName`
		if node.NodeType == "ExpressionStatement" {
			if identifyValidExternalUpgradeCall(node.Expression.Expression, expectedExternalCallName, typeName) {
				return UPGRADE_EXTERNAL_CALL
			}
		}

		// To support try external calls and external calls within tenary statements.
		if node.NodeType == "FunctionCall" {
			if identifyValidExternalUpgradeCall(node.Expression, expectedExternalCallName, typeName) {
				return UPGRADE_EXTERNAL_CALL
			}
		}

		// To support internal upgrade functions.
		if node.NodeType == "ExpressionStatement" {
			if identifyValidInternalUpgradeCall(node.Expression, internalFunctionTypes, typeName) {
				return UPGRADE_INTERNAL_CALL
			}
		}

		// To support internal upgrade function calls within tenary statements.
		if node.NodeType == "FunctionCall" {
			// cast node into an expression-like type with relevant fields that we need
			expression := solc.Expression{
				Expression: node.Expression,
				Arguments:  node.Arguments,
			}
			if identifyValidInternalUpgradeCall(&expression, internalFunctionTypes, typeName) {
				return UPGRADE_INTERNAL_CALL
			}
		}
	}

	// Else return false.
	return NOT_FOUND
}

func identifyValidExternalUpgradeCall(expression *solc.Expression, expectedExternalCallName string, typeName string) bool {
	// To support external upgrade calls.
	if expression != nil && expression.Expression != nil {
		if expression.MemberName == expectedExternalCallName && expression.Expression.TypeDescriptions.TypeString == typeName {
			return true
		}
	}

	return false
}

func identifyValidInternalUpgradeCall(expression *solc.Expression, internalFunctionTypes InternalUpgradeFunctionType, typeName string) bool {
	// To support internal upgrade functions.
	if expression != nil && expression.Expression != nil {
		if expression.Expression.Name == internalFunctionTypes.name && expression.Expression.TypeDescriptions.TypeString == internalFunctionTypes.typeName {
			// Assert that the second argument is of type `typeName` that was cast (within the argument list) into an address
			if expression.Arguments[1].Arguments[0].TypeDescriptions.TypeString == typeName {
				// Assert that the fourth argument is of type `abi.encodeCall(IContract.upgrade, (...))`
				expectedTypeString := "type(" + typeName + ")"
				if expression.Arguments[3].Arguments[0].Expression.TypeDescriptions.TypeString == expectedTypeString &&
					expression.Arguments[3].Arguments[0].MemberName == "upgrade" &&
					expression.Arguments[3].Expression.Expression.Name == "abi" &&
					expression.Arguments[3].Expression.MemberName == "encodeCall" {
					// If all of the above passes, return true
					return true
				}
			}
		}
	}

	return false
}

// Get the AST of OPCM's upgrade function.
// Returns an error if zero or more than one external upgrade function is found.
func getOpcmUpgradeFunctionAst(opcmArtifact *solc.ForgeArtifact, upgradeFunctionName string) (*solc.AstNode, error) {
	opcmUpgradeFunctions := []solc.AstNode{}
	for _, astNode := range opcmArtifact.Ast.Nodes {
		if astNode.NodeType == "ContractDefinition" && astNode.Name == "OPContractsManagerUpgrader" {
			for _, node := range astNode.Nodes {
				if node.NodeType == "FunctionDefinition" &&
					node.Name == upgradeFunctionName {
					opcmUpgradeFunctions = append(opcmUpgradeFunctions, node)
				}
			}
		}
	}

	if len(opcmUpgradeFunctions) == 0 {
		return nil, fmt.Errorf("no external %s function found in OPContractsManagerUpgrader", upgradeFunctionName)
	}

	if len(opcmUpgradeFunctions) > 1 {
		return nil, fmt.Errorf("multiple external %s functions found in OPContractsManagerUpgrader, expected 1", upgradeFunctionName)
	}

	return &opcmUpgradeFunctions[0], nil
}

// Get the number of upgrade functions from the input artifact.
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
