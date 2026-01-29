package prestates

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPrestateProviderCache_CreateAndCache(t *testing.T) {
	cache := NewPrestateProviderCache(nil, "", func(_ context.Context, prestateHash common.Hash) (types.PrestateProvider, error) {
		return &stubPrestateProvider{commitment: prestateHash}, nil
	})

	hash1 := common.Hash{0xaa}
	hash2 := common.Hash{0xbb}
	provider1a, err := cache.GetOrCreate(context.Background(), hash1)
	require.NoError(t, err)
	commitment, err := provider1a.AbsolutePreStateCommitment(context.Background())
	require.NoError(t, err)
	require.Equal(t, hash1, commitment)

	provider1b, err := cache.GetOrCreate(context.Background(), hash1)
	require.NoError(t, err)
	require.Same(t, provider1a, provider1b)
	commitment, err = provider1b.AbsolutePreStateCommitment(context.Background())
	require.NoError(t, err)
	require.Equal(t, hash1, commitment)

	provider2, err := cache.GetOrCreate(context.Background(), hash2)
	require.NoError(t, err)
	require.NotSame(t, provider1a, provider2)
	commitment, err = provider2.AbsolutePreStateCommitment(context.Background())
	require.NoError(t, err)
	require.Equal(t, hash2, commitment)
}

func TestPrestateProviderCache_CreateFails(t *testing.T) {
	hash1 := common.Hash{0xaa}
	expectedErr := errors.New("boom")
	cache := NewPrestateProviderCache(nil, "", func(_ context.Context, prestateHash common.Hash) (types.PrestateProvider, error) {
		return nil, expectedErr
	})
	provider, err := cache.GetOrCreate(context.Background(), hash1)
	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, provider)
}

type stubPrestateProvider struct {
	commitment common.Hash
}

func (s *stubPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	return s.commitment, nil
}
