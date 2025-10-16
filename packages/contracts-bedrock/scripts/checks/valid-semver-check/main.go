package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
)

func main() {
	if _, err := common.ProcessFilesGlob(
		[]string{"forge-artifacts/**/*.json"},
		[]string{"forge-artifacts/L2StandardBridgeInterop.sol/**.json", "forge-artifacts/OptimismPortalInterop.sol/**.json", "forge-artifacts/RISCV.sol/**.json", "forge-artifacts/EAS.sol/**.json", "forge-artifacts/SchemaRegistry.sol/**.json"},
		processFile,
	); err != nil {
		fmt.Printf("Error: %v/n", err)
		os.Exit(1)
	}
}

func processFile(path string) (*common.Void, []error) {
	artifact, err := common.ReadForgeArtifact(path)
	if err != nil {
		return nil, []error{err}
	}

	// Only check src/ contracts.
	if !strings.HasPrefix(artifact.Ast.AbsolutePath, "src/") {
		return nil, nil
	}

	version, err := getVersion(artifact)
	if err != nil {
		return nil, nil
	}

	err = assertValidSemver(version)
	if err != nil {
		return nil, []error{err}
	}

	fmt.Println("✅ ", artifact.Ast.AbsolutePath)

	return nil, nil
}

func getVersion(artifact *solc.ForgeArtifact) (string, error) {
	for _, node := range artifact.Ast.Nodes {
		if node.NodeType == "ContractDefinition" {
			// Check if there is a version constant definition.
			for _, subNode := range node.Nodes {
				if subNode.NodeType == "VariableDeclaration" &&
					subNode.Mutability == "constant" &&
					subNode.Name == "version" &&
					subNode.Visibility == "public" {
					if subNode.Value.(map[string]interface{})["value"] == nil {
						fmt.Println("WARNING: version constant value is nil", node.Name)
						return "", nil
					}
					return subNode.Value.(map[string]interface{})["value"].(string), nil
				}
			}

			// Check if there is a version function definition.
			for _, subNode := range node.Nodes {
				if subNode.NodeType == "FunctionDefinition" &&
					subNode.Name == "version" &&
					subNode.Visibility == "public" {
					if subNode.Body.Statements == nil {
						return "", fmt.Errorf("version function has no body")
					}
					if len(subNode.Body.Statements) != 1 || subNode.Body.Statements[0].NodeType != "Return" {
						return "", fmt.Errorf("expected version function to have a single statement that returns the version string")
					}
					if subNode.Body.Statements[0].Expression.Value == nil {
						fmt.Println("WARNING: version function value is nil", node.Name)
						return "", nil
					}

					return subNode.Body.Statements[0].Expression.Value.(string), nil
				}
			}
		}
	}

	return "", fmt.Errorf("version function or constant definition not found")
}

func assertValidSemver(version string) error {
	parts := strings.Split(version, ".")

	if len(parts) != 3 {
		return fmt.Errorf("version should be 3 parts")
	}

	_, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("major version should be a number")
	}

	_, err = strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("minor version should be a number")
	}

	_, err = strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("patch version should be a number")
	}

	return nil
}
