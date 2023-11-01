package node

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	ophttp "github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type rpcServer struct {
	endpoint   string
	apis       []rpc.API
	httpServer *ophttp.HTTPServer
	appVersion string
	log        log.Logger
	sources.L2Client
}

func newRPCServer(ctx context.Context, rpcCfg *RPCConfig, rollupCfg *rollup.Config, l2Client l2EthClient, dr driverClient, log log.Logger, appVersion string, m metrics.Metricer) (*rpcServer, error) {
	api := NewNodeAPI(rollupCfg, l2Client, dr, log.New("rpc", "node"), m)
	// TODO: extend RPC config with options for WS, IPC and HTTP RPC connections
	endpoint := net.JoinHostPort(rpcCfg.ListenAddr, strconv.Itoa(rpcCfg.ListenPort))
	r := &rpcServer{
		endpoint: endpoint,
		apis: []rpc.API{{
			Namespace:     "optimism",
			Service:       api,
			Authenticated: false,
		}},
		appVersion: appVersion,
		log:        log,
	}
	return r, nil
}

func (s *rpcServer) EnableAdminAPI(api *adminAPI) {
	s.apis = append(s.apis, rpc.API{
		Namespace:     "admin",
		Version:       "",
		Service:       api,
		Authenticated: false,
	})
}

func (s *rpcServer) EnableP2P(backend *p2p.APIBackend) {
	s.apis = append(s.apis, rpc.API{
		Namespace:     p2p.NamespaceRPC,
		Version:       "",
		Service:       backend,
		Authenticated: false,
	})
}

func (s *rpcServer) Start() error {
	srv := rpc.NewServer()
	if err := node.RegisterApis(s.apis, nil, srv); err != nil {
		return err
	}

	// The CORS and VHosts arguments below must be set in order for
	// other services to connect to the opnode. VHosts in particular
	// defaults to localhost, which will prevent containers from
	// calling into the opnode without an "invalid host" error.
	nodeHandler := node.NewHTTPHandlerStack(srv, []string{"*"}, []string{"*"}, nil)

	mux := http.NewServeMux()
	mux.Handle("/", nodeHandler)
	mux.HandleFunc("/healthz", healthzHandler(s.appVersion))

	hs, err := ophttp.StartHTTPServer(s.endpoint, mux)
	if err != nil {
		return fmt.Errorf("failed to start HTTP RPC server: %w", err)
	}
	s.httpServer = hs
	return nil
}

func (r *rpcServer) Stop(ctx context.Context) error {
	return r.httpServer.Stop(ctx)
}

func (r *rpcServer) Addr() net.Addr {
	return r.httpServer.Addr()
}

func healthzHandler(appVersion string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(appVersion))
	}
}
