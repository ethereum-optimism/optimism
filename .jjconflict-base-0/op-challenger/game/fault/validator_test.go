package fault

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

var (
	prestate          = []byte{0x00, 0x01, 0x02, 0x03}
	mockProviderError = fmt.Errorf("mock provider error")
	mockLoaderError   = fmt.Errorf("mock loader error")
)

func TestValidate(t *testing.T) {
	t.Run("ValidPrestates", func(t *testing.T) {
		prestateHash := crypto.Keccak256(prestate)
		prestateHash[0] = mipsevm.VMStatusUnfinished
		player := &PrestateValidator{
			load:     newMockPrestateLoader(false, common.BytesToHash(prestateHash)),
			provider: newMockPrestateProvider(false, prestate),
		}
		err := player.Validate(context.Background())
		require.NoError(t, err)
	})

	t.Run("ProviderErrors", func(t *testing.T) {
		player := &PrestateValidator{
			load:     newMockPrestateLoader(false, common.BytesToHash(prestate)),
			provider: newMockPrestateProvider(true, prestate),
		}
		err := player.Validate(context.Background())
		require.ErrorIs(t, err, mockProviderError)
	})

	t.Run("LoaderErrors", func(t *testing.T) {
		player := &PrestateValidator{
			load:     newMockPrestateLoader(true, common.BytesToHash(prestate)),
			provider: newMockPrestateProvider(false, prestate),
		}
		err := player.Validate(context.Background())
		require.ErrorIs(t, err, mockLoaderError)
	})

	t.Run("PrestateMismatch", func(t *testing.T) {
		player := &PrestateValidator{
			load:     newMockPrestateLoader(false, common.BytesToHash([]byte{0x00})),
			provider: newMockPrestateProvider(false, prestate),
		}
		err := player.Validate(context.Background())
		require.ErrorIs(t, err, gameTypes.ErrInvalidPrestate)
	})
}

var _ types.PrestateProvider = (*mockPrestateProvider)(nil)

type mockPrestateProvider struct {
	prestateErrors bool
	prestate       []byte
}

func newMockPrestateProvider(prestateErrors bool, prestate []byte) *mockPrestateProvider {
	return &mockPrestateProvider{
		prestateErrors: prestateErrors,
		prestate:       prestate,
	}
}

func (m *mockPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	if m.prestateErrors {
		return common.Hash{}, mockProviderError
	}
	hash := common.BytesToHash(crypto.Keccak256(m.prestate))
	hash[0] = mipsevm.VMStatusUnfinished
	return hash, nil
}

func newMockPrestateLoader(prestateError bool, prestate common.Hash) PrestateLoader {
	return func(ctx context.Context) (common.Hash, error) {
		if prestateError {
			return common.Hash{}, mockLoaderError
		}
		return prestate, nil
	}
}
