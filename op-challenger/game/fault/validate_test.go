package fault

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var (
	mockProviderError     = fmt.Errorf("mock provider error")
	mockLoaderError       = fmt.Errorf("mock loader error")
	mockGenesisOutputRoot = common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
)

func TestValidateAbsolutePrestate(t *testing.T) {
	t.Run("ValidPrestates", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		prestateHash := crypto.Keccak256(prestate)
		prestateHash[0] = mipsevm.VMStatusUnfinished
		prestateProvider := newMockPrestateProvider(prestate, common.Hash{})
		mockLoader := newMockPrestateLoader(false, common.BytesToHash(prestateHash))
		err := ValidateAbsolutePrestate(context.Background(), prestateProvider, mockLoader)
		require.NoError(t, err)
	})

	t.Run("ProviderErrors", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		prestateProvider := newMockPrestateProvider(prestate, common.Hash{})
		prestateProvider.prestateErrors = true
		mockLoader := newMockPrestateLoader(false, common.BytesToHash(prestate))
		err := ValidateAbsolutePrestate(context.Background(), prestateProvider, mockLoader)
		require.ErrorIs(t, err, mockProviderError)
	})

	t.Run("LoaderErrors", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		prestateProvider := newMockPrestateProvider(prestate, common.Hash{})
		mockLoader := newMockPrestateLoader(true, common.BytesToHash(prestate))
		err := ValidateAbsolutePrestate(context.Background(), prestateProvider, mockLoader)
		require.ErrorIs(t, err, mockLoaderError)
	})

	t.Run("PrestateMismatch", func(t *testing.T) {
		prestateProvider := newMockPrestateProvider([]byte{0x00, 0x01, 0x02, 0x03}, common.Hash{})
		mockLoader := newMockPrestateLoader(false, common.BytesToHash([]byte{0x00}))
		err := ValidateAbsolutePrestate(context.Background(), prestateProvider, mockLoader)
		require.Error(t, err)
	})
}

func TestValidateGenesisOutputRoot(t *testing.T) {
	t.Run("ValidGenesisOutputRoots", func(t *testing.T) {
		prestateProvider := newMockPrestateProvider(nil, mockGenesisOutputRoot)
		mockLoader := newMockPrestateLoader(false, mockGenesisOutputRoot)
		err := ValidateGenesisOutputRoot(context.Background(), prestateProvider, mockLoader)
		require.NoError(t, err)
	})

	t.Run("ProviderErrors", func(t *testing.T) {
		prestateProvider := newMockPrestateProvider(nil, mockGenesisOutputRoot)
		prestateProvider.genesisErrors = true
		mockLoader := newMockPrestateLoader(false, mockGenesisOutputRoot)
		err := ValidateGenesisOutputRoot(context.Background(), prestateProvider, mockLoader)
		require.ErrorIs(t, err, mockProviderError)
	})

	t.Run("LoaderErrors", func(t *testing.T) {
		prestateProvider := newMockPrestateProvider(nil, mockGenesisOutputRoot)
		mockLoader := newMockPrestateLoader(true, mockGenesisOutputRoot)
		err := ValidateGenesisOutputRoot(context.Background(), prestateProvider, mockLoader)
		require.ErrorIs(t, err, mockLoaderError)
	})

	t.Run("GenesisOutputRootMismatch", func(t *testing.T) {
		prestateProvider := newMockPrestateProvider(nil, mockGenesisOutputRoot)
		mockLoader := newMockPrestateLoader(false, common.BytesToHash([]byte{0x00}))
		err := ValidateGenesisOutputRoot(context.Background(), prestateProvider, mockLoader)
		require.Error(t, err)
	})
}

func newMockPrestateLoader(prestateError bool, prestate common.Hash) func(ctx context.Context) (common.Hash, error) {
	return func(ctx context.Context) (common.Hash, error) {
		if prestateError {
			return common.Hash{}, mockLoaderError
		}
		return prestate, nil
	}
}

type mockPrestateProvider struct {
	prestate       []byte
	genesis        common.Hash
	prestateErrors bool
	genesisErrors  bool
}

func newMockPrestateProvider(prestate []byte, genesis common.Hash) *mockPrestateProvider {
	return &mockPrestateProvider{prestate, genesis, false, false}
}

func (m *mockPrestateProvider) AbsolutePreStateCommitment(ctx context.Context) (common.Hash, error) {
	if m.prestateErrors {
		return common.Hash{}, mockProviderError
	}
	hash := common.BytesToHash(crypto.Keccak256(m.prestate))
	hash[0] = mipsevm.VMStatusUnfinished
	return hash, nil
}

func (m *mockPrestateProvider) GenesisOutputRoot(ctx context.Context) (common.Hash, error) {
	if m.genesisErrors {
		return common.Hash{}, mockProviderError
	}
	return m.genesis, nil
}
