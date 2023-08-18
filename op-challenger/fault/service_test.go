package fault

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
)

var (
	mockTraceProviderError = fmt.Errorf("mock trace provider error")
	mockLoaderError        = fmt.Errorf("mock loader error")
)

// TestValidateAbsolutePrestate tests that the absolute prestate is validated
// correctly by the service component.
func TestValidateAbsolutePrestate(t *testing.T) {
	t.Run("ValidPrestates", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		prestateHash := mockStateHash(prestate)
		mockTraceProvider := newMockTraceProvider(false, prestate)
		mockLoader := newMockLoader(false, prestateHash)
		err := ValidateAbsolutePrestate(context.Background(), mockTraceProvider, mockLoader)
		require.NoError(t, err)
	})

	t.Run("TraceProviderErrors", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		mockTraceProvider := newMockTraceProvider(true, prestate)
		mockLoader := newMockLoader(false, mockStateHash(prestate))
		err := ValidateAbsolutePrestate(context.Background(), mockTraceProvider, mockLoader)
		require.ErrorIs(t, err, mockTraceProviderError)
	})

	t.Run("LoaderErrors", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		mockTraceProvider := newMockTraceProvider(false, prestate)
		mockLoader := newMockLoader(true, mockStateHash(prestate))
		err := ValidateAbsolutePrestate(context.Background(), mockTraceProvider, mockLoader)
		require.ErrorIs(t, err, mockLoaderError)
	})

	t.Run("PrestateMismatch", func(t *testing.T) {
		mockTraceProvider := newMockTraceProvider(false, []byte{0x00, 0x01, 0x02, 0x03})
		mockLoader := newMockLoader(false, common.Hash{0x00})
		err := ValidateAbsolutePrestate(context.Background(), mockTraceProvider, mockLoader)
		require.Error(t, err)
	})
}

type mockTraceProvider struct {
	prestateErrors bool
	prestate       []byte
}

func newMockTraceProvider(prestateErrors bool, prestate []byte) *mockTraceProvider {
	return &mockTraceProvider{
		prestateErrors: prestateErrors,
		prestate:       prestate,
	}
}
func (m *mockTraceProvider) Get(ctx context.Context, i uint64) (common.Hash, error) {
	panic("not implemented")
}
func (m *mockTraceProvider) GetStepData(ctx context.Context, i uint64) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	panic("not implemented")
}
func (m *mockTraceProvider) AbsolutePreState(ctx context.Context) ([]byte, error) {
	if m.prestateErrors {
		return nil, mockTraceProviderError
	}
	return m.prestate, nil
}

func mockStateHash(state []byte) common.Hash {
	h := crypto.Keccak256Hash(state)
	h[0] = types.VMStatusValid
	return h
}

func (m *mockTraceProvider) StateHash(ctx context.Context, state []byte) (common.Hash, error) {
	return mockStateHash(state), nil
}

type mockLoader struct {
	prestateError bool
	prestateHash  common.Hash
}

func newMockLoader(prestateError bool, prestateHash common.Hash) *mockLoader {
	return &mockLoader{
		prestateError: prestateError,
		prestateHash:  prestateHash,
	}
}
func (m *mockLoader) FetchClaims(ctx context.Context) ([]types.Claim, error) {
	panic("not implemented")
}
func (m *mockLoader) FetchGameDepth(ctx context.Context) (uint64, error) {
	panic("not implemented")
}
func (m *mockLoader) FetchAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	if m.prestateError {
		return common.Hash{}, mockLoaderError
	}
	return m.prestateHash, nil
}
