package main

import (
	"context"
	"fmt"
	"os"
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

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Version = params.VersionWithCommit(GitCommit, GitDate)
	app.Name = "op-superchain"
	app.Usage = "Optimism Superchain Backend"
	app.Description = "Runs the superchain backend"
	app.Action = cliapp.LifecycleCmd(SuperchainBackendMain)

	logFlags := oplog.CLIFlags(EnvVarPrefix)
	rpcFlags := rpc.CLIFlags(EnvVarPrefix)
	backendFlags := superchain.CLIFlags(EnvVarPrefix)
	app.Flags = append(append(backendFlags, rpcFlags...), logFlags...)

	ctx := opio.WithInterruptBlocker(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("Application Failed", "err", err)
	}
}

func SuperchainBackendMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	m := metrics.With(metrics.NewRegistry())

	cfg, err := superchain.ReadCLIConfig(ctx).Config()
	if err != nil {
		return nil, fmt.Errorf("unable to parse superchain flags: %w", err)
	}

	backend, err := superchain.NewBackend(ctx.Context, log, m, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to start superchain backend: %w", err)
	}

	rpcConfig := rpc.ReadCLIConfig(ctx)
	rpcApis := []gethRPC.API{{Namespace: "superchain", Service: backend}}
	rpcOpts := []rpc.ServerOption{rpc.WithAPIs(rpcApis), rpc.WithLogger(log)}

	rpcServer := rpc.NewServer(rpcConfig.ListenAddr, rpcConfig.ListenPort, ctx.App.Version, rpcOpts...)
	return rpc.NewService(log, rpcServer), nil
}
