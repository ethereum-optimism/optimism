package flags

import (
	"log"
	"os"
	"path"
)

const (
	EnvVarPrefix             = "DEPLOYER"
	L1RPCURLFlagName         = "l1-rpc-url"
	CacheDirFlagName         = "cache-dir"
	L1ChainIDFlagName        = "l1-chain-id"
	ArtifactsLocatorFlagName = "artifacts-locator"
	L2ChainIDsFlagName       = "l2-chain-ids"
	WorkdirFlagName          = "workdir"
	OutdirFlagName           = "outdir"
	PrivateKeyFlagName       = "private-key"
	IntentTypeFlagName       = "intent-type"
	VerifierAPIKeyFlagName   = "verifier-api-key"
	EtherscanAPIKeyFlagName  = "etherscan-api-key" // Deprecated: use VerifierAPIKeyFlagName
	InputFileFlagName        = "input-file"
	ContractNameFlagName     = "contract-name"
	VerifierTypeFlagName     = "verifier"
	VerifierUrlFlagName      = "verifier-url"
)

func DefaultCacheDir() string {
	var cacheDir string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		cacheDir = ".op-deployer/cache"
		log.Printf("error getting user home directory: %v, using fallback directory: %s\n", err, cacheDir)
	} else {
		cacheDir = path.Join(homeDir, ".op-deployer/cache")
	}

	return cacheDir
}
