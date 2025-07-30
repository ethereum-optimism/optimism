package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "WITHDRAWAL"

func setupLogging(ctx *cli.Context) (log.Logger, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	logger := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
	oplog.SetGlobalLogHandler(logger.Handler())
	return logger, nil
}

func interruptible(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		ctx.Context = ctxinterrupt.WithCancelOnInterrupt(ctx.Context)
		return action(ctx)
	}
}

func createTxMgr(ctx *cli.Context, logger log.Logger, rpcUrlFlag string) (*txmgr.SimpleTxManager, error) {
	txMgrConfig := txmgr.ReadCLIConfig(ctx)
	txMgrConfig.L1RPCURL = ctx.String(rpcUrlFlag)
	txMgrConfig.ReceiptQueryInterval = time.Second

	txMgr, err := txmgr.NewSimpleTxManager("challenger", logger, &metrics.NoopTxMetrics{}, txMgrConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create the transaction manager: %w", err)
	}
	return txMgr, nil
}

func createEthClient(ctx *cli.Context, rpcUrlFlag string) (*sources.EthClient, error) {
	url := ctx.String(rpcUrlFlag)
	l1RPC, err := client.NewRPC(ctx.Context, log.Root(), url, client.WithDialAttempts(10))
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}
	l1Client, err := sources.NewEthClient(l1RPC, log.Root(), nil, sources.DefaultEthClientConfig(10))
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 client: %w", err)
	}
	return l1Client, nil
}

func loadRollupConfig(ctx *cli.Context, rollupConfigFlag string) (*rollup.Config, error) {
	file, err := os.Open(ctx.String(rollupConfigFlag))
	if err != nil {
		return nil, fmt.Errorf("failed to open rollup config file: %w", err)
	}
	defer file.Close()
	var rollupConfig rollup.Config
	return &rollupConfig, rollupConfig.ParseRollupConfig(file)
}

func loadDepsetConfig(ctx *cli.Context, depSetFlag string) (depset.DependencySet, error) {
	data, err := os.ReadFile(ctx.String(depSetFlag))
	if err != nil {
		return nil, fmt.Errorf("failed to read depset config: %w", err)
	}
	var depsetConfig depset.StaticConfigDependencySet
	err = json.Unmarshal(data, &depsetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse depset config: %w", err)
	}
	return &depsetConfig, nil
}

func createSupervisorClient(ctx *cli.Context, supervisorFlag string) (*sources.SupervisorClient, error) {
	cl, err := dial.DialSupervisorClientWithTimeout(ctx.Context, log.Root(), ctx.String(supervisorFlag))
	if err != nil {
		return nil, fmt.Errorf("failed to dial supervisor: %w", err)
	}
	return cl, nil
}
