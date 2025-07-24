package node

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/HashKeyChain/verse/op-node/rollup"
	opmetrics "github.com/HashKeyChain/verse/op-service/metrics"
	oprpc "github.com/HashKeyChain/verse/op-service/rpc"
	"github.com/HashKeyChain/verse/op-supervisor/supervisor/backend/depset"
)

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
	})
	return server
}
