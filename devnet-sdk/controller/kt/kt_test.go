package kt

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/fake"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/run"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKurtosisControllerSurface(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

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
			// Create a fake Kurtosis context that will return our test error
			fakeCtx := &fake.KurtosisContext{
				EnclaveCtx: &fake.EnclaveContext{
					RunErr: tt.runErr,
				},
			}

			// Create a KurtosisRunner with our fake context
			runner, err := run.NewKurtosisRunner(
				run.WithKurtosisRunnerEnclave("test-enclave"),
				run.WithKurtosisRunnerKurtosisContext(fakeCtx),
			)
			require.NoError(t, err)

			// Create the controller surface
			surface := &KurtosisControllerSurface{
				runner: runner,
			}

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
		})
	}
}
