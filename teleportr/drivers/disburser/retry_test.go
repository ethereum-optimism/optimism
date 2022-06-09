package disburser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/stretchr/testify/require"
)

func TestIsRetryableError(t *testing.T) {
	var resCode int32
	var res atomic.Value
	res.Store([]byte{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(atomic.LoadInt32(&resCode)))
		_, _ = w.Write(res.Load().([]byte))
	}))
	defer server.Close()

	client, err := ethclient.Dial(server.URL)
	require.NoError(t, err)

	tests := []struct {
		code      int
		retryable bool
	}{
		{
			503,
			true,
		},
		{
			524,
			true,
		},
		{
			429,
			true,
		},
		{
			500,
			false,
		},
		{
			200,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("http %d", tt.code), func(t *testing.T) {
			atomic.StoreInt32(&resCode, int32(tt.code))
			_, err := client.BlockNumber(context.Background())
			require.Equal(t, tt.retryable, IsRetryableError(err))
		})
	}

	require.True(t, IsRetryableError(context.DeadlineExceeded))
	require.True(t, IsRetryableError(errors.New("read: connection reset by peer")))
}
