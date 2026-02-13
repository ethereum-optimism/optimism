package main

import (
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
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
