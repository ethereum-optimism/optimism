package enclave

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			manager, err := NewKurtosisEnclaveManager(WithKurtosisContext(tt.fakeCtx))
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
