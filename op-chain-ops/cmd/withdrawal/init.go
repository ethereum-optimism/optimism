package main

import (
	"fmt"
	"math/big"

	op_service "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/urfave/cli/v2"
)

var (
	L2Flag = &cli.StringFlag{
		Name:    "l2",
		Usage:   "HTTP provider URL for L2.",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "L2"),
	}
	ValueFlag = &cli.StringFlag{
		Name:    "value",
		Usage:   "Value to send with withdrawal",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "VALUE"),
		Value:   "1",
	}
)

func InitWithdrawal(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}
	txMgrConfig := txmgr.ReadCLIConfig(ctx)
	txMgrConfig.L1RPCURL = ctx.String(L2Flag.Name)

	txMgr, err := createTxMgr(ctx, logger, L2Flag.Name)
	if err != nil {
		return err
	}

	valueStr := ctx.String(ValueFlag.Name)
	value, ok := new(big.Int).SetString(valueStr, 10)
	if !ok {
		return fmt.Errorf("invalid value: %s", valueStr)
	}

	rcpt, err := txMgr.Send(ctx.Context, txmgr.TxCandidate{
		To:    &predeploys.L2ToL1MessagePasserAddr,
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("failed to send withdrawal transaction: %w", err)
	}
	// Force printing full hashes
	logger.Info("Send withdrawal", "tx", rcpt.TxHash.Hex(), "blockHash", rcpt.BlockHash.Hex(), "blockNumber", rcpt.BlockNumber)
	return nil
}

func initFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		L2Flag,
		ValueFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, oplog.CLIFlags(EnvVarPrefix)...)
	return cliFlags
}

var InitCommand = &cli.Command{
	Name:   "init",
	Usage:  "Initiates a withdrawal on the L2",
	Action: interruptible(InitWithdrawal),
	Flags:  initFlags(),
}
