package chain_container

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type echoService struct{}

func (echoService) Ping(_ context.Context) (string, error) { return "pong", nil }

func post(t *testing.T, h http.Handler, path, method string) *httptest.ResponseRecorder {
	t.Helper()
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":[]}`)
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// THE DORMANCY GATE for the container half of the change: a container built with no options
// registers nothing on its handler, so a stock supernode's per-chain RPC surface is byte-identical
// to what it was before extra routes existed.
func TestNoExtraRoutesByDefault(t *testing.T) {
	c := &simpleChainContainer{log: createTestLogger(t), chainID: eth.ChainIDFromUInt64(420)}
	require.Empty(t, c.extraRPCRoutes)

	h := oprpc.NewHandler("", oprpc.WithLogger(c.log))
	c.registerExtraRPCRoutes(h)

	// Nothing was mounted, so the sub-route falls through to the handler's root mux and finds no
	// JSON-RPC server there.
	rec := post(t, h, "/claimed", "test_ping")
	require.NotEqual(t, http.StatusOK, rec.Code)
}

// WithExtraRPCRoutes mounts the API at the sub-route and ONLY at the sub-route. The distinction is
// the whole reason the option exists: the chain's own route must keep meaning the chain.
func TestExtraRouteIsMountedAtTheSubRouteOnly(t *testing.T) {
	c := &simpleChainContainer{log: createTestLogger(t), chainID: eth.ChainIDFromUInt64(420)}
	WithExtraRPCRoutes(ExtraRPCRoute{
		Route: "/claimed",
		API:   gethrpc.API{Namespace: "test", Service: echoService{}},
	})(c)
	require.Len(t, c.extraRPCRoutes, 1)

	h := oprpc.NewHandler("", oprpc.WithLogger(c.log))
	c.registerExtraRPCRoutes(h)

	rec := post(t, h, "/claimed", "test_ping")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "pong")

	// The handler's root — which in production carries the virtual node's own namespaces — never
	// learns about it.
	rec = post(t, h, "/", "test_ping")
	require.NotContains(t, rec.Body.String(), "pong")
}

// The handler is rebuilt on every virtual-node restart, so registration has to be idempotent across
// handlers while the service behind it outlives them.
func TestExtraRoutesAreReRegisteredOnAFreshHandler(t *testing.T) {
	c := &simpleChainContainer{log: createTestLogger(t), chainID: eth.ChainIDFromUInt64(420)}
	WithExtraRPCRoutes(ExtraRPCRoute{
		Route: "/claimed",
		API:   gethrpc.API{Namespace: "test", Service: echoService{}},
	})(c)

	for i := 0; i < 3; i++ {
		h := oprpc.NewHandler("", oprpc.WithLogger(c.log))
		c.registerExtraRPCRoutes(h)
		rec := post(t, h, "/claimed", "test_ping")
		require.Equal(t, http.StatusOK, rec.Code, "restart %d", i)
		require.Contains(t, rec.Body.String(), "pong")
	}
}
