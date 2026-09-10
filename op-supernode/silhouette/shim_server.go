package silhouette

import (
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

// Serving. Production uses a standalone listener so this is a pure EL component swap beneath a
// stock op-node/supernode. InProc retains the identical JSON-RPC handlers for focused tests.

// APIs is the RPC surface: the Engine API, the eth_ query set, and the two self-declaration
// namespaces. Nothing else — no admin, no debug, no txpool, no net.
func (s *Shim) APIs() []rpc.API {
	return []rpc.API{
		{Namespace: "engine", Service: &EngineAPI{s: s}, Authenticated: true},
		{Namespace: "eth", Service: &EthAPI{s: s}, Public: true},
		{Namespace: "silhouette", Service: &SilhouetteAPI{s: s}, Public: true},
		{Namespace: "web3", Service: Web3API{}, Public: true},
	}
}

// PublicAPIs is the query surface a silhouette chain owes the network, and only that: the eth_ set
// and the self-declaration namespace. The Engine API is deliberately excluded — it is the private
// channel between this chain's node and its execution client, and publishing a newPayload endpoint
// for a chain nobody may extend would be an invitation with no legitimate caller.
//
// This exists for deployments that expose a query-only endpoint separately from the authenticated
// Engine endpoint. Self-declaration remains visible at the service layer in either shape.
func (s *Shim) PublicAPIs() []rpc.API {
	return []rpc.API{
		{Namespace: "eth", Service: &EthAPI{s: s}, Public: true},
		{Namespace: "silhouette", Service: &SilhouetteAPI{s: s}, Public: true},
		{Namespace: "web3", Service: Web3API{}, Public: true},
	}
}

// InProc registers the shim on a fresh geth RPC server and dials it in-process, returning the client
// an op-node engine client is built on.
func (s *Shim) InProc() (client.RPC, *rpc.Server, error) {
	srv := rpc.NewServer()
	for _, api := range s.APIs() {
		if err := srv.RegisterName(api.Namespace, api.Service); err != nil {
			srv.Stop()
			return nil, nil, err
		}
	}
	return client.NewBaseRPCClient(rpc.DialInProc(srv)), srv, nil
}

// Standalone builds an op-service/rpc server carrying the shim's namespaces. The caller starts and
// stops it, and decides on JWT authentication.
func (s *Shim) Standalone(host string, port int, opts ...oprpc.Option) *oprpc.Server {
	srv := oprpc.NewServer(host, port, ClientVersion, opts...)
	for _, api := range s.APIs() {
		srv.AddAPI(api)
	}
	return srv
}
