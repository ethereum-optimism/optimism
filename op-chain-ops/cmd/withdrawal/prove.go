package main

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/urfave/cli/v2"
)

var (
	L1Flag = &cli.StringFlag{
		Name:    "l1",
		Usage:   "HTTP provider URL for L1.",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "L1"),
	}
	TxFlag = &cli.StringFlag{
		Name:    "tx",
		Usage:   "Transaction hash of withdrawal on L2",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "TX"),
	}
	PortalAddressFlag = &cli.StringFlag{
		Name:    "portal-address",
		Usage:   "Address of the optimism portal contract.",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "PORTAL_ADDRESS"),
	}
)

func ProveWithdrawal(ctx *cli.Context) error {
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
	proofClient := gethclient.New(rpcClient)
	l2Client := ethclient.NewClient(rpcClient)

	l1Client, err := ethclient.DialContext(ctx.Context, ctx.String(L1Flag.Name))
	if err != nil {
		return fmt.Errorf("failed to connect to L1: %w", err)
	}

	rcpt, err := l2Client.TransactionReceipt(ctx.Context, txHash)
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	portalAddr := common.HexToAddress(ctx.String(PortalAddressFlag.Name))
	portal, err := bindingspreview.NewOptimismPortal2(portalAddr, l1Client)
	if err != nil {
		return fmt.Errorf("failed to bind portal: %w", err)
	}
	factoryAddr, err := portal.DisputeGameFactory(&bind.CallOpts{Context: ctx.Context})
	if err != nil {
		return fmt.Errorf("failed to fetch dispute game factory address from portal: %w", err)
	}

	factory, err := bindings.NewDisputeGameFactoryCaller(factoryAddr, l1Client)
	if err != nil {
		return fmt.Errorf("failed to bind dispute game factory: %w", err)
	}

	_, err = wait.ForGamePublished(ctx.Context, l1Client, portalAddr, factoryAddr, rcpt.BlockNumber)
	if err != nil {
		return fmt.Errorf("could not find a dispute game at or above l2 block number %v: %w", rcpt.BlockNumber, err)
	}

	params, err := withdrawals.ProveWithdrawalParametersFaultProofs(ctx.Context, proofClient, l2Client, l2Client, txHash, factory, &portal.OptimismPortal2Caller)
	if err != nil {
		return fmt.Errorf("could not create withdrawal proof parameters: %w", err)
	}

	txData, err := w3.MustNewFunc("proveWithdrawalTransaction("+
		"(uint256 Nonce, address Sender, address Target, uint256 Value, uint256 GasLimit, bytes Data),"+
		"uint256,"+
		"(bytes32 Version, bytes32 StateRoot, bytes32 MessagePasserStorageRoot, bytes32 LatestBlockhash),"+
		"bytes[])", "").EncodeArgs(
		bindingspreview.TypesWithdrawalTransaction{
			Nonce:    params.Nonce,
			Sender:   params.Sender,
			Target:   params.Target,
			Value:    params.Value,
			GasLimit: params.GasLimit,
			Data:     params.Data,
		},
		params.L2OutputIndex,
		params.OutputRootProof,
		params.WithdrawalProof,
	)
	if err != nil {
		return fmt.Errorf("failed to pack withdrawal transaction: %w", err)
	}

	rcpt, err = txMgr.Send(ctx.Context, txmgr.TxCandidate{
		TxData: txData,
		To:     &portalAddr,
	})
	if err != nil {
		return fmt.Errorf("failed to prove withdrawal: %w", err)
	}

	logger.Info("Proved withdrawal", "tx", rcpt.TxHash.Hex())
	return nil
}

func proveFlags() []cli.Flag {
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

var ProveCommand = &cli.Command{
	Name:   "prove",
	Usage:  "Prove a withdrawal on the L1",
	Action: interruptible(ProveWithdrawal),
	Flags:  proveFlags(),
}
