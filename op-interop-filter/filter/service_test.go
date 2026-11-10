package filter

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// chainIDAPI serves eth_chainId and fails the first failures calls.
type chainIDAPI struct {
	failures atomic.Int32
}

func (a *chainIDAPI) ChainId() (hexutil.Uint64, error) {
	if a.failures.Add(-1) >= 0 {
		return 0, errors.New("node not ready")
	}
	return 901, nil
}

// newChainIDServer returns an HTTP server for api that stops when the test ends.
func newChainIDServer(t *testing.T, api *chainIDAPI) *http.Server {
	t.Helper()
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", api))
	httpServer := &http.Server{Handler: server}
	t.Cleanup(func() { _ = httpServer.Close() })
	return httpServer
}

// serveChainID serves api over HTTP on ln until the test ends.
func serveChainID(t *testing.T, ln net.Listener, api *chainIDAPI) {
	t.Helper()
	httpServer := newChainIDServer(t, api)
	go func() { _ = httpServer.Serve(ln) }()
}

func TestDialL2RPC_RetriesChainID(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	api := &chainIDAPI{}
	api.failures.Store(startAttempts - 1)
	serveChainID(t, ln, api)

	rpcClient, chainID, err := dialL2RPC(context.Background(), testlog.Logger(t, log.LevelError),
		"http://"+ln.Addr().String(), time.Millisecond)
	require.NoError(t, err)
	t.Cleanup(rpcClient.Close)
	require.Equal(t, eth.ChainIDFromUInt64(901), chainID)
}

func TestDialL2RPC_FailsWhenChainIDStaysUnavailable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	api := &chainIDAPI{}
	api.failures.Store(startAttempts)
	serveChainID(t, ln, api)

	_, _, err = dialL2RPC(context.Background(), testlog.Logger(t, log.LevelError),
		"http://"+ln.Addr().String(), time.Millisecond)
	require.ErrorContains(t, err, "node not ready")
	require.Zero(t, api.failures.Load(), "dialL2RPC must try exactly startAttempts times")
}
