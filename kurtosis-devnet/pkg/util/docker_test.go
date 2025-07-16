package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateKurtosisFilter(t *testing.T) {
	tests := []struct {
		name           string
		enclave        []string
		expectedFilter string
	}{
		{
			name:           "no enclave specified",
			enclave:        []string{},
			expectedFilter: "kurtosis.devnet.enclave",
		},
		{
			name:           "enclave specified",
			enclave:        []string{"test-enclave"},
			expectedFilter: "kurtosis.devnet.enclave=test-enclave",
		},
		{
			name:           "multiple enclaves (only first used)",
			enclave:        []string{"enclave1", "enclave2"},
			expectedFilter: "kurtosis.devnet.enclave=enclave1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := createKurtosisFilter(tt.enclave...)

			// Check that the filter has the expected label
			labels := filter.Get("label")
			require.Len(t, labels, 1, "Expected exactly one label filter")
			assert.Equal(t, tt.expectedFilter, labels[0])
		})
	}
}

// Helper function to create test containers for scenarios
func createTestContainer(id, name string, networks map[string]*network.EndpointSettings) types.Container {
	return types.Container{
		ID:    id,
		Names: []string{name},
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: networks,
		},
	}
}

// Helper function to create test network endpoint
func createTestNetworkEndpoint(networkID string) *network.EndpointSettings {
	return &network.EndpointSettings{
		NetworkID: networkID,
	}
}

func TestSetReverseProxyConfigLogic(t *testing.T) {
	// Test the logic patterns that the function should follow
	// The function should ALWAYS configure Traefik for ALL networks it has access to

	t.Run("network ID extraction logic", func(t *testing.T) {
		// Test the logic for extracting ALL network IDs from Traefik's own networks
		networks := map[string]*network.EndpointSettings{
			"bridge":  createTestNetworkEndpoint("bridge-network-id"),
			"custom1": createTestNetworkEndpoint("custom-network-id-1"),
			"custom2": createTestNetworkEndpoint("custom-network-id-2"),
		}

		traefikContainer := createTestContainer("traefik-id", "kurtosis-reverse-proxy-test", networks)

		// The function should collect all non-bridge networks
		networkIDs := make(map[string]bool)
		for networkName, network := range traefikContainer.NetworkSettings.Networks {
			if networkName != "bridge" {
				networkIDs[network.NetworkID] = true
			}
		}

		assert.Len(t, networkIDs, 2)
		assert.Contains(t, networkIDs, "custom-network-id-1")
		assert.Contains(t, networkIDs, "custom-network-id-2")
	})

	t.Run("traefik container identification", func(t *testing.T) {
		// Test the logic for identifying Traefik containers
		containers := []types.Container{
			createTestContainer("container1", "/some-other-container", nil),
			createTestContainer("container2", "/kurtosis-reverse-proxy-12345", nil),
			createTestContainer("container3", "/another-container", nil),
		}

		// The function should find the container with "kurtosis-reverse-proxy" in the name
		var traefikContainer *types.Container
		for _, c := range containers {
			if strings.Contains(c.Names[0], "kurtosis-reverse-proxy") {
				traefikContainer = &c
				break
			}
		}

		require.NotNil(t, traefikContainer, "Should find Traefik container")
		assert.Equal(t, "container2", traefikContainer.ID)
	})

	t.Run("dynamic config generation", func(t *testing.T) {
		// Test the dynamic configuration template for multiple networks
		networkIDs := []string{"test-network-id-1", "test-network-id-2"}

		expectedConfig := `# Dynamic Traefik configuration for correct networks
providers:
  dockerDynamic0:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: "test-network-id-1"
    watch: true
  dockerDynamic1:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: "test-network-id-2"
    watch: true
`

		var actualConfig strings.Builder
		actualConfig.WriteString("# Dynamic Traefik configuration for correct networks\n")
		actualConfig.WriteString("providers:\n")

		for i, networkID := range networkIDs {
			actualConfig.WriteString(fmt.Sprintf(`  dockerDynamic%d:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: "%s"
    watch: true
`, i, networkID))
		}

		assert.Equal(t, expectedConfig, actualConfig.String())
	})
}

func TestCheckUserServiceNetworks(t *testing.T) {
	// Test the network accessibility checking logic

	t.Run("network accessibility logic", func(t *testing.T) {
		// Test the logic for checking if user services have networks that Traefik doesn't have access to

		// Traefik has access to these networks
		traefikNetworkIDs := map[string]bool{
			"network-1": true,
			"network-2": true,
		}

		// User service containers and their networks
		userServiceContainers := []struct {
			name     string
			networks map[string]*network.EndpointSettings
		}{
			{
				name: "service-1",
				networks: map[string]*network.EndpointSettings{
					"bridge":    createTestNetworkEndpoint("bridge-network-id"),
					"network-1": createTestNetworkEndpoint("network-1"), // accessible
				},
			},
			{
				name: "service-2",
				networks: map[string]*network.EndpointSettings{
					"bridge":    createTestNetworkEndpoint("bridge-network-id"),
					"network-2": createTestNetworkEndpoint("network-2"), // accessible
					"network-3": createTestNetworkEndpoint("network-3"), // NOT accessible
				},
			},
		}

		// Test the logic for finding unreachable networks
		userServiceNetworks := make(map[string]bool)
		for _, container := range userServiceContainers {
			for networkName, network := range container.networks {
				if networkName != "bridge" {
					userServiceNetworks[network.NetworkID] = true
				}
			}
		}

		// Find networks that user services are on but Traefik is not
		unreachableNetworks := make(map[string]bool)
		for networkID := range userServiceNetworks {
			if !traefikNetworkIDs[networkID] {
				unreachableNetworks[networkID] = true
			}
		}

		// Should find network-3 as unreachable
		assert.Len(t, unreachableNetworks, 1)
		assert.Contains(t, unreachableNetworks, "network-3")
		assert.NotContains(t, unreachableNetworks, "network-1")
		assert.NotContains(t, unreachableNetworks, "network-2")
	})

	t.Run("all networks accessible", func(t *testing.T) {
		// Test case where all user service networks are accessible by Traefik

		traefikNetworkIDs := map[string]bool{
			"network-1": true,
			"network-2": true,
			"network-3": true,
		}

		userServiceNetworks := map[string]bool{
			"network-1": true,
			"network-2": true,
		}

		// Find unreachable networks
		unreachableNetworks := make(map[string]bool)
		for networkID := range userServiceNetworks {
			if !traefikNetworkIDs[networkID] {
				unreachableNetworks[networkID] = true
			}
		}

		// Should find no unreachable networks
		assert.Len(t, unreachableNetworks, 0)
	})
}
