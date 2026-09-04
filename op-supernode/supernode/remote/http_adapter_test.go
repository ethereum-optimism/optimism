package remote_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote"
)

func TestHTTPAdapterErrorOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	adapter := remote.NewHTTPAdapter(eth.ChainIDFromUInt64(10), srv.URL, srv.Client())
	_, ok, err := adapter.NextFinalized(context.Background(), 0)
	require.Error(t, err)
	require.False(t, ok)
}

func TestHTTPAdapterNullBlockIsNotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blockTime":2,"block":null}`))
	}))
	defer srv.Close()

	adapter := remote.NewHTTPAdapter(eth.ChainIDFromUInt64(10), srv.URL, srv.Client())
	blk, ok, err := adapter.NextFinalized(context.Background(), 5)
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, blk)
	require.Equal(t, uint64(2), adapter.BlockTime(), "blockTime is cached even when no block is returned")
}
