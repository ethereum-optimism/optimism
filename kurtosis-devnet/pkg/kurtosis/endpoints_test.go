package kurtosis

import (
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/inspect"
	"github.com/stretchr/testify/assert"
)

func TestFindRPCEndpoints(t *testing.T) {
	testServices := make(inspect.ServiceMap)

	testServices["el-1-geth-lighthouse"] = inspect.PortMap{
		"metrics":       {Port: 52643},
		"tcp-discovery": {Port: 52644},
		"udp-discovery": {Port: 51936},
		"engine-rpc":    {Port: 52642},
		"rpc":           {Port: 52645},
		"ws":            {Port: 52646},
	}

	testServices["op-batcher-op-kurtosis"] = inspect.PortMap{
		"http": {Port: 53572},
	}

	testServices["op-cl-1-op-node-op-geth-op-kurtosis"] = inspect.PortMap{
		"udp-discovery": {Port: 50990},
		"http":          {Port: 53503},
		"tcp-discovery": {Port: 53504},
	}

	testServices["op-el-1-op-geth-op-node-op-kurtosis"] = inspect.PortMap{
		"udp-discovery": {Port: 53233},
		"engine-rpc":    {Port: 53399},
		"metrics":       {Port: 53400},
		"rpc":           {Port: 53402},
		"ws":            {Port: 53403},
		"tcp-discovery": {Port: 53401},
	}

	testServices["vc-1-geth-lighthouse"] = inspect.PortMap{
		"metrics": {Port: 53149},
	}

	testServices["cl-1-lighthouse-geth"] = inspect.PortMap{
		"metrics":       {Port: 52691},
		"tcp-discovery": {Port: 52692},
		"udp-discovery": {Port: 58275},
		"http":          {Port: 52693},
	}

	tests := []struct {
		name         string
		services     inspect.ServiceMap
		findFn       func(*ServiceFinder) ([]descriptors.Node, descriptors.ServiceMap)
		wantNodes    []descriptors.Node
		wantServices descriptors.ServiceMap
	}{
		{
			name:     "find L1 endpoints",
			services: testServices,
			findFn: func(f *ServiceFinder) ([]descriptors.Node, descriptors.ServiceMap) {
				return f.FindL1Services()
			},
			wantNodes: []descriptors.Node{
				{
					Services: descriptors.ServiceMap{
						"cl": descriptors.Service{
							Name: "cl-1-lighthouse-geth",
							Endpoints: descriptors.EndpointMap{
								"metrics":       {Port: 52691},
								"tcp-discovery": {Port: 52692},
								"udp-discovery": {Port: 58275},
								"http":          {Port: 52693},
							},
						},
						"el": descriptors.Service{
							Name: "el-1-geth-lighthouse",
							Endpoints: descriptors.EndpointMap{
								"metrics":       {Port: 52643},
								"tcp-discovery": {Port: 52644},
								"udp-discovery": {Port: 51936},
								"engine-rpc":    {Port: 52642},
								"rpc":           {Port: 52645},
								"ws":            {Port: 52646},
							},
						},
					},
				},
			},
			wantServices: descriptors.ServiceMap{},
		},
		{
			name:     "find op-kurtosis L2 endpoints",
			services: testServices,
			findFn: func(f *ServiceFinder) ([]descriptors.Node, descriptors.ServiceMap) {
				return f.FindL2Services("op-kurtosis")
			},
			wantNodes: []descriptors.Node{
				{
					Services: descriptors.ServiceMap{
						"cl": descriptors.Service{
							Name: "op-cl-1-op-node-op-geth-op-kurtosis",
							Endpoints: descriptors.EndpointMap{
								"udp-discovery": {Port: 50990},
								"http":          {Port: 53503},
								"tcp-discovery": {Port: 53504},
							},
						},
						"el": descriptors.Service{
							Name: "op-el-1-op-geth-op-node-op-kurtosis",
							Endpoints: descriptors.EndpointMap{
								"udp-discovery": {Port: 53233},
								"engine-rpc":    {Port: 53399},
								"metrics":       {Port: 53400},
								"tcp-discovery": {Port: 53401},
								"rpc":           {Port: 53402},
								"ws":            {Port: 53403},
							},
						},
					},
				},
			},
			wantServices: descriptors.ServiceMap{
				"batcher": descriptors.Service{
					Name: "op-batcher-op-kurtosis",
					Endpoints: descriptors.EndpointMap{
						"http": {Port: 53572},
					},
				},
			},
		},
		{
			name: "custom host in endpoint",
			services: inspect.ServiceMap{
				"op-batcher-custom-host": inspect.PortMap{
					"http": {Host: "custom.host", Port: 8080},
				},
			},
			findFn: func(f *ServiceFinder) ([]descriptors.Node, descriptors.ServiceMap) {
				return f.FindL2Services("custom-host")
			},
			wantNodes: nil,
			wantServices: descriptors.ServiceMap{
				"batcher": descriptors.Service{
					Name: "op-batcher-custom-host",
					Endpoints: descriptors.EndpointMap{
						"http": {Host: "custom.host", Port: 8080},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finder := NewServiceFinder(tt.services, WithL2Networks([]string{"op-kurtosis", "network1", "network2", "custom-host"}))
			gotNodes, gotServices := tt.findFn(finder)
			assert.Equal(t, tt.wantNodes, gotNodes)
			assert.Equal(t, tt.wantServices, gotServices)
		})
	}
}

func TestFindL2ServicesSkipsOtherNetworks(t *testing.T) {
	// Create a service map with services from multiple L2 networks
	services := inspect.ServiceMap{
		// network1 services
		"op-batcher-network1": inspect.PortMap{
			"http": {Port: 8080},
		},
		"op-proposer-network1": inspect.PortMap{
			"http": {Port: 8082},
		},
		"op-cl-1-op-node-op-geth-network1": inspect.PortMap{
			"http": {Port: 8084},
		},

		// network2 services
		"op-batcher-network2": inspect.PortMap{
			"http": {Port: 8081},
		},
		"op-proposer-network2": inspect.PortMap{
			"http": {Port: 8083},
		},
		"op-cl-1-op-node-op-geth-network2": inspect.PortMap{
			"http": {Port: 8085},
		},

		// network3 services
		"op-batcher-network3": inspect.PortMap{
			"http": {Port: 8086},
		},
		"op-proposer-network3": inspect.PortMap{
			"http": {Port: 8087},
		},
		"op-cl-1-op-node-op-geth-network3": inspect.PortMap{
			"http": {Port: 8088},
		},

		// Common service without network suffix
		"op-common-service": inspect.PortMap{
			"http": {Port: 8089},
		},
	}

	// Create a service finder with all networks configured
	finder := NewServiceFinder(
		services,
		WithL2Networks([]string{"network1", "network2", "network3"}),
	)

	// Test finding services for network2
	t.Run("find network2 services", func(t *testing.T) {
		nodes, serviceMap := finder.FindL2Services("network2")

		// Verify nodes
		assert.Len(t, nodes, 1)
		if len(nodes) > 0 {
			assert.Contains(t, nodes[0].Services, "cl")
			assert.Equal(t, "op-cl-1-op-node-op-geth-network2", nodes[0].Services["cl"].Name)
		}

		// Verify services
		assert.Len(t, serviceMap, 3) // batcher, proposer, common-service
		assert.Contains(t, serviceMap, "batcher")
		assert.Contains(t, serviceMap, "proposer")
		assert.Contains(t, serviceMap, "common-service")
		assert.Equal(t, "op-batcher-network2", serviceMap["batcher"].Name)
		assert.Equal(t, "op-proposer-network2", serviceMap["proposer"].Name)
		assert.Equal(t, "op-common-service", serviceMap["common-service"].Name)

		// Verify network1 and network3 services are not included
		for _, service := range serviceMap {
			assert.NotContains(t, service.Name, "network1")
			assert.NotContains(t, service.Name, "network3")
		}
	})

	// Test with a network that doesn't exist
	t.Run("find non-existent network services", func(t *testing.T) {
		nodes, serviceMap := finder.FindL2Services("non-existent")

		// Should only find common services
		assert.Len(t, nodes, 0)
		assert.Len(t, serviceMap, 1)
		assert.Contains(t, serviceMap, "common-service")
		assert.Equal(t, "op-common-service", serviceMap["common-service"].Name)
	})
}

func TestServiceTag(t *testing.T) {
	finder := NewServiceFinder(inspect.ServiceMap{})

	tests := []struct {
		name      string
		input     string
		wantTag   string
		wantIndex int
	}{
		{
			name:      "simple service without index",
			input:     "batcher",
			wantTag:   "batcher",
			wantIndex: 0,
		},
		{
			name:      "service with index 1",
			input:     "node-1",
			wantTag:   "node",
			wantIndex: 1,
		},
		{
			name:      "service with index 2",
			input:     "node-2",
			wantTag:   "node",
			wantIndex: 2,
		},
		{
			name:      "service with double digit index",
			input:     "node-10",
			wantTag:   "node",
			wantIndex: 10,
		},
		{
			name:      "service with index in the middle",
			input:     "node-1-suffix",
			wantTag:   "node",
			wantIndex: 1,
		},
		{
			name:      "service with multiple hyphens",
			input:     "multi-part-name-1",
			wantTag:   "multi-part-name",
			wantIndex: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTag, gotIndex := finder.serviceTag(tt.input)
			assert.Equal(t, tt.wantTag, gotTag)
			assert.Equal(t, tt.wantIndex, gotIndex)
		})
	}
}
