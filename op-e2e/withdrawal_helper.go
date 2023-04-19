package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

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

	receipt, err := waitForTransaction(tx.Hash(), l2Client, 10*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "withdrawal initiated on L2 sequencer")
	require.Equal(t, opts.ExpectedStatus, receipt.Status, "transaction had incorrect status")

	for i, client := range opts.VerifyClients {
		t.Logf("Waiting for tx %v on verification client %d", tx.Hash(), i)
		receiptVerif, err := waitForTransaction(tx.Hash(), client, 10*time.Duration(cfg.DeployConfig.L2BlockTime)*time.Second)
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

func ProveAndFinalizeWithdrawal(t *testing.T, cfg SystemConfig, l1Client *ethclient.Client, l2Node *node.Node, ethPrivKey *ecdsa.PrivateKey, l2WithdrawalReceipt *types.Receipt) (*types.Receipt, *types.Receipt) {
	params, proveReceipt := ProveWithdrawal(t, cfg, l1Client, l2Node, ethPrivKey, l2WithdrawalReceipt)
	finalizeReceipt := FinalizeWithdrawal(t, cfg, l1Client, ethPrivKey, l2WithdrawalReceipt, params)
	return proveReceipt, finalizeReceipt
}

func ProveWithdrawal(t *testing.T, cfg SystemConfig, l1Client *ethclient.Client, l2Node *node.Node, ethPrivKey *ecdsa.PrivateKey, l2WithdrawalReceipt *types.Receipt) (withdrawals.ProvenWithdrawalParameters, *types.Receipt) {
	// Get l2BlockNumber for proof generation
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	defer cancel()
	blockNumber, err := withdrawals.WaitForFinalizationPeriod(ctx, l1Client, predeploys.DevOptimismPortalAddr, l2WithdrawalReceipt.BlockNumber)
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
	oracle, err := bindings.NewL2OutputOracleCaller(predeploys.DevL2OutputOracleAddr, l1Client)
	require.Nil(t, err)

	params, err := withdrawals.ProveWithdrawalParameters(context.Background(), proofCl, receiptCl, l2WithdrawalReceipt.TxHash, header, oracle)
	require.Nil(t, err)

	portal, err := bindings.NewOptimismPortal(predeploys.DevOptimismPortalAddr, l1Client)
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
	proveReceipt, err := waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "prove withdrawal")
	require.Equal(t, types.ReceiptStatusSuccessful, proveReceipt.Status)
	return params, proveReceipt
}

func FinalizeWithdrawal(t *testing.T, cfg SystemConfig, l1Client *ethclient.Client, privKey *ecdsa.PrivateKey, withdrawalReceipt *types.Receipt, params withdrawals.ProvenWithdrawalParameters) *types.Receipt {
	// Wait for finalization and then create the Finalized Withdrawal Transaction
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	defer cancel()
	_, err := withdrawals.WaitForFinalizationPeriod(ctx, l1Client, predeploys.DevOptimismPortalAddr, withdrawalReceipt.BlockNumber)
	require.Nil(t, err)

	opts, err := bind.NewKeyedTransactorWithChainID(privKey, cfg.L1ChainIDBig())
	require.Nil(t, err)
	portal, err := bindings.NewOptimismPortal(predeploys.DevOptimismPortalAddr, l1Client)
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
	finalizeReceipt, err := waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "finalize withdrawal")
	require.Equal(t, types.ReceiptStatusSuccessful, finalizeReceipt.Status)
	return finalizeReceipt
}
