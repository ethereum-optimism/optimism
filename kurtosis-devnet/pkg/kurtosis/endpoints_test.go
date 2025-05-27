package kurtosis

import (
	"encoding/json"
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
	t.Run("L2 chain1 services", func(t *testing.T) {
		nodes, services := finder.FindL2Services(chain1)

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

// createTestServiceMap creates a service map based on the provided scenario output
func createTestServiceMap() inspect.ServiceMap {
	services := inspect.ServiceMap{
		// L1 Services - must match pattern expected by triageNode function
		"cl-1-teku-geth": {
			"http":          &descriptors.PortInfo{Port: 32777},
			"metrics":       &descriptors.PortInfo{Port: 32778},
			"tcp-discovery": &descriptors.PortInfo{Port: 32779},
			"udp-discovery": &descriptors.PortInfo{Port: 32769},
		},
		"el-1-geth-teku": {
			"engine-rpc":    &descriptors.PortInfo{Port: 32774},
			"metrics":       &descriptors.PortInfo{Port: 32775},
			"rpc":           &descriptors.PortInfo{Port: 32772},
			"tcp-discovery": &descriptors.PortInfo{Port: 32776},
			"udp-discovery": &descriptors.PortInfo{Port: 32768},
			"ws":            &descriptors.PortInfo{Port: 32773},
		},
		"fileserver": {
			"http": &descriptors.PortInfo{Port: 32771},
		},
		"grafana": {
			"http": &descriptors.PortInfo{Port: 32815},
		},
		"prometheus": {
			"http": &descriptors.PortInfo{Port: 32814},
		},

		// L2 Chain1 Services
		"op-batcher-op-kurtosis-1": {
			"http":    &descriptors.PortInfo{Port: 32791},
			"metrics": &descriptors.PortInfo{Port: 32792},
		},
		"op-proposer-op-kurtosis-1": {
			"http":    &descriptors.PortInfo{Port: 32793},
			"metrics": &descriptors.PortInfo{Port: 32794},
		},
		"op-cl-2151908-1-op-node-op-geth-op-kurtosis-1": {
			"http":          &descriptors.PortInfo{Port: 32785},
			"metrics":       &descriptors.PortInfo{Port: 32786},
			"rpc-interop":   &descriptors.PortInfo{Port: 32788},
			"tcp-discovery": &descriptors.PortInfo{Port: 32787},
			"udp-discovery": &descriptors.PortInfo{Port: 32771},
		},
		"op-el-2151908-1-op-geth-op-node-op-kurtosis-1": {
			"engine-rpc":    &descriptors.PortInfo{Port: 32782},
			"metrics":       &descriptors.PortInfo{Port: 32783},
			"rpc":           &descriptors.PortInfo{Port: 32780},
			"tcp-discovery": &descriptors.PortInfo{Port: 32784},
			"udp-discovery": &descriptors.PortInfo{Port: 32770},
			"ws":            &descriptors.PortInfo{Port: 32781},
		},
		"proxyd-2151908": {
			"http":    &descriptors.PortInfo{Port: 32790},
			"metrics": &descriptors.PortInfo{Port: 32789},
		},

		// L2 Chain2 Services
		"op-batcher-op-kurtosis-2": {
			"http":    &descriptors.PortInfo{Port: 32806},
			"metrics": &descriptors.PortInfo{Port: 32807},
		},
		"op-proposer-op-kurtosis-2": {
			"http":    &descriptors.PortInfo{Port: 32808},
			"metrics": &descriptors.PortInfo{Port: 32809},
		},
		"op-cl-2151909-1-op-node-op-geth-op-kurtosis-2": {
			"http":          &descriptors.PortInfo{Port: 32800},
			"metrics":       &descriptors.PortInfo{Port: 32801},
			"rpc-interop":   &descriptors.PortInfo{Port: 32803},
			"tcp-discovery": &descriptors.PortInfo{Port: 32802},
			"udp-discovery": &descriptors.PortInfo{Port: 32773},
		},
		"op-el-2151909-1-op-geth-op-node-op-kurtosis-2": {
			"engine-rpc":    &descriptors.PortInfo{Port: 32797},
			"metrics":       &descriptors.PortInfo{Port: 32798},
			"rpc":           &descriptors.PortInfo{Port: 32795},
			"tcp-discovery": &descriptors.PortInfo{Port: 32799},
			"udp-discovery": &descriptors.PortInfo{Port: 32772},
			"ws":            &descriptors.PortInfo{Port: 32796},
		},
		"proxyd-2151909": {
			"http":    &descriptors.PortInfo{Port: 32805},
			"metrics": &descriptors.PortInfo{Port: 32804},
		},

		// Shared L2 Services
		"op-faucet": {
			"rpc": &descriptors.PortInfo{Port: 32813},
		},
		"op-challenger-service-2151908-2151909": {
			"metrics": &descriptors.PortInfo{Port: 32812},
		},
		"op-supervisor-service-superchain": {
			"metrics": &descriptors.PortInfo{Port: 32811},
			"rpc":     &descriptors.PortInfo{Port: 32810},
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
		services:        make(inspect.ServiceMap),
		nodeServices:    []string{"cl", "el"},
		l2ServicePrefix: "op-",
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

		// Test L2 node format
		idx, accept, ok = parser("op-cl-2151908-1-op-node-op-geth-op-kurtosis-1")
		assert.True(t, ok, "Should recognize L2 CL node")
		assert.Equal(t, 0, idx, "Should extract index 0 from L2 CL node")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "2151908"}), "Should accept matching chain ID")
		assert.False(t, accept(&spec.ChainSpec{NetworkID: "2151909"}), "Should not accept different chain ID")

		// Test with various suffixes to see what is recognized
		_, _, ok = parser("cl-1-teku-geth-with-extra-parts")
		assert.True(t, ok, "Should recognize L1 CL node regardless of suffix")

		_, _, ok = parser("op-cl-2151908-1-op-node-op-geth-op-kurtosis-1-with-extra-parts")
		assert.True(t, ok, "Should recognize L2 CL node regardless of suffix")

		// This is considered invalid
		_, _, ok = parser("cl")
		assert.False(t, ok, "Should not recognize simple 'cl'")

		_, _, ok = parser("op-cl")
		assert.False(t, ok, "Should not recognize simple 'op-cl'")
	})

	// Test the exclusive L2 service parser (batcher, proposer, proxyd)
	t.Run("triageExclusiveL2Service", func(t *testing.T) {
		parser := finder.triageExclusiveL2Service("op-batcher-")

		// Valid format
		idx, accept, ok := parser("op-batcher-123456")
		assert.True(t, ok, "Should recognize batcher")
		assert.Equal(t, -1, idx, "Exclusive services have -1 index")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "123456"}), "Should accept chain with matching ID")
		assert.False(t, accept(&spec.ChainSpec{NetworkID: "654321"}), "Should not accept chain with different ID")

		// With suffix
		_, _, ok = parser("op-batcher-123456-with-suffix")
		assert.True(t, ok, "Should recognize batcher regardless of suffix")

		// Invalid formats
		_, _, ok = parser("batcher-123456")
		assert.False(t, ok, "Should not recognize batcher without op- prefix")

		_, _, ok = parser("op-batcher")
		assert.False(t, ok, "Should not recognize op-batcher without chain ID")
	})

	// Test the multi-chain service parser (challenger)
	t.Run("triageMultiL2Service", func(t *testing.T) {
		parser := finder.triageMultiL2Service("op-challenger-")

		// Valid format with service identifier and two chain IDs
		idx, accept, ok := parser("op-challenger-any-123456-654321")
		assert.True(t, ok, "Should recognize challenger for two chains")
		assert.Equal(t, -1, idx, "Multi-chain services have -1 index")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "123456"}), "Should accept first chain")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "654321"}), "Should accept second chain")
		assert.False(t, accept(&spec.ChainSpec{NetworkID: "789012"}), "Should not accept unrelated chain")

		// Valid format with service identifier and one chain ID
		_, accept, ok = parser("op-challenger-any-123456")
		assert.True(t, ok, "Should recognize challenger for one chain")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "123456"}), "Should accept the only chain")
		assert.False(t, accept(&spec.ChainSpec{NetworkID: "654321"}), "Should not accept different chain")

		// Invalid formats
		_, _, ok = parser("challenger-123456")
		assert.False(t, ok, "Should not recognize challenger without prefix")

		_, _, ok = parser("op-challenger")
		assert.False(t, ok, "Should not recognize op-challenger without service ID")
	})

	// Test the superchain service parser (supervisor)
	t.Run("triageSuperchainService", func(t *testing.T) {
		// Create some chains for the dependency set
		chain1 := eth.ChainIDFromUInt64(123456)
		chain2 := eth.ChainIDFromUInt64(654321)

		// Create a dependency set
		depSetData := map[eth.ChainID]*depset.StaticConfigDependency{
			chain1: {},
			chain2: {},
		}
		depSet, err := depset.NewStaticConfigDependencySet(depSetData)
		require.NoError(t, err)

		// Serialize dependency set
		jsonData, err := json.Marshal(depSet)
		require.NoError(t, err)

		// Create a new finder with the dependency set
		finderWithDS := &ServiceFinder{
			services:        make(inspect.ServiceMap),
			nodeServices:    []string{"cl", "el"},
			l2ServicePrefix: "op-",
			depsets: map[string]descriptors.DepSet{
				"superchain": descriptors.DepSet(jsonData),
			},
		}

		parser := finderWithDS.triageSuperchainService("op-supervisor-")

		// Valid format - "op-supervisor-{service_id}-{depset_name}"
		idx, accept, ok := parser("op-supervisor-id-superchain")
		assert.True(t, ok, "Should recognize supervisor")
		assert.Equal(t, -1, idx, "Superchain services have -1 index")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "123456"}), "Should accept chain1")
		assert.True(t, accept(&spec.ChainSpec{NetworkID: "654321"}), "Should accept chain2")
		assert.False(t, accept(&spec.ChainSpec{NetworkID: "789012"}), "Should not accept unrelated chain")

		// Invalid formats
		_, _, ok = parser("supervisor-superchain")
		assert.False(t, ok, "Should not recognize supervisor without prefix")

		_, _, ok = parser("op-supervisor")
		assert.False(t, ok, "Should not recognize op-supervisor without service ID and depset name")

		// Test with non-existing depset
		_, _, ok = parser("op-supervisor-id-nonexistent")
		assert.False(t, ok, "Should not recognize supervisor with non-existent depset")
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
