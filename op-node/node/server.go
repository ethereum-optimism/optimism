package node

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

func newRPCServer(rpcCfg *RPCConfig, rollupCfg *rollup.Config, l2Client l2EthClient, dr driverClient,
	safeDB SafeDBReader, log log.Logger, metrics opmetrics.RPCMetricer, appVersion string) *oprpc.Server {
	server := oprpc.NewServer(rpcCfg.ListenAddr, rpcCfg.ListenPort, appVersion,
		oprpc.WithLogger(log),
		oprpc.WithCORSHosts([]string{"*"}), // CORS is not important on op-node, but we used to do this on the old op-node RPC server, so kept for compatibility.
		oprpc.WithRPCRecorder(metrics.NewRecorder("main")),
	)
	api := NewNodeAPI(rollupCfg, l2Client, dr, safeDB, log)
	server.AddAPI(rpc.API{
		Namespace: "optimism",
		Service:   api,
	})
	return server
}
