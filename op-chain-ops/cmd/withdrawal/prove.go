package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	opnode_bindings "github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
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

	// Prove using SuperRoots Flags
	SupervisorFlag = &cli.StringFlag{
		Name:    "supervisor",
		Usage:   "HTTP provider URL for supervisor. Only required for proving using super roots.",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "SUPERVISOR"),
	}
	DepSetFlag = &cli.StringFlag{
		Name:    "depset",
		Usage:   "Path to the dependency set file. Only required for proving using super roots.",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "DEPSET"),
	}
	RollupConfigFlag = &cli.StringFlag{
		Name:    "rollup.config",
		Usage:   "Path to the rollup config of the target chain. Only required for proving using super roots.",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "ROLLUP_CONFIG"),
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

	factory, err := opnode_bindings.NewDisputeGameFactoryCaller(factoryAddr, l1Client)
	if err != nil {
		return fmt.Errorf("failed to bind dispute game factory: %w", err)
	}

	_, err = wait.ForGamePublished(ctx.Context, l1Client, portalAddr, factoryAddr, rcpt.BlockNumber)
	if err != nil {
		return fmt.Errorf("could not find a dispute game at or above l2 block number %v: %w", rcpt.BlockNumber, err)
	}

	l1EthClient, err := createEthClient(ctx, L1Flag.Name)
	if err != nil {
		return fmt.Errorf("failed to create L1 eth client: %w", err)
	}
	boundPortal := bindings.NewBindings[bindings.OptimismPortal2](bindings.WithClient(l1EthClient), bindings.WithTo(portalAddr))
	usesSuperRoots, err := contractio.Read(boundPortal.SuperRootsActive(), ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to fetch uses super roots from portal: %w", err)
	}

	var txData []byte
	if !usesSuperRoots {
		logger.Info("Proving withdrawal using output root proof")
		txData, err = txDataForOutputRootProof(ctx.Context, proofClient, l2Client, txHash, factory, portal)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Proving withdrawal using super root proof")
		txData, err = txDataForSuperRootProof(ctx, l1EthClient, proofClient, l2Client, txHash, factory, portal)
		if err != nil {
			return err
		}
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

func txDataForOutputRootProof(ctx context.Context, proofClient *gethclient.Client, l2Client *ethclient.Client, txHash common.Hash, factory *opnode_bindings.DisputeGameFactoryCaller, portal *bindingspreview.OptimismPortal2) ([]byte, error) {
	params, err := withdrawals.ProveWithdrawalParametersFaultProofs(ctx, proofClient, l2Client, l2Client, txHash, factory, &portal.OptimismPortal2Caller)
	if err != nil {
		return nil, fmt.Errorf("could not create withdrawal proof parameters: %w", err)
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
		return nil, fmt.Errorf("failed to pack output root prove withdrawal transaction: %w", err)
	}
	return txData, nil
}

func txDataForSuperRootProof(ctx *cli.Context, l1EthClient apis.EthClient, proofClient *gethclient.Client, l2Client *ethclient.Client, txHash common.Hash, factory *opnode_bindings.DisputeGameFactoryCaller, portal *bindingspreview.OptimismPortal2) ([]byte, error) {
	supervisorClient, err := createSupervisorClient(ctx, SupervisorFlag.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create supervisor client: %w", err)
	}
	rollupCfg, err := loadRollupConfig(ctx, RollupConfigFlag.Name)
	if err != nil {
		return nil, err
	}
	depSet, err := loadDepsetConfig(ctx, DepSetFlag.Name)
	if err != nil {
		return nil, err
	}

	portalL2ChainID, err := l2ChainIDForPortal(ctx.Context, l1EthClient, portal)
	if err != nil {
		return nil, fmt.Errorf("failed to get portal chain ID: %w", err)
	}
	if portalL2ChainID != rollupCfg.L2ChainID.Uint64() {
		return nil, fmt.Errorf("portal chain ID %d does not match the provided rollup config chain ID %d", portalL2ChainID, rollupCfg.L2ChainID.Uint64())
	}

	params, err := withdrawals.ProveWithdrawalParametersSuperRoots(
		ctx.Context,
		rollupCfg,
		depSet,
		proofClient,
		l2Client,
		l2Client,
		txHash,
		supervisorClient,
		factory,
		&portal.OptimismPortal2Caller,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create withdrawal proof parameters: %w", err)
	}
	txData, err := w3.MustNewFunc("proveWithdrawalTransaction("+
		"(uint256 Nonce, address Sender, address Target, uint256 Value, uint256 GasLimit, bytes Data),"+
		"address DisputeGameProxy,"+
		"uint256 OutputRootIndex,"+
		"(bytes1 Version, uint64 Timestamp, (uint256 ChainID, bytes32 Root)[] OutputRoots),"+
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
		params.DisputeGameProxy,
		params.OutputRootIndex,
		params.SuperRootProof,
		params.OutputRootProof,
		params.WithdrawalProof,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack super root prove withdrawal transaction: %w", err)
	}
	return txData, nil
}

func l2ChainIDForPortal(ctx context.Context, l1EthClient apis.EthClient, portal *bindingspreview.OptimismPortal2) (uint64, error) {
	systemConfigAddr, err := portal.SystemConfig(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, fmt.Errorf("failed to get system config address from portal: %w", err)
	}
	systemConfig := bindings.NewSystemConfig(bindings.WithClient(l1EthClient), bindings.WithTo(systemConfigAddr))
	l2ChainID, err := contractio.Read(systemConfig.L2ChainID(), ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to read L2 chain ID from system config: %w", err)
	}
	return l2ChainID.Uint64(), nil
}

func proveFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		L1Flag,
		L2Flag,
		TxFlag,
		PortalAddressFlag,
		// Super Roots Flags
		SupervisorFlag,
		DepSetFlag,
		RollupConfigFlag,
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
