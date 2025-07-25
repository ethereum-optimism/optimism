package kurtosis

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/inspect"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/spec"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindChainServices(t *testing.T) {
	// Create test chains based on the scenario
	chain1 := &spec.ChainSpec{
		Name:      "op-kurtosis-1",
		NetworkID: "2151908",
	}
	chain2 := &spec.ChainSpec{
		Name:      "op-kurtosis-2",
		NetworkID: "2151909",
	}
	chains := []*spec.ChainSpec{chain1, chain2}

	// Create mock dependency set
	depSets := createTestDepSets(t)

	// Create mock service map based on inspect data from the scenario
	services := createTestServiceMap()

	// Create service finder with the test data
	finder := NewServiceFinder(
		services,
		WithL2Chains(chains),
		WithDepSets(depSets),
	)

	// Test triage directly to ensure services are correctly triaged
	t.Run("triage services", func(t *testing.T) {
		assert.NotNil(t, finder.triagedServices, "Triaged services should not be nil")
		assert.NotEmpty(t, finder.triagedServices, "Triaged services should not be empty")

		// Count service types
		tagCount := make(map[string]int)
		for _, svc := range finder.triagedServices {
			tagCount[svc.tag]++
		}

		// Verify expected service counts
		assert.Equal(t, 3, tagCount["cl"], "Should have 3 CL services")
		assert.Equal(t, 3, tagCount["el"], "Should have 3 EL service")
		assert.Equal(t, 2, tagCount["batcher"], "Should have 2 batcher services")
		assert.Equal(t, 2, tagCount["proposer"], "Should have 2 proposer services")
		assert.Equal(t, 2, tagCount["proxyd"], "Should have 2 proxyd services")
		assert.Equal(t, 1, tagCount["challenger"], "Should have 1 challenger service")
		assert.Equal(t, 1, tagCount["supervisor"], "Should have 1 supervisor service")
		assert.Equal(t, 1, tagCount["faucet"], "Should have 1 faucet service")
	})

	// Test L1 service discovery
	t.Run("L1 services", func(t *testing.T) {
		nodes, services := finder.FindL1Services()

		// Verify L1 nodes
		assert.Equal(t, 1, len(nodes), "Should have exactly 1 node")

		// Verify L1 services
		assert.Equal(t, 1, len(services), "Should have exactly 1 service")
		assert.Contains(t, services, "faucet", "Should have faucet service")
	})

	// Test L2 services for both chains
	for _, chain := range chains {
		t.Run(fmt.Sprintf("L2 %s services", chain.Name), func(t *testing.T) {
			nodes, services := finder.FindL2Services(chain)

			assert.Equal(t, 1, len(nodes), "Should have exactly 1 node")
			assert.Equal(t, 6, len(services), "Should have exactly 6 services")

			assert.Contains(t, services, "batcher", "Should have batcher service")
			assert.Contains(t, services, "proposer", "Should have proposer service")
			assert.Contains(t, services, "proxyd", "Should have proxyd service")
			assert.Contains(t, services, "challenger", "Should have challenger service")
			assert.Contains(t, services, "supervisor", "Should have supervisor service")
			assert.Contains(t, services, "faucet", "Should have faucet service")
		})
	}
}

// createTestServiceMap creates a service map based on the provided scenario output
func createTestServiceMap() inspect.ServiceMap {
	services := inspect.ServiceMap{
		// L1 Services - must match pattern expected by triageNode function
		"cl-1-teku-geth": &inspect.Service{
			Ports: inspect.PortMap{
				"http":          &descriptors.PortInfo{Port: 32777},
				"metrics":       &descriptors.PortInfo{Port: 32778},
				"tcp-discovery": &descriptors.PortInfo{Port: 32779},
				"udp-discovery": &descriptors.PortInfo{Port: 32769},
			},
		},
		"el-1-geth-teku": &inspect.Service{
			Ports: inspect.PortMap{
				"engine-rpc":    &descriptors.PortInfo{Port: 32774},
				"metrics":       &descriptors.PortInfo{Port: 32775},
				"rpc":           &descriptors.PortInfo{Port: 32772},
				"tcp-discovery": &descriptors.PortInfo{Port: 32776},
				"udp-discovery": &descriptors.PortInfo{Port: 32768},
				"ws":            &descriptors.PortInfo{Port: 32773},
			},
		},
		"fileserver": &inspect.Service{
			Ports: inspect.PortMap{
				"http": &descriptors.PortInfo{Port: 32771},
			},
		},
		"grafana": &inspect.Service{
			Ports: inspect.PortMap{
				"http": &descriptors.PortInfo{Port: 32815},
			},
		},
		"prometheus": &inspect.Service{
			Ports: inspect.PortMap{
				"http": &descriptors.PortInfo{Port: 32814},
			},
		},

		// L2 Chain1 Services
		"op-batcher-op-kurtosis-1": &inspect.Service{
			Ports: inspect.PortMap{
				"http":    &descriptors.PortInfo{Port: 32791},
				"metrics": &descriptors.PortInfo{Port: 32792},
			},
			Labels: map[string]string{
				kindLabel:      "batcher",
				networkIDLabel: "2151908",
			},
		},
		"op-proposer-op-kurtosis-1": &inspect.Service{
			Ports: inspect.PortMap{
				"http":    &descriptors.PortInfo{Port: 32793},
				"metrics": &descriptors.PortInfo{Port: 32794},
			},
			Labels: map[string]string{
				kindLabel:      "proposer",
				networkIDLabel: "2151908",
			},
		},
		"op-cl-2151908-1": &inspect.Service{
			Ports: inspect.PortMap{
				"http":          &descriptors.PortInfo{Port: 32785},
				"metrics":       &descriptors.PortInfo{Port: 32786},
				"rpc-interop":   &descriptors.PortInfo{Port: 32788},
				"tcp-discovery": &descriptors.PortInfo{Port: 32787},
				"udp-discovery": &descriptors.PortInfo{Port: 32771},
			},
			Labels: map[string]string{
				kindLabel:      "cl",
				networkIDLabel: "2151908",
				nodeIndexLabel: "0",
			},
		},
		"op-el-2151908-1": &inspect.Service{
			Ports: inspect.PortMap{
				"engine-rpc":    &descriptors.PortInfo{Port: 32782},
				"metrics":       &descriptors.PortInfo{Port: 32783},
				"rpc":           &descriptors.PortInfo{Port: 32780},
				"tcp-discovery": &descriptors.PortInfo{Port: 32784},
				"udp-discovery": &descriptors.PortInfo{Port: 32770},
				"ws":            &descriptors.PortInfo{Port: 32781},
			},
			Labels: map[string]string{
				kindLabel:      "el",
				networkIDLabel: "2151908",
				nodeIndexLabel: "0",
			},
		},
		"proxyd-2151908": &inspect.Service{
			Ports: inspect.PortMap{
				"http":    &descriptors.PortInfo{Port: 32790},
				"metrics": &descriptors.PortInfo{Port: 32789},
			},
			Labels: map[string]string{
				kindLabel:      "proxyd",
				networkIDLabel: "2151908",
			},
		},

		// L2 Chain2 Services
		"op-batcher-op-kurtosis-2": &inspect.Service{
			Ports: inspect.PortMap{
				"http":    &descriptors.PortInfo{Port: 32806},
				"metrics": &descriptors.PortInfo{Port: 32807},
			},
			Labels: map[string]string{
				kindLabel:      "batcher",
				networkIDLabel: "2151909",
			},
		},
		"op-proposer-op-kurtosis-2": &inspect.Service{
			Ports: inspect.PortMap{
				"http":    &descriptors.PortInfo{Port: 32808},
				"metrics": &descriptors.PortInfo{Port: 32809},
			},
			Labels: map[string]string{
				kindLabel:      "proposer",
				networkIDLabel: "2151909",
			},
		},
		"op-cl-2151909-1": &inspect.Service{
			Ports: inspect.PortMap{
				"http":          &descriptors.PortInfo{Port: 32800},
				"metrics":       &descriptors.PortInfo{Port: 32801},
				"rpc-interop":   &descriptors.PortInfo{Port: 32803},
				"tcp-discovery": &descriptors.PortInfo{Port: 32802},
				"udp-discovery": &descriptors.PortInfo{Port: 32773},
			},
			Labels: map[string]string{
				kindLabel:      "cl",
				networkIDLabel: "2151909",
				nodeIndexLabel: "0",
			},
		},
		"op-el-2151909-1": &inspect.Service{
			Ports: inspect.PortMap{
				"engine-rpc":    &descriptors.PortInfo{Port: 32797},
				"metrics":       &descriptors.PortInfo{Port: 32798},
				"rpc":           &descriptors.PortInfo{Port: 32795},
				"tcp-discovery": &descriptors.PortInfo{Port: 32799},
				"udp-discovery": &descriptors.PortInfo{Port: 32772},
				"ws":            &descriptors.PortInfo{Port: 32796},
			},
			Labels: map[string]string{
				kindLabel:      "el",
				networkIDLabel: "2151909",
				nodeIndexLabel: "0",
			},
		},
		"proxyd-2151909": &inspect.Service{
			Ports: inspect.PortMap{
				"http":    &descriptors.PortInfo{Port: 32805},
				"metrics": &descriptors.PortInfo{Port: 32804},
			},
			Labels: map[string]string{
				kindLabel:      "proxyd",
				networkIDLabel: "2151909",
			},
		},

		// Shared L2 Services
		"op-faucet": &inspect.Service{
			Ports: inspect.PortMap{
				"rpc": &descriptors.PortInfo{Port: 32813},
			},
			Labels: map[string]string{
				kindLabel: "faucet",
			},
		},
		"challenger-service": &inspect.Service{ // intentionally not following conventions, to force use of labels.
			Ports: inspect.PortMap{
				"metrics": &descriptors.PortInfo{Port: 32812},
			},
			Labels: map[string]string{
				kindLabel:      "challenger",
				networkIDLabel: "2151908-2151909",
			},
		},
		"op-supervisor-service-superchain": &inspect.Service{
			Ports: inspect.PortMap{
				"metrics": &descriptors.PortInfo{Port: 32811},
				"rpc":     &descriptors.PortInfo{Port: 32810},
			},
			Labels: map[string]string{
				kindLabel:                 "supervisor",
				supervisorSuperchainLabel: "superchain",
			},
		},
		"validator-key-generation-cl-validator-keystore": {},
	}

	return services
}

// createTestDepSets creates test dependency sets for the test
func createTestDepSets(t *testing.T) map[string]descriptors.DepSet {
	// Create the dependency set for the superchain
	depSetData := map[eth.ChainID]*depset.StaticConfigDependency{
		eth.ChainIDFromUInt64(2151908): {},
		eth.ChainIDFromUInt64(2151909): {},
	}

	depSet, err := depset.NewStaticConfigDependencySet(depSetData)
	require.NoError(t, err)

	jsonData, err := json.Marshal(depSet)
	require.NoError(t, err)

	return map[string]descriptors.DepSet{
		"superchain": descriptors.DepSet(jsonData),
	}
}

// TestTriageFunctions tests the actual implementation of triage functions
func TestTriageFunctions(t *testing.T) {
	// Create a minimal finder with default values
	finder := &ServiceFinder{
		services: make(inspect.ServiceMap),
	}

	// Test the triageNode function for recognizing services
	t.Run("triageNode", func(t *testing.T) {
		// Test CL node parser
		parser := finder.triageNode("cl-")

		// Test L1 node format
		idx, accept, ok := parser("cl-1-teku-geth")
		assert.True(t, ok, "Should recognize L1 CL node")
		assert.Equal(t, 0, idx, "Should extract index 0 from L1 CL node")
		assert.True(t, accept(&spec.ChainSpec{Name: l1Placeholder}), "Should accept L1")

		// Test with various suffixes to see what is recognized
		_, _, ok = parser("cl-1-teku-geth-with-extra-parts")
		assert.True(t, ok, "Should recognize L1 CL node regardless of suffix")

		// This is considered invalid
		_, _, ok = parser("cl")
		assert.False(t, ok, "Should not recognize simple 'cl'")
	})

	// Test the universal L2 service parser (faucet)
	t.Run("triageUniversalL2Service", func(t *testing.T) {
		parser := finder.triageUniversalL2Service("op-faucet")

		// Valid format
		idx, accept, ok := parser("op-faucet")
		assert.True(t, ok, "Should recognize faucet")
		assert.Equal(t, -1, idx, "Universal services have -1 index")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "123456"}), "Should accept any chain")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "654321"}), "Should accept any chain")

		// Invalid formats
		_, _, ok = parser("faucet")
		assert.False(t, ok, "Should not recognize faucet without prefix")

		_, _, ok = parser("op-faucet-with-suffix")
		assert.False(t, ok, "Should not recognize op-faucet with suffix")
	})
}
