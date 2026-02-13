package verify

import (
	"encoding/json"
	"fmt"
	"path"

	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum/go-ethereum/log"
)

// SourceContent represents the content of a source file
type SourceContent struct {
	Content string
}

// OptimizerSettings represents compiler optimizer configuration
type OptimizerSettings struct {
	Enabled bool `json:"enabled"`
	Runs    int  `json:"runs"`
}

// ArtifactMetadata contains processed artifact information
type ArtifactMetadata struct {
	ContractPath    string
	CompilerVersion string
	Optimizer       OptimizerSettings
	EVMVersion      string
	Sources         map[string]SourceContent
}

// Map state.json struct fields to forge artifact paths
var contractNameExceptions = map[string]string{
	"OptimismPortal":              "OptimismPortal2.sol/OptimismPortal2.json",
	"L1StandardBridgeProxy":       "L1ChugSplashProxy.sol/L1ChugSplashProxy.json",
	"L1CrossDomainMessengerProxy": "ResolvedDelegateProxy.sol/ResolvedDelegateProxy.json",
	"Opcm":                        "OPContractsManager.sol/OPContractsManager.json",
	"OpcmContractsContainer":      "OPContractsManager.sol/OPContractsManagerContractsContainer.json",
	"OpcmGameTypeAdder":           "OPContractsManager.sol/OPContractsManagerGameTypeAdder.json",
	"OpcmDeployer":                "OPContractsManager.sol/OPContractsManagerDeployer.json",
	"OpcmUpgrader":                "OPContractsManager.sol/OPContractsManagerUpgrader.json",
	"OpcmInteropMigrator":         "OPContractsManager.sol/OPContractsManagerInteropMigrator.json",
	"OpcmStandardValidator":       "OPContractsManagerStandardValidator.sol/OPContractsManagerStandardValidator.json",
	"OpcmMigrator":                "OPContractsManagerMigrator.sol/OPContractsManagerMigrator.json",
	"OpcmV2":                      "OPContractsManagerV2.sol/OPContractsManagerV2.json",
	"OpcmContainer":               "OPContractsManagerContainer.sol/OPContractsManagerContainer.json",
	"OpcmUtils":                   "OPContractsManagerUtils.sol/OPContractsManagerUtils.json",
	"Mips":                        "MIPS64.sol/MIPS64.json",
	"EthLockbox":                  "ETHLockbox.sol/ETHLockbox.json",
}

func getArtifactPath(name string) string {
	// Handle state file contract names (underscore-separated)
	if strings.Contains(name, "_") {
		parts := strings.Split(name, "_")

		// Handle proxy contracts (ending with _proxy)
		if parts[len(parts)-1] == "proxy" {
			return path.Join("Proxy.sol", "Proxy.json")
		}

		// Handle implementation contracts (ending with _impl)
		if parts[len(parts)-1] == "impl" {
			// For impl contracts, remove the _impl suffix and the prefix (like "implementations_" or "superchain_")
			contractName := strings.Join(parts[1:len(parts)-1], "_") // Skip first part (prefix) and last part (_impl)
			// Convert snake_case to PascalCase for contract names
			contractName = convertSnakeToPascal(contractName)
			if artifactPath, exists := contractNameExceptions[contractName]; exists {
				return artifactPath
			}
			return path.Join(contractName+".sol", contractName+".json")
		}

		// For other underscore-separated names, convert to PascalCase
		contractName := convertSnakeToPascal(name)
		if artifactPath, exists := contractNameExceptions[contractName]; exists {
			return artifactPath
		}
		return path.Join(contractName+".sol", contractName+".json")
	}

	// Handle regular contract names (legacy logic)
	lookupName := strings.TrimSuffix(name, "Address")
	lookupName = strings.TrimSuffix(lookupName, "Impl")
	lookupName = strings.TrimSuffix(lookupName, "Singleton")
	lookupName = strings.ToUpper(string(lookupName[0])) + lookupName[1:]

	if artifactPath, exists := contractNameExceptions[lookupName]; exists {
		return artifactPath
	}

	lookupName = strings.TrimSuffix(lookupName, "Proxy")

	// If it was a proxy and not a special case, return "Proxy"
	if strings.HasSuffix(name, "ProxyAddress") {
		return path.Join("Proxy.sol", "Proxy.json")
	}

	return path.Join(lookupName+".sol", lookupName+".json")
}

// GetArtifactPath returns the artifact path for a given contract name
func GetArtifactPath(name string) string {
	return getArtifactPath(name)
}

// convertSnakeToPascal converts snake_case to PascalCase
func convertSnakeToPascal(snake string) string {
	parts := strings.Split(snake, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

// loadArtifact loads and parses a foundry artifact with proper source handling
// Returns both the raw artifact and structured metadata with processed sources
func loadArtifact(artifactsFS foundry.StatDirFs, artifactPath string, logger log.Logger) (*foundry.Artifact, *ArtifactMetadata, error) {
	f, err := artifactsFS.Open(artifactPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open artifact %s: %w", artifactPath, err)
	}
	defer f.Close()

	var art foundry.Artifact
	if err := json.NewDecoder(f).Decode(&art); err != nil {
		return nil, nil, fmt.Errorf("failed to decode artifact: %w", err)
	}

	// Process sources with remapping (reusing the better approach from original code)
	sources := make(map[string]SourceContent)
	for sourcePath, sourceInfo := range art.Metadata.Sources {
		remappedKey := art.SearchRemappings(sourcePath)
		sources[remappedKey] = SourceContent{Content: sourceInfo.Content}
		logger.Debug("added source contract", "originalPath", sourcePath, "remappedKey", remappedKey)
	}

	// Extract contract path from compilation target
	var contractPath string
	for path, name := range art.Metadata.Settings.CompilationTarget {
		contractPath = fmt.Sprintf("%s:%s", path, name)
		break
	}

	if contractPath == "" {
		return nil, nil, fmt.Errorf("failed to find compilation target in artifact")
	}

	// Extract compiler version
	compilerVersion := art.Metadata.Compiler.Version
	if compilerVersion == "" {
		return nil, nil, fmt.Errorf("compiler version not found in artifact")
	}

	// Parse optimizer settings
	var optimizer OptimizerSettings
	if len(art.Metadata.Settings.Optimizer) > 0 {
		if err := json.Unmarshal(art.Metadata.Settings.Optimizer, &optimizer); err != nil {
			return nil, nil, fmt.Errorf("failed to parse optimizer settings: %w", err)
		}
	}

	// Extract EVM version
	evmVersion := art.Metadata.Settings.EVMVersion

	metadata := &ArtifactMetadata{
		ContractPath:    contractPath,
		CompilerVersion: compilerVersion,
		Optimizer:       optimizer,
		EVMVersion:      evmVersion,
		Sources:         sources,
	}

	return &art, metadata, nil
}
