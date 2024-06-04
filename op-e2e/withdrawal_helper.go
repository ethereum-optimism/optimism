package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	legacybindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/require"
)

type ClientProvider interface {
	NodeClient(name string) *ethclient.Client
}

func SendWithdrawal(t *testing.T, cfg SystemConfig, l2Client *ethclient.Client, privKey *ecdsa.PrivateKey, applyOpts WithdrawalTxOptsFn) (*types.Transaction, *types.Receipt) {
	opts := defaultWithdrawalTxOpts()
	applyOpts(opts)

	// Bind L2 Withdrawer Contract
	l2withdrawer, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, l2Client)
	require.Nil(t, err, "binding withdrawer on L2")

	// Initiate Withdrawal
	l2opts, err := bind.NewKeyedTransactorWithChainID(privKey, cfg.L2ChainIDBig())
	require.Nil(t, err)
	l2opts.Value = opts.Value
	tx, err := l2withdrawer.InitiateWithdrawal(l2opts, l2opts.From, big.NewInt(int64(opts.Gas)), opts.Data)
	require.Nil(t, err, "sending initiate withdraw tx")

	receipt, err := geth.WaitForTransaction(tx.Hash(), l2Client, 10*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "withdrawal initiated on L2 sequencer")
	require.Equal(t, opts.ExpectedStatus, receipt.Status, "transaction had incorrect status")

	for i, client := range opts.VerifyClients {
		t.Logf("Waiting for tx %v on verification client %d", tx.Hash(), i)
		receiptVerif, err := geth.WaitForTransaction(tx.Hash(), client, 10*time.Duration(cfg.DeployConfig.L2BlockTime)*time.Second)
		require.Nilf(t, err, "Waiting for L2 tx on verification client %d", i)
		require.Equalf(t, receipt, receiptVerif, "Receipts should be the same on sequencer and verification client %d", i)
	}
	return tx, receipt
}

type WithdrawalTxOptsFn func(opts *WithdrawalTxOpts)

type WithdrawalTxOpts struct {
	ToAddr         *common.Address
	Nonce          uint64
	Value          *big.Int
	Gas            uint64
	Data           []byte
	ExpectedStatus uint64
	VerifyClients  []*ethclient.Client
}

// VerifyOnClients adds additional l2 clients that should sync the block the tx is included in
// Checks that the receipt received from these clients is equal to the receipt received from the sequencer
func (o *WithdrawalTxOpts) VerifyOnClients(clients ...*ethclient.Client) {
	o.VerifyClients = append(o.VerifyClients, clients...)
}

func defaultWithdrawalTxOpts() *WithdrawalTxOpts {
	return &WithdrawalTxOpts{
		ToAddr:         nil,
		Nonce:          0,
		Value:          common.Big0,
		Gas:            21_000,
		Data:           nil,
		ExpectedStatus: types.ReceiptStatusSuccessful,
	}
}

func ProveAndFinalizeWithdrawal(t *testing.T, cfg SystemConfig, clients ClientProvider, l2NodeName string, ethPrivKey *ecdsa.PrivateKey, l2WithdrawalReceipt *types.Receipt) (*types.Receipt, *types.Receipt, *types.Receipt, *types.Receipt) {
	params, proveReceipt := ProveWithdrawal(t, cfg, clients, l2NodeName, ethPrivKey, l2WithdrawalReceipt)
	finalizeReceipt, resolveClaimReceipt, resolveReceipt := FinalizeWithdrawal(t, cfg, clients.NodeClient("l1"), ethPrivKey, proveReceipt, params)
	return proveReceipt, finalizeReceipt, resolveClaimReceipt, resolveReceipt
}

func ProveWithdrawal(t *testing.T, cfg SystemConfig, clients ClientProvider, l2NodeName string, ethPrivKey *ecdsa.PrivateKey, l2WithdrawalReceipt *types.Receipt) (withdrawals.ProvenWithdrawalParameters, *types.Receipt) {
	// Get l2BlockNumber for proof generation
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	defer cancel()

	l1Client := clients.NodeClient("l1")
	var blockNumber uint64
	var err error
	if e2eutils.UseFaultProofs() {
		blockNumber, err = wait.ForGamePublished(ctx, l1Client, config.L1Deployments.OptimismPortalProxy, config.L1Deployments.DisputeGameFactoryProxy, l2WithdrawalReceipt.BlockNumber)
		require.Nil(t, err)
	} else {
		blockNumber, err = wait.ForOutputRootPublished(ctx, l1Client, config.L1Deployments.L2OutputOracleProxy, l2WithdrawalReceipt.BlockNumber)
		require.Nil(t, err)
	}

	receiptCl := clients.NodeClient(l2NodeName)
	blockCl := clients.NodeClient(l2NodeName)
	proofCl := gethclient.New(receiptCl.Client())

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Get the latest header
	header, err := receiptCl.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	require.Nil(t, err)

	oracle, err := bindings.NewL2OutputOracleCaller(config.L1Deployments.L2OutputOracleProxy, l1Client)
	require.Nil(t, err)

	factory, err := bindings.NewDisputeGameFactoryCaller(config.L1Deployments.DisputeGameFactoryProxy, l1Client)
	require.Nil(t, err)

	portal2, err := bindingspreview.NewOptimismPortal2Caller(config.L1Deployments.OptimismPortalProxy, l1Client)
	require.Nil(t, err)

	params, err := ProveWithdrawalParameters(context.Background(), proofCl, receiptCl, blockCl, l2WithdrawalReceipt.TxHash, header, oracle, factory, portal2)
	require.Nil(t, err)

	portal, err := bindings.NewOptimismPortal(config.L1Deployments.OptimismPortalProxy, l1Client)
	require.Nil(t, err)

	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainIDBig())
	require.Nil(t, err)

	// Prove withdrawal
	tx, err := portal.ProveWithdrawalTransaction(
		opts,
		bindings.TypesWithdrawalTransaction{
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
	require.Nil(t, err)

	// Ensure that our withdrawal was proved successfully
	proveReceipt, err := geth.WaitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "prove withdrawal")
	require.Equal(t, types.ReceiptStatusSuccessful, proveReceipt.Status)
	return params, proveReceipt
}

func ProveWithdrawalParameters(ctx context.Context, proofCl withdrawals.ProofClient, l2ReceiptCl withdrawals.ReceiptClient, l2BlockCl withdrawals.BlockClient, txHash common.Hash, header *types.Header, l2OutputOracleContract *bindings.L2OutputOracleCaller, disputeGameFactoryContract *bindings.DisputeGameFactoryCaller, optimismPortal2Contract *bindingspreview.OptimismPortal2Caller) (withdrawals.ProvenWithdrawalParameters, error) {
	if e2eutils.UseFaultProofs() {
		return withdrawals.ProveWithdrawalParametersFaultProofs(ctx, proofCl, l2ReceiptCl, l2BlockCl, txHash, disputeGameFactoryContract, optimismPortal2Contract)
	} else {
		return withdrawals.ProveWithdrawalParameters(ctx, proofCl, l2ReceiptCl, l2BlockCl, txHash, header, l2OutputOracleContract)
	}
}

func FinalizeWithdrawal(t *testing.T, cfg SystemConfig, l1Client *ethclient.Client, privKey *ecdsa.PrivateKey, withdrawalProofReceipt *types.Receipt, params withdrawals.ProvenWithdrawalParameters) (*types.Receipt, *types.Receipt, *types.Receipt) {
	// Wait for finalization and then create the Finalized Withdrawal Transaction
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	defer cancel()

	wd := crossdomain.Withdrawal{
		Nonce:    params.Nonce,
		Sender:   &params.Sender,
		Target:   &params.Target,
		Value:    params.Value,
		GasLimit: params.GasLimit,
		Data:     params.Data,
	}

	opts, err := bind.NewKeyedTransactorWithChainID(privKey, cfg.L1ChainIDBig())
	require.Nil(t, err)

	var resolveClaimReceipt *types.Receipt
	var resolveReceipt *types.Receipt
	if e2eutils.UseFaultProofs() {
		portal2, err := bindingspreview.NewOptimismPortal2(config.L1Deployments.OptimismPortalProxy, l1Client)
		require.Nil(t, err)

		wdHash, err := wd.Hash()
		require.Nil(t, err)

		game, err := portal2.ProvenWithdrawals(&bind.CallOpts{}, wdHash, opts.From)
		require.Nil(t, err)
		require.NotNil(t, game, "withdrawal should be proven")

		proxy, err := legacybindings.NewFaultDisputeGame(game.DisputeGameProxy, l1Client)
		require.Nil(t, err)

		caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
		gameContract, err := contracts.NewFaultDisputeGameContract(context.Background(), metrics.NoopContractMetrics, game.DisputeGameProxy, caller)
		require.Nil(t, err)

		timedCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		require.NoError(t, wait.For(timedCtx, time.Second, func() (bool, error) {
			err := gameContract.CallResolveClaim(context.Background(), 0)
			t.Logf("Could not resolve dispute game claim: %v", err)
			return err == nil, nil
		}))

		resolveClaimTx, err := proxy.ResolveClaim(opts, common.Big0, common.Big0)
		require.Nil(t, err)

		resolveClaimReceipt, err = wait.ForReceiptOK(ctx, l1Client, resolveClaimTx.Hash())
		require.Nil(t, err, "resolve claim")
		require.Equal(t, types.ReceiptStatusSuccessful, resolveClaimReceipt.Status)

		resolveTx, err := proxy.Resolve(opts)
		require.Nil(t, err)

		resolveReceipt, err = wait.ForReceiptOK(ctx, l1Client, resolveTx.Hash())
		require.Nil(t, err, "resolve")
		require.Equal(t, types.ReceiptStatusSuccessful, resolveReceipt.Status)
	}

	if e2eutils.UseFaultProofs() {
		err := wait.ForWithdrawalCheck(ctx, l1Client, wd, config.L1Deployments.OptimismPortalProxy, opts.From)
		require.Nil(t, err)
	} else {
		err := wait.ForFinalizationPeriod(ctx, l1Client, withdrawalProofReceipt.BlockNumber, config.L1Deployments.L2OutputOracleProxy)
		require.Nil(t, err)
	}

	portal, err := bindings.NewOptimismPortal(config.L1Deployments.OptimismPortalProxy, l1Client)
	require.Nil(t, err)

	// Finalize withdrawal
	tx, err := portal.FinalizeWithdrawalTransaction(opts, wd.WithdrawalTransaction())
	require.Nil(t, err)

	// Ensure that our withdrawal was finalized successfully
	finalizeReceipt, err := wait.ForReceiptOK(ctx, l1Client, tx.Hash())
	require.Nil(t, err, "finalize withdrawal")
	require.Equal(t, types.ReceiptStatusSuccessful, finalizeReceipt.Status)
	return finalizeReceipt, resolveClaimReceipt, resolveReceipt
}
