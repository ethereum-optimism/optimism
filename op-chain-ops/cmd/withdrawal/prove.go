package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	opnode_bindings "github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum"
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
	RollupConfigFlag = &cli.StringFlag{
		Name:    "rollup.config",
		Usage:   "Path to the rollup config of the target chain. Only required for proving using super roots.",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "ROLLUP_CONFIG"),
	}
	DisputeGameFlag = &cli.StringFlag{
		Name:    "dispute-game",
		Usage:   "Address of SuperFaultDisputeGame. Required when proving super root withdrawals. Reads super root proof from on-chain extraData.",
		EnvVars: op_service.PrefixEnvVar(EnvVarPrefix, "DISPUTE_GAME"),
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
		disputeGameStr := ctx.String(DisputeGameFlag.Name)
		if disputeGameStr == "" {
			return errors.New("--dispute-game is required when proving super root withdrawals")
		}

		logger.Info("Proving withdrawal using super root from dispute game extraData")
		disputeGameAddr := common.HexToAddress(disputeGameStr)

		rollupCfg, err := loadRollupConfig(ctx, RollupConfigFlag.Name)
		if err != nil {
			return fmt.Errorf("failed to load rollup config: %w", err)
		}

		txData, err = txDataForSuperRootProofFromGame(
			ctx.Context,
			l1Client,
			l1EthClient,
			proofClient,
			l2Client,
			txHash,
			disputeGameAddr,
			portalAddr,
			portal,
			rollupCfg,
		)
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

// txDataForSuperRootProofFromGame builds withdrawal proof tx data by reading the super root proof
// directly from a SuperFaultDisputeGame's extraData, without requiring supervisor-rpc or depset.
func txDataForSuperRootProofFromGame(
	ctx context.Context,
	l1Client *ethclient.Client,
	l1EthClient apis.EthClient,
	proofClient *gethclient.Client,
	l2Client *ethclient.Client,
	txHash common.Hash,
	disputeGameAddr common.Address,
	portalAddr common.Address,
	portal *bindingspreview.OptimismPortal2,
	rollupCfg *rollup.Config,
) ([]byte, error) {
	// Load ABI and prepare contract calls
	gameABI := snapshots.LoadSuperFaultDisputeGameABI()

	// Call extraData() to get encoded super root proof
	extraDataCallData, err := gameABI.Pack("extraData")
	if err != nil {
		return nil, fmt.Errorf("failed to pack extraData call: %w", err)
	}
	extraDataResult, err := l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &disputeGameAddr,
		Data: extraDataCallData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call extraData: %w", err)
	}

	// Unpack extraData result (returns bytes)
	unpackedExtra, err := gameABI.Unpack("extraData", extraDataResult)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack extraData: %w", err)
	}
	if len(unpackedExtra) == 0 {
		return nil, errors.New("extraData returned empty result")
	}
	extraDataBytes, ok := unpackedExtra[0].([]byte)
	if !ok {
		return nil, errors.New("extraData result is not []byte")
	}

	// Decode super root proof from extraData
	superRootProof, err := withdrawals.DecodeSuperRootProof(extraDataBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode super root proof: %w", err)
	}

	// Get target L2 chain ID from portal
	targetChainID, err := l2ChainIDForPortal(ctx, l1EthClient, portal)
	if err != nil {
		return nil, fmt.Errorf("failed to get target chain ID from portal: %w", err)
	}

	// Find output root index for target chain in the super root proof
	var outputRootIndex *big.Int
	for i, outputRoot := range superRootProof.OutputRoots {
		if outputRoot.ChainID.Uint64() == targetChainID {
			outputRootIndex = big.NewInt(int64(i))
			break
		}
	}
	if outputRootIndex == nil {
		return nil, fmt.Errorf("target chain ID %d not found in super root proof", targetChainID)
	}

	// Get L2 sequence number (timestamp) from the dispute game
	seqNumCallData, err := gameABI.Pack("l2SequenceNumber")
	if err != nil {
		return nil, fmt.Errorf("failed to pack l2SequenceNumber call: %w", err)
	}
	seqNumResult, err := l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &disputeGameAddr,
		Data: seqNumCallData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call l2SequenceNumber: %w", err)
	}
	unpackedSeq, err := gameABI.Unpack("l2SequenceNumber", seqNumResult)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack l2SequenceNumber: %w", err)
	}
	if len(unpackedSeq) == 0 {
		return nil, errors.New("l2SequenceNumber returned empty result")
	}
	l2SequenceNumber, ok := unpackedSeq[0].(*big.Int)
	if !ok {
		return nil, errors.New("l2SequenceNumber result is not *big.Int")
	}

	// Convert sequence number (timestamp) to L2 block number using rollup config
	l2BlockNumber, err := rollupCfg.TargetBlockNumber(l2SequenceNumber.Uint64())
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 block number from sequence number: %w", err)
	}

	// Fetch the L2 header at that block
	l2Header, err := l2Client.HeaderByNumber(ctx, new(big.Int).SetUint64(l2BlockNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 header: %w", err)
	}

	// Get withdrawal receipt and parse MessagePassed event
	receipt, err := l2Client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawal receipt: %w", err)
	}
	ev, err := withdrawals.ParseMessagePassed(receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse withdrawal event: %w", err)
	}

	// Build the withdrawal storage proof
	withdrawalProof, storageRoot, err := withdrawals.GetWithdrawalProof(ctx, proofClient, ev, l2Header)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawal proof: %w", err)
	}

	// Pack the proveWithdrawalTransaction call data
	txData, err := w3.MustNewFunc("proveWithdrawalTransaction("+
		"(uint256 Nonce, address Sender, address Target, uint256 Value, uint256 GasLimit, bytes Data),"+
		"address DisputeGameProxy,"+
		"uint256 OutputRootIndex,"+
		"(bytes1 Version, uint64 Timestamp, (uint256 ChainID, bytes32 Root)[] OutputRoots),"+
		"(bytes32 Version, bytes32 StateRoot, bytes32 MessagePasserStorageRoot, bytes32 LatestBlockhash),"+
		"bytes[])", "").EncodeArgs(
		bindingspreview.TypesWithdrawalTransaction{
			Nonce:    ev.Nonce,
			Sender:   ev.Sender,
			Target:   ev.Target,
			Value:    ev.Value,
			GasLimit: ev.GasLimit,
			Data:     ev.Data,
		},
		disputeGameAddr,
		outputRootIndex,
		superRootProof,
		opnode_bindings.TypesOutputRootProof{
			Version:                  [32]byte{},
			StateRoot:                l2Header.Root,
			MessagePasserStorageRoot: storageRoot,
			LatestBlockhash:          l2Header.Hash(),
		},
		withdrawalProof,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack prove withdrawal transaction: %w", err)
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
		RollupConfigFlag,
		DisputeGameFlag,
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
