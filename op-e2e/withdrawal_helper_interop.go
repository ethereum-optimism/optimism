package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func SendWithdrawalInterop(t *testing.T, l2ChainID *big.Int, l2Client *ethclient.Client,
	privKey *ecdsa.PrivateKey, applyOpts WithdrawalTxOptsFn, l1BlockTime uint64,
	l2BlockTime uint64,) (*types.Transaction, *types.Receipt) {
	opts := defaultWithdrawalTxOpts()
	applyOpts(opts)

	// Bind L2 Withdrawer Contract
	l2withdrawer, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, l2Client)
	require.Nil(t, err, "binding withdrawer on L2")

	// Initiate Withdrawal
	l2opts, err := bind.NewKeyedTransactorWithChainID(privKey, l2ChainID)
	require.Nil(t, err)
	l2opts.Value = opts.Value
	tx, err := l2withdrawer.InitiateWithdrawal(l2opts, l2opts.From, big.NewInt(int64(opts.Gas)), opts.Data)
	require.Nil(t, err, "sending initiate withdraw tx")

	receipt, err := geth.WaitForTransaction(tx.Hash(), l2Client, 10*time.Duration(l1BlockTime)*time.Second)
	require.Nil(t, err, "withdrawal initiated on L2 sequencer")
	require.Equal(t, opts.ExpectedStatus, receipt.Status, "transaction had incorrect status")

	for i, client := range opts.VerifyClients {
		t.Logf("Waiting for tx %v on verification client %d", tx.Hash(), i)
		receiptVerif, err := geth.WaitForTransaction(tx.Hash(), client, 10*time.Duration(l2BlockTime)*time.Second)
		require.Nilf(t, err, "Waiting for L2 tx on verification client %d", i)
		require.Equalf(t, receipt, receiptVerif, "Receipts should be the same on sequencer and verification client %d", i)
	}
	return tx, receipt
}

func ProveAndFinalizeWithdrawalInterop(
	t *testing.T, l1BlockTime uint64, l1Client *ethclient.Client, l2Node EthInstance,
	ethPrivKey *ecdsa.PrivateKey, l2WithdrawalReceipt *types.Receipt, l1Deployments *genesis.L1Deployments, l1ChainID *big.Int,
) (*types.Receipt, *types.Receipt) {
	params, proveReceipt := ProveWithdrawalInterop(t, l1BlockTime, l1Client, l2Node, ethPrivKey, l2WithdrawalReceipt, l1Deployments, l1ChainID)
	finalizeReceipt := FinalizeWithdrawalInterop(t, l1BlockTime, l1Client, ethPrivKey, proveReceipt, params, l1ChainID, l1Deployments)
	return proveReceipt, finalizeReceipt
}

func ProveWithdrawalInterop(
	t *testing.T, l1BlockTime uint64, l1Client *ethclient.Client, l2Node EthInstance,
	ethPrivKey *ecdsa.PrivateKey, l2WithdrawalReceipt *types.Receipt, l1Deployments *genesis.L1Deployments,
	l1ChainID *big.Int,
) (withdrawals.ProvenWithdrawalParameters, *types.Receipt) {
	// Get l2BlockNumber for proof generation
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Duration(l1BlockTime)*time.Second)
	defer cancel()

	blockNumber, err := wait.ForOutputRootPublished(ctx, l1Client, l1Deployments.L2OutputOracleProxy, l2WithdrawalReceipt.BlockNumber)
	require.Nil(t, err)

	rpcClient, err := rpc.Dial(l2Node.WSEndpoint())
	require.Nil(t, err)
	proofCl := gethclient.New(rpcClient)
	receiptCl := ethclient.NewClient(rpcClient)

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Get the latest header
	header, err := receiptCl.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	require.Nil(t, err)

	// Now create withdrawal
	oracle, err := bindings.NewL2OutputOracleCaller(l1Deployments.L2OutputOracleProxy, l1Client)
	require.Nil(t, err)

	params, err := withdrawals.ProveWithdrawalParameters(context.Background(), proofCl, receiptCl, l2WithdrawalReceipt.TxHash, header, oracle)
	require.Nil(t, err)

	portal, err := bindings.NewOptimismPortal(l1Deployments.OptimismPortalProxy, l1Client)
	require.Nil(t, err)

	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, l1ChainID)
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
	proveReceipt, err := geth.WaitForTransaction(tx.Hash(), l1Client, 3*time.Duration(l1BlockTime)*time.Second)
	require.Nil(t, err, "prove withdrawal")
	require.Equal(t, types.ReceiptStatusSuccessful, proveReceipt.Status)
	return params, proveReceipt
}

func FinalizeWithdrawalInterop(
	t *testing.T, l1BlockTime uint64, l1Client *ethclient.Client, privKey *ecdsa.PrivateKey,
	withdrawalProofReceipt *types.Receipt, params withdrawals.ProvenWithdrawalParameters, l1ChainID *big.Int,
	l1Deployments *genesis.L1Deployments,
) *types.Receipt {
	// Wait for finalization and then create the Finalized Withdrawal Transaction
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Duration(l1BlockTime)*time.Second)
	defer cancel()

	err := wait.ForFinalizationPeriod(ctx, l1Client, withdrawalProofReceipt.BlockNumber, l1Deployments.L2OutputOracleProxy)
	require.Nil(t, err)

	opts, err := bind.NewKeyedTransactorWithChainID(privKey, l1ChainID)
	require.Nil(t, err)
	portal, err := bindings.NewOptimismPortal(l1Deployments.OptimismPortalProxy, l1Client)
	require.Nil(t, err)
	// Finalize withdrawal
	tx, err := portal.FinalizeWithdrawalTransaction(
		opts,
		bindings.TypesWithdrawalTransaction{
			Nonce:    params.Nonce,
			Sender:   params.Sender,
			Target:   params.Target,
			Value:    params.Value,
			GasLimit: params.GasLimit,
			Data:     params.Data,
		},
	)
	require.Nil(t, err)

	// Ensure that our withdrawal was finalized successfully
	finalizeReceipt, err := geth.WaitForTransaction(tx.Hash(), l1Client, 3*time.Duration(l1BlockTime)*time.Second)
	require.Nil(t, err, "finalize withdrawal")
	require.Equal(t, types.ReceiptStatusSuccessful, finalizeReceipt.Status)
	return finalizeReceipt
}
