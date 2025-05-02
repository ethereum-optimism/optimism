package kt

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/fake"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/run"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKurtosisControllerSurface(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	// Create a test environment
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

	// Create a test service context with port data
	testSvcCtx := &testServiceContext{
		publicPorts: map[string]interfaces.PortSpec{
			"http": &testPortSpec{number: 8080},
		},
		privatePorts: map[string]interfaces.PortSpec{
			"http": &testPortSpec{number: 8082},
		},
	}

	tests := []struct {
		name        string
		serviceName string
		operation   string // "start" or "stop"
		runErr      error
		wantErr     bool
	}{
		{
			name:        "successful service start",
			serviceName: "test-service",
			operation:   "start",
			runErr:      nil,
			wantErr:     false,
		},
		{
			name:        "service already running",
			serviceName: "test-service",
			operation:   "start",
			runErr:      errors.New("is already in use by container"),
			wantErr:     false,
		},
		{
			name:        "error starting service",
			serviceName: "test-service",
			operation:   "start",
			runErr:      testErr,
			wantErr:     true,
		},
		{
			name:        "successful service stop",
			serviceName: "test-service",
			operation:   "stop",
			runErr:      nil,
			wantErr:     false,
		},
		{
			name:        "error stopping service",
			serviceName: "test-service",
			operation:   "stop",
			runErr:      testErr,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake enclave context that will return our test service context
			fakeEnclaveCtx := &fake.EnclaveContext{
				RunErr: tt.runErr,
				Services: map[services.ServiceName]interfaces.ServiceContext{
					"test-service": testSvcCtx,
				},
			}

			// Create a fake Kurtosis context that will return our fake enclave context
			fakeCtx := &fake.KurtosisContext{
				EnclaveCtx: fakeEnclaveCtx,
			}

			// Create a KurtosisRunner with our fake context
			runner, err := run.NewKurtosisRunner(
				run.WithKurtosisRunnerEnclave("test-enclave"),
				run.WithKurtosisRunnerKurtosisContext(fakeCtx),
			)
			require.NoError(t, err)

			// Create the controller surface with all required fields
			surface := &KurtosisControllerSurface{
				env:         env,
				kurtosisCtx: fakeCtx,
				runner:      runner,
			}

			// Create the mock DevnetFS
			mockDevnetFS, err := newMockDevnetFS(env)
			require.NoError(t, err)
			surface.devnetfs = mockDevnetFS

			switch tt.operation {
			case "start":
				err = surface.StartService(ctx, tt.serviceName)
			case "stop":
				err = surface.StopService(ctx, tt.serviceName)
			default:
				t.Fatalf("unknown operation: %s", tt.operation)
			}

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// For successful start operations, verify that the service endpoints were updated
			if tt.operation == "start" && !tt.wantErr {
				svc := findSvcInEnv(env, tt.serviceName)
				require.NotNil(t, svc)
				require.Equal(t, 8080, svc[0].Endpoints["http"].Port)
				require.Equal(t, 8082, svc[0].Endpoints["http"].PrivatePort)
			}
		})
	}
}
