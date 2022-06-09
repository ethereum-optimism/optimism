package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type NoopChainDataReader struct {
	CallCount int
	Data      *ChainData
}

func (n *NoopChainDataReader) Get(ctx context.Context) (*ChainData, error) {
	n.CallCount++
	return &ChainData{
		MaxBalance:             n.Data.MaxBalance,
		DisburserBalance:       n.Data.DisburserBalance,
		NextDisbursementID:     n.Data.NextDisbursementID,
		DepositContractBalance: n.Data.DepositContractBalance,
		NextDepositID:          n.Data.NextDepositID,
		MaxDepositAmount:       n.Data.MaxDepositAmount,
		MinDepositAmount:       n.Data.MinDepositAmount,
	}, nil
}

func TestCachingChainDataReaderGet(t *testing.T) {
	inner := &NoopChainDataReader{
		Data: &ChainData{
			NextDisbursementID: 1,
		},
	}
	require.Equal(t, inner.CallCount, 0)
	cdr := NewCachingChainDataReader(inner, 5*time.Millisecond)
	data, err := cdr.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, inner.CallCount)
	require.NotNil(t, data)
	inner.Data = &ChainData{
		NextDisbursementID: 2,
	}
	data, err = cdr.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, inner.CallCount)
	require.EqualValues(t, data.NextDisbursementID, 1)
	time.Sleep(10 * time.Millisecond)
	data, err = cdr.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, inner.CallCount)
	require.EqualValues(t, data.NextDisbursementID, 2)
}
