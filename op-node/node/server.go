package node

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-node/metrics"

	"github.com/ethereum-optimism/optimism/op-node/p2p"

	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// TODO(inphi): add metrics

type rpcServer struct {
	endpoint   string
	apis       []rpc.API
	httpServer *http.Server
	appVersion string
	listenAddr net.Addr
	log        log.Logger
	l2.Source
}

func newRPCServer(ctx context.Context, rpcCfg *RPCConfig, rollupCfg *rollup.Config, l2Client l2EthClient, log log.Logger, appVersion string, m *metrics.Metrics) (*rpcServer, error) {
	api := newNodeAPI(rollupCfg, l2Client, log.New("rpc", "node"), m)
	// TODO: extend RPC config with options for WS, IPC and HTTP RPC connections
	endpoint := net.JoinHostPort(rpcCfg.ListenAddr, strconv.Itoa(rpcCfg.ListenPort))
	r := &rpcServer{
		endpoint: endpoint,
		apis: []rpc.API{{
			Namespace:     "optimism",
			Service:       api,
			Public:        true,
			Authenticated: false,
		}},
		appVersion: appVersion,
		log:        log,
	}
	return r, nil
}

func (s *rpcServer) EnableP2P(backend *p2p.APIBackend) {
	s.apis = append(s.apis, rpc.API{
		Namespace:     p2p.NamespaceRPC,
		Version:       "",
		Service:       backend,
		Public:        true,
		Authenticated: false,
	})
}

func (s *rpcServer) Start() error {
	srv := rpc.NewServer()
	if err := node.RegisterApis(s.apis, nil, srv, true); err != nil {
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

	listener, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		return err
	}
	s.listenAddr = listener.Addr()

	s.httpServer = &http.Server{Handler: mux}
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) { // todo improve error handling
			s.log.Error("http server failed", "err", err)
		}
	}()
	return nil
}

func (r *rpcServer) Stop() {
	_ = r.httpServer.Shutdown(context.Background())
}

func (r *rpcServer) Addr() net.Addr {
	return r.listenAddr
}

func healthzHandler(appVersion string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(appVersion))
	}
}
