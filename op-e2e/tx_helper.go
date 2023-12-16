package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// SendDepositTx creates and sends a deposit transaction.
// The L1 transaction, including sender, is configured by the l1Opts param.
// The L2 transaction options can be configured by modifying the DepositTxOps value supplied to applyL2Opts
// Will verify that the transaction is included with the expected status on L1 and L2
// Returns the receipt of the L2 transaction
func SendDepositTx(t *testing.T, cfg SystemConfig, l1Client *ethclient.Client, l2Client *ethclient.Client, l1Opts *bind.TransactOpts, applyL2Opts DepositTxOptsFn) *types.Receipt {
	l2Opts := defaultDepositTxOpts(l1Opts)
	applyL2Opts(l2Opts)

	// Find deposit contract
	depositContract, err := bindings.NewOptimismPortal(cfg.L1Deployments.OptimismPortalProxy, l1Client)
	require.Nil(t, err)

	// Finally send TX
	// Add 10% padding for the L1 gas limit because the estimation process can be affected by the 1559 style cost scale
	// for buying L2 gas in the portal contracts.
	tx, err := transactions.PadGasEstimate(l1Opts, 1.1, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return depositContract.DepositTransaction(opts, l2Opts.ToAddr, l2Opts.Value, l2Opts.GasLimit, l2Opts.IsCreation, l2Opts.Data)
	})
	require.Nil(t, err, "with deposit tx")

	// Wait for transaction on L1
	l1Receipt, err := geth.WaitForTransaction(tx.Hash(), l1Client, 10*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	// Wait for transaction to be included on L2
	reconstructedDep, err := derive.UnmarshalDepositLogEvent(l1Receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	// Use a long wait because the l2Client may not be configured to receive gossip from the sequencer
	// so has to wait for the batcher to submit and then import those blocks from L1.
	l2Receipt, err := geth.WaitForTransaction(tx.Hash(), l2Client, 60*time.Second)
	require.NoError(t, err)
	require.Equal(t, l2Opts.ExpectedStatus, l2Receipt.Status, "l2 transaction status")
	return l2Receipt
}

type DepositTxOptsFn func(l2Opts *DepositTxOpts)

type DepositTxOpts struct {
	ToAddr         common.Address
	Value          *big.Int
	GasLimit       uint64
	IsCreation     bool
	Data           []byte
	ExpectedStatus uint64
}

func defaultDepositTxOpts(opts *bind.TransactOpts) *DepositTxOpts {
	return &DepositTxOpts{
		ToAddr:         opts.From,
		Value:          opts.Value,
		GasLimit:       1_000_000,
		IsCreation:     false,
		Data:           nil,
		ExpectedStatus: types.ReceiptStatusSuccessful,
	}
}

// SendL2Tx creates and sends a transaction.
// The supplied privKey is used to specify the account to send from and the transaction is sent to the supplied l2Client
// Transaction options and expected status can be configured in the applyTxOpts function by modifying the supplied TxOpts
// Will verify that the transaction is included with the expected status on l2Client and any clients added to TxOpts.VerifyClients
func SendL2Tx(t *testing.T, cfg SystemConfig, l2Client *ethclient.Client, privKey *ecdsa.PrivateKey, applyTxOpts TxOptsFn) *types.Receipt {
	opts := defaultTxOpts()
	applyTxOpts(opts)
	tx := types.MustSignNewTx(privKey, types.LatestSignerForChainID(cfg.L2ChainIDBig()), &types.DynamicFeeTx{
		ChainID:   cfg.L2ChainIDBig(),
		Nonce:     opts.Nonce, // Already have deposit
		To:        opts.ToAddr,
		Value:     opts.Value,
		GasTipCap: opts.GasTipCap,
		GasFeeCap: opts.GasFeeCap,
		Gas:       opts.Gas,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := l2Client.SendTransaction(ctx, tx)
	require.NoError(t, err, "Sending L2 tx")

	receipt, err := geth.WaitForTransaction(tx.Hash(), l2Client, 10*time.Duration(cfg.DeployConfig.L2BlockTime)*time.Second)
	require.NoError(t, err, "Waiting for L2 tx")
	require.Equal(t, opts.ExpectedStatus, receipt.Status, "TX should have expected status")

	for i, client := range opts.VerifyClients {
		t.Logf("Waiting for tx %v on verification client %d", tx.Hash(), i)
		receiptVerif, err := geth.WaitForTransaction(tx.Hash(), client, 10*time.Duration(cfg.DeployConfig.L2BlockTime)*time.Second)
		require.NoErrorf(t, err, "Waiting for L2 tx on verification client %d", i)
		require.Equalf(t, receipt, receiptVerif, "Receipts should be the same on sequencer and verification client %d", i)
	}
	return receipt
}

type TxOptsFn func(opts *TxOpts)

type TxOpts struct {
	ToAddr         *common.Address
	Nonce          uint64
	Value          *big.Int
	Gas            uint64
	GasTipCap      *big.Int
	GasFeeCap      *big.Int
	Data           []byte
	ExpectedStatus uint64
	VerifyClients  []*ethclient.Client
}

// VerifyOnClients adds additional l2 clients that should sync the block the tx is included in
// Checks that the receipt received from these clients is equal to the receipt received from the sequencer
func (o *TxOpts) VerifyOnClients(clients ...*ethclient.Client) {
	o.VerifyClients = append(o.VerifyClients, clients...)
}

func defaultTxOpts() *TxOpts {
	return &TxOpts{
		ToAddr:         nil,
		Nonce:          0,
		Value:          common.Big0,
		GasTipCap:      big.NewInt(10),
		GasFeeCap:      big.NewInt(200),
		Gas:            21_000,
		Data:           nil,
		ExpectedStatus: types.ReceiptStatusSuccessful,
	}
}
