package verify

import (
	"path"
	"strings"
)

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
	"Mips":                        "MIPS64.sol/MIPS64.json",
	"EthLockbox":                  "ETHLockbox.sol/ETHLockbox.json",
}

func getArtifactPath(name string) string {
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
