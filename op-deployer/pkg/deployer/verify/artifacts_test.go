package verify

import (
	"testing"
)

func TestGetArtifactPath(t *testing.T) {
	// Test the actual state file contract names that were causing issues
	testCases := map[string]string{
		"superchain_superchain_config_proxy": "Proxy.sol/Proxy.json",
		"superchain_protocol_versions_proxy": "Proxy.sol/Proxy.json",
		"superchain_superchain_config_impl":  "SuperchainConfig.sol/SuperchainConfig.json",
		"implementations_opcm_impl":          "OPContractsManager.sol/OPContractsManager.json",
	}

	for contractName, expectedPath := range testCases {
		actualPath := GetArtifactPath(contractName)
		if actualPath != expectedPath {
			t.Errorf("contract name %q should map to %q, got %q", contractName, expectedPath, actualPath)
		}
	}
}
