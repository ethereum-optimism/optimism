package fault

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

var (
	errMock = errors.New("mock error")
)

type mockContractCaller struct {
	callResolveErrors bool
}

func (m *mockContractCaller) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if m.callResolveErrors {
		return nil, errMock
	}
	return nil, nil
}

func TestResolver_CallResolve(t *testing.T) {
	t.Run("should call resolve", func(t *testing.T) {
		m := &mockContractCaller{}
		r, err := NewResolver(m, common.Address{})
		require.NoError(t, err)

		res, err := r.CallResolve(context.Background())
		require.NoError(t, err)
		require.True(t, res)
	})

	t.Run("should return error if eth_call fails", func(t *testing.T) {
		m := &mockContractCaller{callResolveErrors: true}
		r, err := NewResolver(m, common.Address{})
		require.NoError(t, err)

		res, err := r.CallResolve(context.Background())
		require.Error(t, err)
		require.False(t, res)
	})
}
