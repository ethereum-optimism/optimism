package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

type ForgeArtifact struct {
	// ABI is a nested JSON data structure, including some objects/maps.
	// We declare it as interface, and not raw-message, such that Go decodes into map[string]interface{}
	// where possible. The JSON-encoder will then sort the keys (default Go JSON behavior on maps),
	// to reproduce the sortKeys(abi) result of the legacy Typescript version of the snapshort-generator.
	ABI interface{} `json:"abi"`
	Ast *struct {
		NodeType string `json:"nodeType"`
		Nodes    []struct {
			NodeType     string `json:"nodeType"`
			Name         string `json:"name"`
			ContractKind string `json:"contractKind"`
			Abstract     bool   `json:"abstract"`
		} `json:"nodes"`
	} `json:"ast"`
	StorageLayout struct {
		Storage []struct {
			Type   string          `json:"type"`
			Label  json.RawMessage `json:"label"`
			Offset json.RawMessage `json:"offset"`
			Slot   json.RawMessage `json:"slot"`
		} `json:"storage"`
		Types map[string]struct {
			Label         string          `json:"label"`
			NumberOfBytes json.RawMessage `json:"numberOfBytes"`
		} `json:"types"`
	} `json:"storageLayout"`
	Bytecode struct {
		Object string `json:"object"`
	} `json:"bytecode"`
}

type AbiSpecStorageLayoutEntry struct {
	Bytes  json.RawMessage `json:"bytes"`
	Label  json.RawMessage `json:"label"`
	Offset json.RawMessage `json:"offset"`
	Slot   json.RawMessage `json:"slot"`
	Type   string          `json:"type"`
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Println("Expected path of contracts-bedrock as CLI argument")
		os.Exit(1)
	}
	rootDir := flag.Arg(0)
	err := generateSnapshots(rootDir)
	if err != nil {
		fmt.Printf("Failed to generate snapshots: %v\n", err)
		os.Exit(1)
	}
}

func generateSnapshots(rootDir string) error {

	forgeArtifactsDir := filepath.Join(rootDir, "forge-artifacts")
	srcDir := filepath.Join(rootDir, "src")
	outDir := filepath.Join(rootDir, "snapshots")

	storageLayoutDir := filepath.Join(outDir, "storageLayout")
	abiDir := filepath.Join(outDir, "abi")

	fmt.Printf("writing abi and storage layout snapshots to %s\n", outDir)

	// Clean and recreate directories
	if err := os.RemoveAll(storageLayoutDir); err != nil {
		return fmt.Errorf("failed to remove storage layout dir: %w", err)
	}
	if err := os.RemoveAll(abiDir); err != nil {
		return fmt.Errorf("failed to remove ABI dir: %w", err)
	}
	if err := os.MkdirAll(storageLayoutDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create storage layout dir: %w", err)
	}
	if err := os.MkdirAll(abiDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create ABI dir: %w", err)
	}

	contractSources, err := getAllContractsSources(srcDir)
	if err != nil {
		return fmt.Errorf("failed to retrieve contract sources: %w", err)
	}

	knownAbis := make(map[string]interface{})

	for _, contractFile := range contractSources {
		contractArtifacts := filepath.Join(forgeArtifactsDir, contractFile)
		files, err := os.ReadDir(contractArtifacts)
		if err != nil {
			return fmt.Errorf("failed to scan contract artifacts of %q: %w", contractFile, err)
		}

		for _, file := range files {
			artifactPath := filepath.Join(contractArtifacts, file.Name())
			data, err := os.ReadFile(artifactPath)
			if err != nil {
				return fmt.Errorf("failed to read artifact %q: %w", artifactPath, err)
			}
			var artifact ForgeArtifact
			if err := json.Unmarshal(data, &artifact); err != nil {
				return fmt.Errorf("failed to decode artifact %q: %w", artifactPath, err)
			}

			contractName, err := parseArtifactName(file.Name())
			if err != nil {
				return fmt.Errorf("failed to parse artifact name %q: %w", file.Name(), err)
			}

			// HACK: This is a hack to ignore libraries and abstract contracts. Not robust against changes to solc's internal ast repr
			if artifact.Ast == nil {
				return fmt.Errorf("ast isn't present in forge-artifacts. Did you run forge build with `--ast`? Artifact: %s", artifactPath)
			}
			// Check if the artifact is a contract
			isContract := false
			for _, node := range artifact.Ast.Nodes {
				if node.NodeType == "ContractDefinition" &&
					node.Name == contractName &&
					node.ContractKind == "contract" &&
					!node.Abstract {
					isContract = true
					break
				}
			}
			if !isContract {
				fmt.Printf("ignoring library/interface %s\n", contractName)
				continue
			}

			storageLayout := make([]AbiSpecStorageLayoutEntry, 0, len(artifact.StorageLayout.Storage))
			for _, storageEntry := range artifact.StorageLayout.Storage {
				// convert ast-based type to solidity type
				typ, ok := artifact.StorageLayout.Types[storageEntry.Type]
				if !ok {
					return fmt.Errorf("undefined type for %s:%s", contractName, storageEntry.Label)
				}
				storageLayout = append(storageLayout, AbiSpecStorageLayoutEntry{
					Label:  storageEntry.Label,
					Bytes:  typ.NumberOfBytes,
					Offset: storageEntry.Offset,
					Slot:   storageEntry.Slot,
					Type:   typ.Label,
				})
			}

			if existingAbi, exists := knownAbis[contractName]; exists {
				if !jsonEqual(existingAbi, artifact.ABI) {
					return fmt.Errorf("detected multiple artifact versions with different ABIs for %s", contractFile)
				} else {
					fmt.Printf("detected multiple artifacts for %s\n", contractName)
				}
			} else {
				knownAbis[contractName] = artifact.ABI
			}

			// Sort and write snapshots
			if err := writeJSON(filepath.Join(abiDir, contractName+".json"), artifact.ABI); err != nil {
				return fmt.Errorf("failed to write ABI snapshot JSON of %q: %w", contractName, err)
			}

			if err := writeJSON(filepath.Join(storageLayoutDir, contractName+".json"), storageLayout); err != nil {
				return fmt.Errorf("failed to write storage layout snapshot JSON of %q: %w", contractName, err)
			}
		}
	}
	return nil
}

func getAllContractsSources(srcDir string) ([]string, error) {
	var paths []string
	if err := readFilesRecursively(srcDir, &paths); err != nil {
		return nil, fmt.Errorf("failed to retrieve files: %w", err)
	}

	var solFiles []string
	for _, p := range paths {
		if filepath.Ext(p) == ".sol" {
			solFiles = append(solFiles, filepath.Base(p))
		}
	}
	sort.Strings(solFiles)
	return solFiles, nil
}

func readFilesRecursively(dir string, paths *[]string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		filePath := filepath.Join(dir, file.Name())
		if file.IsDir() {
			if err := readFilesRecursively(filePath, paths); err != nil {
				return fmt.Errorf("failed to recurse into %q: %w", filePath, err)
			}
		} else {
			*paths = append(*paths, filePath)
		}
	}
	return nil
}

// ContractName.0.9.8.json -> ContractName.sol
// ContractName.json -> ContractName.sol
func parseArtifactName(artifactVersionFile string) (string, error) {
	re := regexp.MustCompile(`(.*?)\.([0-9]+\.[0-9]+\.[0-9]+)?`)
	match := re.FindStringSubmatch(artifactVersionFile)
	if len(match) < 2 {
		return "", fmt.Errorf("invalid artifact file name: %q", artifactVersionFile)
	}
	return match[1], nil
}

func writeJSON(filename string, data interface{}) error {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}
	jsonData := out.Bytes()
	if len(jsonData) > 0 && jsonData[len(jsonData)-1] == '\n' { // strip newline
		jsonData = jsonData[:len(jsonData)-1]
	}
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func jsonEqual(a, b interface{}) bool {
	jsonA, errA := json.Marshal(a)
	jsonB, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(jsonA) == string(jsonB)
}
