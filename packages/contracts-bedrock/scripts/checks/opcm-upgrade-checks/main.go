package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
)

var excludedFiles = []string{
	"OPPrestateUpdater.sol",
	"OPContractsManager.sol",
}
var opcmArtifactPath = "forge-artifacts/OPContractsManager.sol/OPContractsManager.json"
var opcmUpgradeFunctionSelector = "ff2dd5a1"

func main() {
	// Get all L1 contract file names.
	l1Dir := "src/L1/"
	fileNames, err := os.ReadDir(l1Dir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create a new array of string paths and only add the file to
	// the array if it's not in the global excludedFiles array.
	includedFiles := []string{}
	for _, fileName := range fileNames {
		if slices.Contains(excludedFiles, fileName.Name()) {
			continue
		}
		artifactPath := "forge-artifacts/" + fileName.Name() + "/*.json"
		includedFiles = append(includedFiles, artifactPath)
	}

	// Process.
	if _, err := common.ProcessFilesGlob(
		includedFiles,
		[]string{},
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

	// Find if it contains any upgrade function.
	upgradeFunctions := []solc.AstNode{}
	for _, astNode := range artifact.Ast.Nodes {
		if astNode.NodeType == "ContractDefinition" {
			for _, node := range astNode.Nodes {
				if hasUpgradeFunction(node) {
					upgradeFunctions = append(upgradeFunctions, node)
				}
			}
		}
	}

	// If there are no upgradeFunctions, return early.
	if len(upgradeFunctions) == 0 {
		return nil, nil
	}

	// Get OPCM's AST.
	opcmAst, err := common.ReadForgeArtifact(opcmArtifactPath)
	if err != nil {
		return nil, []error{err}
	}

	// Get the AST of OPCM's upgrade function.
	opcmUpgradeAst := solc.AstNode{}
	for _, astNode := range opcmAst.Ast.Nodes {
		if astNode.NodeType == "ContractDefinition" {
			for _, node := range astNode.Nodes {
				if isOpcmUpgradeFunction(node) {
					opcmUpgradeAst = node
					break
				}
			}
		}
	}

	// Check that there is a call to contract.upgrade.
	contractName := strings.Split(filepath.Base(filepath.Dir(artifactPath)), ".")[0]
	typeName := "contract " + contractName
	if !upgradesContract(opcmUpgradeAst.Body.Statements, typeName) {
		return nil, []error{fmt.Errorf("OPCM upgrade function does not call %v.upgrade", contractName)}
	}

	return nil, nil
}

func upgradesContract(nodes []solc.AstNode, typeName string) bool {
	for _, node := range nodes {
		if node.NodeType == "ExpressionStatement" {
			if node.Expression.Expression != nil {
				if node.Expression.Expression.MemberName == "upgrade" && node.Expression.Expression.Expression.TypeDescriptions.TypeString == typeName {
					return true
				}
			}
		}
	}

	return false
}

func isOpcmUpgradeFunction(node solc.AstNode) bool {
	return node.NodeType == "FunctionDefinition" &&
		node.Name == "upgrade" &&
		node.Visibility == "external" &&
		node.FunctionSelector == opcmUpgradeFunctionSelector
}

func hasUpgradeFunction(node solc.AstNode) bool {
	return node.NodeType == "FunctionDefinition" &&
		node.Name == "upgrade" &&
		(node.Visibility == "external" || node.Visibility == "public")
}
