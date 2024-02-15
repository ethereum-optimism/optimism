package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	gethRPC "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
	superchain "github.com/ethereum-optimism/optimism/op-superchain"

	"github.com/urfave/cli/v2"
)

var (
	GitCommit    = ""
	GitDate      = ""
	EnvVarPrefix = "OP_SUPERCHAIN"
)

func prefixEnvVars(name string) []string {
	return []string{EnvVarPrefix + "_" + name}
}

func parseMapFlag(input string) (map[string]string, error) {
	result := map[string]string{}
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("Invalid key-value pair: %s\n", pair)
		}
		result[strings.TrimSpace(keyValue[0])] = strings.TrimSpace(keyValue[1])
	}
	return result, nil
}

// Flags
var (
	/** Required SuperchainBackend Flags **/
	L2NodeAddr = &cli.StringFlag{
		Name:    "l2",
		Usage:   "Address of L2 User JSON-RPC endpoint to use (eth namespace required)",
		Value:   "http://127.0.0.1:9545",
		EnvVars: prefixEnvVars("L2_ETH_RPC"),
	}
	L2PeersNodeAddrs = &cli.StringFlag{
		Name:    "l2-peers",
		Usage:   "List of L2 Peers JSON-rpc endpoints. 'chain1=url1,chain2=url2...'",
		Value:   "",
		EnvVars: prefixEnvVars("L2_PEERS_ETH_RPCS"),
	}
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Version = params.VersionWithCommit(GitCommit, GitDate)
	app.Name = "op-superchain"
	app.Usage = "Optimism Superchain Messaging Backend"
	app.Description = "Runs the superchain messaging backend"
	app.Action = cliapp.LifecycleCmd(SuperchainBackendMain)

	logFlags := oplog.CLIFlags(EnvVarPrefix)
	rpcFlags := rpc.CLIFlags(EnvVarPrefix)
	backendFlags := []cli.Flag{L2NodeAddr, L2PeersNodeAddrs}
	app.Flags = append(append(backendFlags, rpcFlags...), logFlags...)

	ctx := opio.WithInterruptBlocker(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("Application Failed", "err", err)
	}
}

func SuperchainBackendMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	m := metrics.With(metrics.NewRegistry())

	cfg := superchain.SuperchainConfig{
		L2NodeAddr:      ctx.String(L2NodeAddr.Name),
		PeerL2NodeAddrs: map[uint64]string{},
	}

	l2PeersMap, err := parseMapFlag(ctx.String(L2PeersNodeAddrs.Name))
	if err != nil {
		return nil, fmt.Errorf("unable to parse list of peers: %w", err)
	}
	for id, url := range l2PeersMap {
		chainId, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse chain id: %w", err)
		}
		cfg.PeerL2NodeAddrs[chainId] = url
	}

	s, err := superchain.NewSuperchainBackend(ctx.Context, log, m, &cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to start superchain backend: %w", err)
	}

	rpcConfig := rpc.ReadCLIConfig(ctx)
	rpcApis := []gethRPC.API{{Namespace: "superchain", Service: s}}
	rpcServer := rpc.NewServer(rpcConfig.ListenAddr, rpcConfig.ListenPort, ctx.App.Version, rpc.WithAPIs(rpcApis))
	return rpc.NewService(log, rpcServer), nil
}
