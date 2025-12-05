package node

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

<<<<<<< HEAD
type rpcServer struct {
	endpoint   string
	apis       []rpc.API
	httpServer *ophttp.HTTPServer
	appVersion string
	log        log.Logger
	sources.L2Client
	rpcServerTimeout ophttp.HTTPTimeouts
}

func newRPCServer(rpcCfg *RPCConfig, rollupCfg *rollup.Config, l2Client l2EthClient, dr driverClient, safedb SafeDBReader, log log.Logger, appVersion string, m metrics.Metricer) (*rpcServer, error) {
	rpcServerTimeout := ophttp.DefaultTimeouts
	if rpcCfg.ListenTimeout != nil {
		rpcServerTimeout = *rpcCfg.ListenTimeout
	}
	api := NewNodeAPI(rollupCfg, l2Client, dr, safedb, log.New("rpc", "node"), m)
	// TODO: extend RPC config with options for WS, IPC and HTTP RPC connections
	endpoint := net.JoinHostPort(rpcCfg.ListenAddr, strconv.Itoa(rpcCfg.ListenPort))
	r := &rpcServer{
		endpoint: endpoint,
		apis: []rpc.API{{
			Namespace:     "optimism",
			Service:       api,
			Authenticated: false,
		}},
		appVersion:       appVersion,
		log:              log,
		rpcServerTimeout: rpcServerTimeout,
	}
	return r, nil
}

func (s *rpcServer) EnableAdminAPI(api *adminAPI) {
	s.apis = append(s.apis, rpc.API{
		Namespace:     "admin",
		Version:       "",
		Service:       api,
		Authenticated: false,
=======
func newRPCServer(rpcCfg *oprpc.CLIConfig, rollupCfg *rollup.Config, depSet depset.DependencySet, l2Client l2EthClient, dr driverClient,
	safeDB SafeDBReader, log log.Logger, metrics opmetrics.RPCMetricer, appVersion string) *oprpc.Server {
	server := oprpc.NewServer(rpcCfg.ListenAddr, rpcCfg.ListenPort, appVersion,
		oprpc.WithLogger(log),
		oprpc.WithCORSHosts([]string{"*"}), // CORS is not important on op-node, but we used to do this on the old op-node RPC server, so kept for compatibility.
		oprpc.WithRPCRecorder(metrics.NewRecorder("main")),
	)
	api := NewNodeAPI(rollupCfg, depSet, l2Client, dr, safeDB, log)
	server.AddAPI(rpc.API{
		Namespace: "optimism",
		Service:   api,
>>>>>>> upstream/develop
	})
	return server
}
