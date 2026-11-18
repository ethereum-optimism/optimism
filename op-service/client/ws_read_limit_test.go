package client_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type sizedResponseService struct{}

func (sizedResponseService) Sized(n int) string {
	return strings.Repeat("a", n)
}

func startWSServer(t *testing.T) string {
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("test", sizedResponseService{}))
	t.Cleanup(srv.Stop)
	httpSrv := httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
	t.Cleanup(httpSrv.Close)
	return "ws" + strings.TrimPrefix(httpSrv.URL, "http")
}

func callSized(t *testing.T, wsURL string, n int, opts ...client.RPCOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cl, err := client.NewRPC(ctx, testlog.Logger(t, log.LevelInfo), wsURL, opts...)
	require.NoError(t, err)
	defer cl.Close()
	var res string
	if err := cl.CallContext(ctx, &res, "test_sized", n); err != nil {
		return err
	}
	require.Len(t, res, n)
	return nil
}

// TestWebsocketDefaultLimitAboveGethDefault covers both dial sites (eager and lazy) with a
// response above geth's 32 MiB default websocket read limit.
func TestWebsocketDefaultLimitAboveGethDefault(t *testing.T) {
	wsURL := startWSServer(t)
	const aboveGethDefault = 33 * 1024 * 1024
	for name, opts := range map[string][]client.RPCOption{
		"eager": nil,
		"lazy":  {client.WithLazyDial()},
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, callSized(t, wsURL, aboveGethDefault, opts...))
		})
	}
}

// TestWebsocketLimitOverride checks a caller-supplied limit replaces the default and is enforced.
func TestWebsocketLimitOverride(t *testing.T) {
	wsURL := startWSServer(t)
	opt := client.WithGethRPCOptions(rpc.WithWebsocketMessageSizeLimit(1024))
	require.NoError(t, callSized(t, wsURL, 512, opt))
	require.ErrorContains(t, callSized(t, wsURL, 2048, opt), "read limit exceeded")
}
