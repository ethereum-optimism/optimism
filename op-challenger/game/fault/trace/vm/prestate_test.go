package vm

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type stubConverter struct {
	err  error
	hash common.Hash
}

func (s *stubConverter) ConvertStateToProof(_ context.Context, _ string) (*utils.ProofData, uint64, bool, error) {
	if s.err != nil {
		return nil, 0, false, s.err
	}
	return &utils.ProofData{
		ClaimValue: s.hash,
	}, 0, false, nil
}

func newPrestateProvider(prestate common.Hash) *PrestateProvider {
	return NewPrestateProvider("state.json", &stubConverter{hash: prestate})
}

func TestAbsolutePreStateCommitment(t *testing.T) {
	prestate := common.Hash{0xaa, 0xbb}

	t.Run("StateUnavailable", func(t *testing.T) {
		expectedErr := errors.New("kaboom")
		provider := NewPrestateProvider("foo", &stubConverter{err: expectedErr})
		_, err := provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("ExpectedAbsolutePreState", func(t *testing.T) {
		provider := newPrestateProvider(prestate)
		actual, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, prestate, actual)
	})

	t.Run("CacheAbsolutePreState", func(t *testing.T) {
		converter := &stubConverter{hash: prestate}
		provider := NewPrestateProvider(filepath.Join("state.json"), converter)
		first, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)

		// Remove the prestate from disk
		converter.err = errors.New("no soup for you")

		// Value should still be available from cache
		cached, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, first, cached)
	})
}
