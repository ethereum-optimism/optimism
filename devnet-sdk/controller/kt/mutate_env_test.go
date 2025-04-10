package kt

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/stretchr/testify/require"
)

func TestRedundantServiceRefreshEndpoints(t *testing.T) {
	// Create a test service with some initial endpoints
	svc1 := &descriptors.Service{
		Name: "test-service",
		Endpoints: descriptors.EndpointMap{
			"http": {
				Port:        0,
				PrivatePort: 0,
			},
			"ws": {
				Port:        0,
				PrivatePort: 0,
			},
		},
	}
	svc2 := &descriptors.Service{
		Name: "test-service",
		Endpoints: descriptors.EndpointMap{
			"http": {
				Port:        0,
				PrivatePort: 0,
			},
			"ws": {
				Port:        0,
				PrivatePort: 0,
			},
		},
	}

	// Create a redundant service with both services
	redundant := redundantService{svc1, svc2}

	// Create a test service context with new port numbers
	testCtx := &testServiceContext{
		publicPorts: map[string]interfaces.PortSpec{
			"http": &testPortSpec{number: 8080},
			"ws":   &testPortSpec{number: 8081},
		},
		privatePorts: map[string]interfaces.PortSpec{
			"http": &testPortSpec{number: 8082},
			"ws":   &testPortSpec{number: 8083},
		},
	}

	// Call refreshEndpoints
	redundant.refreshEndpoints(testCtx)

	// Verify that both services have been updated with the new port numbers
	for _, svc := range redundant {
		require.Equal(t, 8080, svc.Endpoints["http"].Port)
		require.Equal(t, 8081, svc.Endpoints["ws"].Port)
		require.Equal(t, 8082, svc.Endpoints["http"].PrivatePort)
		require.Equal(t, 8083, svc.Endpoints["ws"].PrivatePort)
	}
}

func TestRedundantServiceEmpty(t *testing.T) {
	// Test behavior with empty redundant service
	redundant := redundantService{}
	testCtx := &testServiceContext{
		publicPorts:  map[string]interfaces.PortSpec{},
		privatePorts: map[string]interfaces.PortSpec{},
	}

	// This should not panic
	redundant.refreshEndpoints(testCtx)
}

func TestUpdateDevnetEnvironmentService(t *testing.T) {
	// Create a test environment with a service
	env := &descriptors.DevnetEnvironment{
		Name: "test-env",
		L1: &descriptors.Chain{
			Services: descriptors.ServiceMap{
				"test-service": &descriptors.Service{
					Name: "test-service",
					Endpoints: descriptors.EndpointMap{
						"http": {
							Port:        0,
							PrivatePort: 0,
						},
					},
				},
			},
		},
	}

	// Create a test service context with new port numbers
	testSvcCtx := &testServiceContext{
		publicPorts: map[string]interfaces.PortSpec{
			"http": &testPortSpec{number: 8080},
		},
		privatePorts: map[string]interfaces.PortSpec{
			"http": &testPortSpec{number: 8082},
		},
	}

	// Create a mock enclave context with our service
	mockEnclave := &mockEnclaveContext{
		services: map[services.ServiceName]interfaces.ServiceContext{
			"test-service": testSvcCtx,
		},
	}

	// Create a mock kurtosis context with our enclave
	mockKurtosisCtx := &mockKurtosisContext{
		enclaves: map[string]interfaces.EnclaveContext{
			"test-env": mockEnclave,
		},
	}

	// Create the controller surface
	controller := &KurtosisControllerSurface{
		kurtosisCtx: mockKurtosisCtx,
		env:         env,
	}

	// Create the mock DevnetFS
	mockDevnetFS, err := newMockDevnetFS(env)
	require.NoError(t, err)
	controller.devnetfs = mockDevnetFS

	// Test updating the service (turning it on)
	updated, err := controller.updateDevnetEnvironmentService(context.Background(), "test-service", true)
	require.NoError(t, err)
	require.True(t, updated)

	// Verify that the service's endpoints were updated
	svc := findSvcInEnv(env, "test-service")
	require.NotNil(t, svc)
	require.Equal(t, 8080, svc[0].Endpoints["http"].Port)
	require.Equal(t, 8082, svc[0].Endpoints["http"].PrivatePort)

	// Test updating a non-existent service
	updated, err = controller.updateDevnetEnvironmentService(context.Background(), "non-existent-service", true)
	require.NoError(t, err)
	require.False(t, updated)
}
