package sync_tester_ext_el

import (
	"os"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Configuration defaults for op-sepolia
const (
	DefaultL2NetworkName      = "op-sepolia"
	DefaultL1ChainID          = 11155111
	DefaultL2ELEndpoint       = "https://ci-sepolia-l2.optimism.io"
	DefaultL1CLBeaconEndpoint = "https://ci-sepolia-beacon.optimism.io"
	DefaultL1ELEndpoint       = "https://ci-sepolia-l1.optimism.io"
	DefaultInitialL2Block     = 32012748

	// Tailscale networking endpoints
	DefaultL2ELEndpointTailscale       = "https://proxyd-l2-sepolia.primary.client.dev.oplabs.cloud"
	DefaultL1CLBeaconEndpointTailscale = "https://beacon-api-proxy-sepolia.primary.client.dev.oplabs.cloud"
	DefaultL1ELEndpointTailscale       = "https://proxyd-l1-sepolia.primary.client.dev.oplabs.cloud"
)

var (
	InitialL2Block = getInitialL2Block()

	// Load configuration from environment variables with defaults
	L2NetworkName = getEnvOrDefault("L2_NETWORK_NAME", DefaultL2NetworkName)
	L1ChainID     = eth.ChainIDFromUInt64(getEnvUint64OrDefault("L1_CHAIN_ID", DefaultL1ChainID))

	// Default endpoints
	L2ELEndpoint       = getEnvOrDefault("L2_EL_ENDPOINT", DefaultL2ELEndpoint)
	L1CLBeaconEndpoint = getEnvOrDefault("L1_CL_BEACON_ENDPOINT", DefaultL1CLBeaconEndpoint)
	L1ELEndpoint       = getEnvOrDefault("L1_EL_ENDPOINT", DefaultL1ELEndpoint)
)

func TestMain(m *testing.M) {
	// Override configuration with Tailscale endpoints if Tailscale networking is enabled
	if os.Getenv("TAILSCALE_NETWORKING") == "true" {
		L2ELEndpoint = getEnvOrDefault("L2_EL_ENDPOINT_TAILSCALE", DefaultL2ELEndpointTailscale)
		L1CLBeaconEndpoint = getEnvOrDefault("L1_CL_BEACON_ENDPOINT_TAILSCALE", DefaultL1CLBeaconEndpointTailscale)
		L1ELEndpoint = getEnvOrDefault("L1_EL_ENDPOINT_TAILSCALE", DefaultL1ELEndpointTailscale)
	}

	presets.DoMain(m, presets.WithMinimalExternalELWithSuperchainRegistry(L1CLBeaconEndpoint, L1ELEndpoint, L2ELEndpoint, L1ChainID, L2NetworkName, eth.FCUState{
		Latest:    InitialL2Block,
		Safe:      InitialL2Block,
		Finalized: InitialL2Block,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)

}

// getEnvOrDefault returns the environment variable value or the default if not set
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

// getEnvUint64OrDefault returns the environment variable value as uint64 or the default if not set
func getEnvUint64OrDefault(envVar string, defaultValue uint64) uint64 {
	if value := os.Getenv(envVar); value != "" {
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getInitialL2Block returns the initial L2 block from env var or default
func getInitialL2Block() uint64 {
	return getEnvUint64OrDefault("INITIAL_L2_BLOCK", DefaultInitialL2Block)
}
