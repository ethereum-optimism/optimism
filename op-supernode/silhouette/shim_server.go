package silhouette

import (
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

// Serving. Two shapes, one set of handlers.
//
// LOOPBACK / IN-PROCESS is the supernode topology: the verifier node and its execution client are
// the same binary, so the "engine RPC" is a Go call through a geth in-proc pipe. That is not a
// shortcut around the RPC — it is the whole RPC, JSON codec included, with the socket removed. Every
// gate in this package runs over it, so the marshalling is exercised exactly as a socket would
// exercise it.
//
// STANDALONE is a real listener, for an operator pointing a stock op-node at the shim as its L2
// engine endpoint. The Engine API normally sits behind a JWT; that is the op-service/rpc server's
// business (oprpc.WithJWTSecret) and a caller's choice, not this package's.

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
// This exists because embedding the shim in a supernode REMOVES a service the network otherwise has:
// on a chain with a real execution client, `eth_getBlockByNumber` is answered by that client at its
// own endpoint. Here the client is in-process, so without this the chain's blocks are unreachable
// and — more to the point — G2 D8's ruling that self-declaration lives at the SERVICE layer would
// have nothing to declare to.
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
