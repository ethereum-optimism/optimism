package enclave

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/fake"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDockerManager implements DockerManager for testing
type MockDockerManager struct{}

func (m *MockDockerManager) DestroyDockerResources(ctx context.Context, enclave ...string) error {
	return nil
}

func TestNewKurtosisEnclaveManager(t *testing.T) {
	tests := []struct {
		name    string
		opts    []KurtosisEnclaveManagerOptions
		wantErr bool
	}{
		{
			name: "create with fake context",
			opts: []KurtosisEnclaveManagerOptions{
				WithKurtosisContext(&fake.KurtosisContext{}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewKurtosisEnclaveManager(tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, manager)
		})
	}
}

func TestGetEnclave(t *testing.T) {
	tests := []struct {
		name      string
		enclave   string
		fakeCtx   *fake.KurtosisContext
		wantErr   bool
		wantCalls []string
	}{
		{
			name:    "get existing enclave",
			enclave: "test-enclave",
			fakeCtx: &fake.KurtosisContext{
				EnclaveCtx: &fake.EnclaveContext{},
			},
			wantErr:   false,
			wantCalls: []string{"get"},
		},
		{
			name:    "create new enclave when not exists",
			enclave: "test-enclave",
			fakeCtx: &fake.KurtosisContext{
				GetErr:     errors.New("enclave not found"),
				EnclaveCtx: &fake.EnclaveContext{},
			},
			wantErr:   false,
			wantCalls: []string{"get", "create"},
		},
		{
			name:    "error on get and create",
			enclave: "test-enclave",
			fakeCtx: &fake.KurtosisContext{
				GetErr:    errors.New("get error"),
				CreateErr: errors.New("create error"),
			},
			wantErr:   true,
			wantCalls: []string{"get", "create"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewKurtosisEnclaveManager(
				WithKurtosisContext(tt.fakeCtx),
				WithDockerManager(&MockDockerManager{}),
			)
			require.NoError(t, err)

			ctx := context.Background()
			enclaveCtx, err := manager.GetEnclave(ctx, tt.enclave)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, enclaveCtx)
		})
	}
}

func TestAutofix(t *testing.T) {
	tests := []struct {
		name          string
		enclave       string
		fakeCtx       *fake.KurtosisContext
		status        interfaces.EnclaveStatus
		statusErr     error
		destroyErr    error
		wantErr       bool
		wantDestroyed bool
	}{
		{
			name:    "running enclave",
			enclave: "test-enclave",
			fakeCtx: &fake.KurtosisContext{},
			status:  interfaces.EnclaveStatusRunning,
			wantErr: false,
		},
		{
			name:          "stopped enclave",
			enclave:       "test-enclave",
			fakeCtx:       &fake.KurtosisContext{},
			status:        interfaces.EnclaveStatusStopped,
			wantErr:       false,
			wantDestroyed: true,
		},
		{
			name:          "empty enclave",
			enclave:       "test-enclave",
			fakeCtx:       &fake.KurtosisContext{},
			status:        interfaces.EnclaveStatusEmpty,
			wantErr:       false,
			wantDestroyed: true,
		},
		{
			name:      "enclave not found",
			enclave:   "test-enclave",
			fakeCtx:   &fake.KurtosisContext{},
			statusErr: errors.New("enclave not found"),
			wantErr:   false,
		},
		{
			name:       "destroy error",
			enclave:    "test-enclave",
			fakeCtx:    &fake.KurtosisContext{},
			status:     interfaces.EnclaveStatusStopped,
			destroyErr: errors.New("destroy error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock context
			tt.fakeCtx.Status = tt.status
			tt.fakeCtx.StatusErr = tt.statusErr
			tt.fakeCtx.DestroyErr = tt.destroyErr

			manager, err := NewKurtosisEnclaveManager(
				WithKurtosisContext(tt.fakeCtx),
				WithDockerManager(&MockDockerManager{}),
			)
			require.NoError(t, err)

			ctx := context.Background()
			err = manager.Autofix(ctx, tt.enclave)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.wantDestroyed {
				assert.True(t, tt.fakeCtx.DestroyCalled, "Destroy should be called")
			} else {
				assert.False(t, tt.fakeCtx.DestroyCalled, "Destroy should not be called")
			}
		})
	}
}

func TestNuke(t *testing.T) {
	tests := []struct {
		name      string
		fakeCtx   *fake.KurtosisContext
		cleanErr  error
		wantErr   bool
		wantClean bool
	}{
		{
			name:      "successful nuke",
			fakeCtx:   &fake.KurtosisContext{},
			wantErr:   false,
			wantClean: true,
		},
		{
			name:     "clean error",
			fakeCtx:  &fake.KurtosisContext{},
			cleanErr: errors.New("clean error"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock context
			tt.fakeCtx.CleanErr = tt.cleanErr

			manager, err := NewKurtosisEnclaveManager(
				WithKurtosisContext(tt.fakeCtx),
				WithDockerManager(&MockDockerManager{}),
			)
			require.NoError(t, err)

			ctx := context.Background()
			err = manager.Nuke(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.wantClean {
				assert.True(t, tt.fakeCtx.CleanCalled, "Clean should be called")
			} else {
				assert.False(t, tt.fakeCtx.CleanCalled, "Clean should not be called")
			}
		})
	}
}
