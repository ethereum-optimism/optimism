package main

import (
	"errors"
	"fmt"

	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/urfave/cli/v2"
)

func FinalizeWithdrawal(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}

	txMgr, err := createTxMgr(ctx, logger, L1Flag.Name)
	if err != nil {
		return err
	}

	txHash := common.HexToHash(ctx.String(TxFlag.Name))
	if txHash == (common.Hash{}) {
		return errors.New("must specify tx hash")
	}

	rpcClient, err := rpc.DialContext(ctx.Context, ctx.String(L2Flag.Name))
	if err != nil {
		return fmt.Errorf("failed to connect to L2: %w", err)
	}
	l2Client := ethclient.NewClient(rpcClient)

	rcpt, err := l2Client.TransactionReceipt(ctx.Context, txHash)
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	msgEvent, err := withdrawals.ParseMessagePassed(rcpt)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	portalAddr := common.HexToAddress(ctx.String(PortalAddressFlag.Name))
	txData, err := w3.MustNewFunc("finalizeWithdrawalTransaction((uint256 Nonce, address Sender, address Target, uint256 Value, uint256 GasLimit, bytes Data))", "").EncodeArgs(
		bindingspreview.TypesWithdrawalTransaction{
			Nonce:    msgEvent.Nonce,
			Sender:   msgEvent.Sender,
			Target:   msgEvent.Target,
			Value:    msgEvent.Value,
			GasLimit: msgEvent.GasLimit,
			Data:     msgEvent.Data,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to pack finalizeWithdrawalTransaction: %w", err)
	}

	rcpt, err = txMgr.Send(ctx.Context, txmgr.TxCandidate{
		TxData: txData,
		To:     &portalAddr,
	})
	if err != nil {
		return fmt.Errorf("failed to prove withdrawal: %w", err)
	}
	logger.Info("Finalized withdrawal", "tx", rcpt.TxHash.Hex())
	return nil
}

func finalizeFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		L1Flag,
		L2Flag,
		TxFlag,
		PortalAddressFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, oplog.CLIFlags(EnvVarPrefix)...)
	return cliFlags
}

var FinalizeCommand = &cli.Command{
	Name:   "finalize",
	Usage:  "Finalize a proven withdrawal on the L1",
	Action: interruptible(FinalizeWithdrawal),
	Flags:  finalizeFlags(),
}
