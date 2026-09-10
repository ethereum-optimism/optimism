package claimfollow

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	gethlog "github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
)

const wireChainID = "424243"

// serve reproduces exactly what a chain container does with an ExtraRPCRoute — a fresh per-chain
// oprpc.Handler behind the shared router, with the module mounted at the sibling sub-route — and
// returns the base URL. Everything below is real: the router's path splitting, the handler's
// sub-route mux, geth's JSON-RPC codec and the stock follow client.
func serve(t *testing.T, m *Module) string {
	t.Helper()
	lgr := testlog.Logger(t, gethlog.LevelInfo)
	h := oprpc.NewHandler("", oprpc.WithLogger(lgr))
	require.NoError(t, h.AddRPC("/"+DefaultRoute))
	require.NoError(t, h.AddAPIToRPC("/"+DefaultRoute, gethrpc.API{Namespace: "optimism", Service: NewAPI(m)}))

	router := resources.NewRouter(lgr, resources.RouterConfig{})
	router.SetHandler(wireChainID, h)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv.URL
}

func dialFollow(t *testing.T, url string) *sources.FollowClient {
	t.Helper()
	rpcClient, err := client.NewRPC(context.Background(), testlog.Logger(t, gethlog.LevelInfo), url)
	require.NoError(t, err)
	t.Cleanup(rpcClient.Close)
	follow, err := sources.NewFollowClient(rpcClient)
	require.NoError(t, err)
	return follow
}

// The wire gate: a STOCK follow consumer reads what this serves, at the sibling route. This pins
// the method name, the namespace, the route and every JSON field name the protocol depends on —
// not just the Go values behind them.
func TestStockFollowClientReadsTheClaimedRoute(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.currentL1 = eth.L1BlockRef{Hash: l1Hash(1), Number: 1}
	// Finalized through the claim's TERMINAL block, so all six fields of the finalized ref are
	// finalized-depth facts. See Module.promote.
	h.r.safe, h.r.finalized = 8, 8
	require.NoError(t, h.step())

	base := serve(t, h.f)
	follow := dialFollow(t, base+"/"+wireChainID+"/"+DefaultRoute)

	status, err := follow.GetFollowStatus(ctx)
	require.NoError(t, err)
	// Full-struct equality, over the wire: the consumer compares whole refs, so anything the JSON
	// round trip dropped would show up here.
	require.Equal(t, wantRef(8), status.LocalSafeL2)
	require.Equal(t, wantRef(8), status.SafeL2)
	require.Equal(t, wantRef(8), status.FinalizedL2)
	require.Equal(t, uint64(1), status.CurrentL1.Number)
}

// The not-yet state over the wire: a complete genesis status, every field intact through the JSON
// round trip. This is the response the private chain's op-node — and therefore the operator's
// batcher behind it — gets from the first tick.
func TestStockFollowClientReadsTheGenesisRefBeforeTheFirstClaim(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	h.r.fill(1, 4, "a", 0)
	h.r.currentL1 = eth.L1BlockRef{Hash: l1Hash(1), Number: 1}
	h.r.safe, h.r.finalized = 4, 4
	require.NoError(t, h.step())

	base := serve(t, h.f)
	follow := dialFollow(t, base+"/"+wireChainID+"/"+DefaultRoute)
	status, err := follow.GetFollowStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, wantGenesisRef(), status.LocalSafeL2)
	require.Equal(t, wantGenesisRef(), status.SafeL2)
	require.Equal(t, wantGenesisRef(), status.FinalizedL2)
	// The field the whole bootstrap turns on: op-batcher rejects a status whose CurrentL1 is empty
	// (op-batcher/batcher/sync_actions.go), and in follow mode this module is its only source.
	require.NotEqual(t, eth.L1BlockRef{}, status.CurrentL1)
	require.Equal(t, uint64(1), status.CurrentL1.Number)
}

// The route is DISTINCT, and that is the point of it. A consumer pointed at the chain's own route
// must not receive these refs: the two chains are different chains, and a sequencing LightCL
// force-resets onto whatever it is told.
func TestTheChainsOwnRouteDoesNotServeTheModule(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 8
	require.NoError(t, h.step())

	base := serve(t, h.f)
	follow := dialFollow(t, base+"/"+wireChainID)
	_, err := follow.GetFollowStatus(ctx)
	require.Error(t, err, "the chain's own route serves the RENDERING chain, never the private one")
}
