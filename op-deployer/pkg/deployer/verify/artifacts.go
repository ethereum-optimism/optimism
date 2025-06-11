package verify

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type contractArtifact struct {
	ContractName    string
	CompilerVersion string
	Optimizer       OptimizerSettings
	EVMVersion      string
	Sources         map[string]SourceContent
	ConstructorArgs abi.Arguments
}

// Map state.json struct fields to forge artifact paths
var contractNameExceptions = map[string]string{
	"OptimismPortalImpl":          "OptimismPortal2.sol/OptimismPortal2.json",
	"L1StandardBridgeProxy":       "L1ChugSplashProxy.sol/L1ChugSplashProxy.json",
	"L1CrossDomainMessengerProxy": "ResolvedDelegateProxy.sol/ResolvedDelegateProxy.json",
	"Opcm":                        "OPContractsManager.sol/OPContractsManager.json",
	"OpcmContractsContainer":      "OPContractsManager.sol/OPContractsManagerContractsContainer.json",
	"OpcmGameTypeAdder":           "OPContractsManager.sol/OPContractsManagerGameTypeAdder.json",
	"OpcmDeployer":                "OPContractsManager.sol/OPContractsManagerDeployer.json",
	"OpcmUpgrader":                "OPContractsManager.sol/OPContractsManagerUpgrader.json",
}

func getArtifactPath(name string) string {
	lookupName := strings.TrimSuffix(name, "Address")
	lookupName = strings.ToUpper(string(lookupName[0])) + lookupName[1:]

	if artifactPath, exists := contractNameExceptions[lookupName]; exists {
		return artifactPath
	}

	lookupName = strings.TrimSuffix(lookupName, "Proxy")
	lookupName = strings.TrimSuffix(lookupName, "Impl")
	lookupName = strings.TrimSuffix(lookupName, "Singleton")

	// If it was a proxy and not a special case, return "Proxy"
	if strings.HasSuffix(name, "ProxyAddress") {
		return path.Join("Proxy.sol", "Proxy.json")
	}

	return path.Join(lookupName+".sol", lookupName+".json")
}

func (v *Verifier) getContractArtifact(name string) (*contractArtifact, error) {
	artifactPath := getArtifactPath(name)

	v.log.Info("Opening artifact", "path", artifactPath, "name", name)
	f, err := v.artifactsFS.Open(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open artifact: %w", err)
	}
	defer f.Close()

	var art foundry.Artifact
	if err := json.NewDecoder(f).Decode(&art); err != nil {
		return nil, fmt.Errorf("failed to decode artifact: %w", err)
	}

	// Add all sources (main contract and dependencies)
	sources := make(map[string]SourceContent)
	for sourcePath, sourceInfo := range art.Metadata.Sources {
		remappedKey := art.SearchRemappings(sourcePath)
		sources[remappedKey] = SourceContent{Content: sourceInfo.Content}
		v.log.Debug("added source contract", "originalPath", sourcePath, "remappedKey", remappedKey)
	}

	var optimizer OptimizerSettings
	if err := json.Unmarshal(art.Metadata.Settings.Optimizer, &optimizer); err != nil {
		return nil, fmt.Errorf("failed to parse optimizer settings: %w", err)
	}

	// Get the contract name from the compilation target
	var contractName string
	for contractFile, name := range art.Metadata.Settings.CompilationTarget {
		contractName = contractFile + ":" + name
		break
	}
	v.log.Info("Compilation target", "target", contractName)

	return &contractArtifact{
		ContractName:    contractName,
		CompilerVersion: art.Metadata.Compiler.Version,
		Optimizer:       optimizer,
		EVMVersion:      art.Metadata.Settings.EVMVersion,
		Sources:         sources,
		ConstructorArgs: art.ABI.Constructor.Inputs,
	}, nil
}
