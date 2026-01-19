package sync_tester_ext_el

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Configuration defaults for op-sepolia
const (
	DefaultNetworkPreset = "op-sepolia"

	// Tailscale networking endpoints
	DefaultL2ELEndpointTailscale       = "https://proxyd-l2-sepolia.primary.client.dev.oplabs.cloud"
	DefaultL1CLBeaconEndpointTailscale = "https://beacon-api-proxy-sepolia.primary.client.dev.oplabs.cloud"
	DefaultL1ELEndpointTailscale       = "https://proxyd-l1-sepolia.primary.client.dev.oplabs.cloud"
)

var (
	// Network presets for different networks against which we test op-node syncing
	networkPresets = map[string]stack.ExtNetworkConfig{
		"op-sepolia": {
			L2NetworkName:      "op-sepolia",
			L1ChainID:          eth.ChainIDFromUInt64(11155111),
			L2ELEndpoint:       "https://ci-sepolia-l2.optimism.io",
			L1CLBeaconEndpoint: "https://ci-sepolia-beacon.optimism.io",
			L1ELEndpoint:       "https://ci-sepolia-l1.optimism.io",
		},
		"base-sepolia": {
			L2NetworkName:      "base-sepolia",
			L1ChainID:          eth.ChainIDFromUInt64(11155111),
			L2ELEndpoint:       "https://base-sepolia-rpc.optimism.io",
			L1CLBeaconEndpoint: "https://ci-sepolia-beacon.optimism.io",
			L1ELEndpoint:       "https://ci-sepolia-l1.optimism.io",
		},
		"unichain-sepolia": {
			L2NetworkName:      "unichain-sepolia",
			L1ChainID:          eth.ChainIDFromUInt64(11155111),
			L2ELEndpoint:       "https://unichain-sepolia-rpc.optimism.io",
			L1CLBeaconEndpoint: "https://ci-sepolia-beacon.optimism.io",
			L1ELEndpoint:       "https://ci-sepolia-l1.optimism.io",
		},
		"op-mainnet": {
			L2NetworkName:      "op-mainnet",
			L1ChainID:          eth.ChainIDFromUInt64(1),
			L2ELEndpoint:       "https://op-mainnet-rpc.optimism.io",
			L1CLBeaconEndpoint: "https://ci-mainnet-beacon.optimism.io",
			L1ELEndpoint:       "https://ci-mainnet-l1.optimism.io",
		},
		"base-mainnet": {
			L2NetworkName:      "base-mainnet",
			L1ChainID:          eth.ChainIDFromUInt64(1),
			L2ELEndpoint:       "https://base-mainnet-rpc.optimism.io",
			L1CLBeaconEndpoint: "https://ci-mainnet-beacon.optimism.io",
			L1ELEndpoint:       "https://ci-mainnet-l1.optimism.io",
		},
	}
)

func GetNetworkPreset(name string) (stack.ExtNetworkConfig, error) {
	var config stack.ExtNetworkConfig
	if name == "" {
		config = networkPresets[DefaultNetworkPreset]
	} else {
		var ok bool
		config, ok = networkPresets[name]
		if !ok {
			return stack.ExtNetworkConfig{}, fmt.Errorf("NETWORK_PRESET %s not found", name)
		}
	}
	// Override configuration with Tailscale endpoints if Tailscale networking is enabled
	if os.Getenv("TAILSCALE_NETWORKING") == "true" {
		config.L2ELEndpoint = getEnvOrDefault("L2_EL_ENDPOINT_TAILSCALE", DefaultL2ELEndpointTailscale)
		config.L1CLBeaconEndpoint = getEnvOrDefault("L1_CL_BEACON_ENDPOINT_TAILSCALE", DefaultL1CLBeaconEndpointTailscale)
		config.L1ELEndpoint = getEnvOrDefault("L1_EL_ENDPOINT_TAILSCALE", DefaultL1ELEndpointTailscale)
	}
	return config, nil
}

// getEnvOrDefault returns the environment variable value or the default if not set
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}
