package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const semverLockFile = "snapshots/semver-lock.json"

type SemverLockOutput struct {
	InitCodeHash   string `json:"initCodeHash"`
	SourceCodeHash string `json:"sourceCodeHash"`
}

type SemverLockResult struct {
	ContractKey      string
	SemverLockOutput SemverLockOutput
}

func main() {
	results, err := common.ProcessFilesGlob(
		[]string{"forge-artifacts/**/*.json"},
		[]string{},
		processFile,
	)
	if err != nil {
		fmt.Printf("Failed to generate semver lock: %v\n", err)
		os.Exit(1)
	}

	// Create the output map
	output := make(map[string]SemverLockOutput)
	for _, result := range results {
		if result == nil {
			continue
		}
		output[result.ContractKey] = result.SemverLockOutput
	}

	// Get and sort the keys
	keys := make([]string, 0, len(output))
	for k := range output {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create a sorted map for output
	sortedOutput := make(map[string]SemverLockOutput)
	for _, k := range keys {
		sortedOutput[k] = output[k]
	}

	// Write to JSON file
	jsonData, err := json.MarshalIndent(sortedOutput, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(semverLockFile, jsonData, 0644); err != nil {
		panic(err)
	}

	fmt.Printf("Wrote semver lock file to \"%s\".\n", semverLockFile)
}

func processFile(file string) (*SemverLockResult, []error) {
	artifact, err := common.ReadForgeArtifact(file)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read artifact: %w", err)}
	}

	var sourceFilePath, contractName, contractKey string
	for path, name := range artifact.Metadata.Settings.CompilationTarget {
		sourceFilePath = path
		contractName = name
		contractKey = sourceFilePath + ":" + name
		break
	}

	// Only apply to files in the src directory.
	if !strings.HasPrefix(sourceFilePath, "src/") {
		return nil, nil
	}

	// Check if the contract has a version function or variable with @custom:semver tag
	hasSemverTag := false
	for _, node := range artifact.Ast.Nodes {
		if node.NodeType != "ContractDefinition" || node.Name != contractName {
			continue
		}
		// Check each node inside the contract
		for _, subNode := range node.Nodes {
			// Skip nodes that aren't version functions or variables
			if (subNode.NodeType != "FunctionDefinition" &&
				subNode.NodeType != "VariableDeclaration") ||
				subNode.Name != "version" {
				continue
			}
			if subNode.Documentation == nil {
				continue
			}
			// Handle documentation based on its actual type
			var docText string
			switch doc := subNode.Documentation.(type) {
			case string:
				docText = doc
			case map[string]interface{}:
				if text, ok := doc["text"].(string); ok {
					docText = text
				}
			case solc.AstDocumentation:
				docText = doc.Text
			case *solc.AstDocumentation:
				docText = doc.Text
			}
			if strings.Contains(docText, "@custom:semver") {
				hasSemverTag = true
				break
			}
		}
		if hasSemverTag {
			break
		}
	}
	if !hasSemverTag {
		return nil, nil
	}

	// Extract the init code from the artifact.
	initCodeBytes, err := hex.DecodeString(strings.TrimPrefix(artifact.Bytecode.Object, "0x"))
	if err != nil {
		return nil, []error{fmt.Errorf("failed to decode hex: %w", err)}
	}

	// Extract the source contents from the AST.
	sourceCode, err := os.ReadFile(sourceFilePath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read source file: %w", err)}
	}

	trimmedSourceCode := []byte(strings.TrimSuffix(string(sourceCode), "\n"))
	initCodeHash := fmt.Sprintf("0x%x", crypto.Keccak256Hash(initCodeBytes))
	sourceCodeHash := fmt.Sprintf("0x%x", crypto.Keccak256Hash(trimmedSourceCode))

	return &SemverLockResult{
		ContractKey: contractKey,
		SemverLockOutput: SemverLockOutput{
			InitCodeHash:   initCodeHash,
			SourceCodeHash: sourceCodeHash,
		},
	}, nil
}
