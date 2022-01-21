package drivers_test

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/go/batch-submitter/drivers"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/mock"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/txmgr"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/utils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func init() {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	testPrivKey = privKey
	testWalletAddr = crypto.PubkeyToAddress(privKey.PublicKey)
	testChainID = new(big.Int).SetUint64(1)
	testGasPrice = new(big.Int).SetUint64(3)
}

var (
	testPrivKey     *ecdsa.PrivateKey
	testWalletAddr  common.Address
	testChainID     = big.NewInt(1)
	testNonce       = uint64(2)
	testGasPrice    = big.NewInt(3)
	testGasLimit    = uint64(4)
	testBlockNumber = uint64(5)
)

// TestCraftClearingTx asserts that CraftClearingTx produces the expected
// unsigned clearing transaction.
func TestCraftClearingTx(t *testing.T) {
	tx := drivers.CraftClearingTx(
		testWalletAddr, testNonce, testGasPrice, testGasLimit,
	)
	require.Equal(t, &testWalletAddr, tx.To())
	require.Equal(t, testNonce, tx.Nonce())
	require.Equal(t, testGasPrice, tx.GasPrice())
	require.Equal(t, testGasLimit, tx.Gas())
	require.Equal(t, new(big.Int), tx.Value())
	require.Nil(t, tx.Data())
}

// TestSignClearingTxSuccess asserts that we will sign a properly formed
// clearing transaction when the call to EstimateGas succeeds.
func TestSignClearingTxEstimateGasSuccess(t *testing.T) {
	l1Client := mock.NewL1Client(mock.L1ClientConfig{
		EstimateGas: func(_ context.Context, _ ethereum.CallMsg) (uint64, error) {
			return testGasLimit, nil
		},
	})

	tx, err := drivers.SignClearingTx(
		context.Background(), testWalletAddr, testNonce, testGasPrice, l1Client,
		testPrivKey, testChainID,
	)
	require.Nil(t, err)
	require.NotNil(t, tx)
	require.Equal(t, &testWalletAddr, tx.To())
	require.Equal(t, testNonce, tx.Nonce())
	require.Equal(t, testGasPrice, tx.GasPrice())
	require.Equal(t, testGasLimit, tx.Gas())
	require.Equal(t, new(big.Int), tx.Value())
	require.Nil(t, tx.Data())

	// Finally, ensure the sender is correct.
	sender, err := types.Sender(types.LatestSignerForChainID(testChainID), tx)
	require.Nil(t, err)
	require.Equal(t, testWalletAddr, sender)
}

// TestSignClearingTxEstimateGasFail asserts that signing a clearing transaction
// will fail if the underlying call to EstimateGas fails.
func TestSignClearingTxEstimateGasFail(t *testing.T) {
	errEstimateGas := errors.New("estimate gas")

	l1Client := mock.NewL1Client(mock.L1ClientConfig{
		EstimateGas: func(_ context.Context, _ ethereum.CallMsg) (uint64, error) {
			return 0, errEstimateGas
		},
	})

	tx, err := drivers.SignClearingTx(
		context.Background(), testWalletAddr, testNonce, testGasPrice, l1Client,
		testPrivKey, testChainID,
	)
	require.Equal(t, errEstimateGas, err)
	require.Nil(t, tx)
}

type clearPendingTxHarness struct {
	l1Client *mock.L1Client
	txMgr    txmgr.TxManager
}

func newClearPendingTxHarnessWithNumConfs(
	l1ClientConfig mock.L1ClientConfig,
	numConfirmations uint64,
) *clearPendingTxHarness {

	if l1ClientConfig.BlockNumber == nil {
		l1ClientConfig.BlockNumber = func(_ context.Context) (uint64, error) {
			return testBlockNumber, nil
		}
	}
	if l1ClientConfig.NonceAt == nil {
		l1ClientConfig.NonceAt = func(_ context.Context, _ common.Address, _ *big.Int) (uint64, error) {
			return testNonce, nil
		}
	}
	if l1ClientConfig.EstimateGas == nil {
		l1ClientConfig.EstimateGas = func(_ context.Context, _ ethereum.CallMsg) (uint64, error) {
			return testGasLimit, nil
		}
	}

	l1Client := mock.NewL1Client(l1ClientConfig)
	txMgr := txmgr.NewSimpleTxManager("test", txmgr.Config{
		MinGasPrice:          utils.GasPriceFromGwei(1),
		MaxGasPrice:          utils.GasPriceFromGwei(100),
		GasRetryIncrement:    utils.GasPriceFromGwei(5),
		ResubmissionTimeout:  time.Second,
		ReceiptQueryInterval: 50 * time.Millisecond,
		NumConfirmations:     numConfirmations,
	}, l1Client)

	return &clearPendingTxHarness{
		l1Client: l1Client,
		txMgr:    txMgr,
	}
}

func newClearPendingTxHarness(l1ClientConfig mock.L1ClientConfig) *clearPendingTxHarness {
	return newClearPendingTxHarnessWithNumConfs(l1ClientConfig, 1)
}

// TestClearPendingTxClearingTxÇonfirms asserts the happy path where our
// clearing transactions confirms unobstructed.
func TestClearPendingTxClearingTxConfirms(t *testing.T) {
	h := newClearPendingTxHarness(mock.L1ClientConfig{
		SendTransaction: func(_ context.Context, _ *types.Transaction) error {
			return nil
		},
		TransactionReceipt: func(_ context.Context, txHash common.Hash) (*types.Receipt, error) {
			return &types.Receipt{
				TxHash:      txHash,
				BlockNumber: big.NewInt(int64(testBlockNumber)),
			}, nil
		},
	})

	err := drivers.ClearPendingTx(
		"test", context.Background(), h.txMgr, h.l1Client, testWalletAddr,
		testPrivKey, testChainID,
	)
	require.Nil(t, err)
}

// TestClearPendingTx∏reviousTxConfirms asserts that if the mempool starts
// rejecting our transactions because the nonce is too low that ClearPendingTx
// will abort continuing to publish a clearing transaction.
func TestClearPendingTxPreviousTxConfirms(t *testing.T) {
	h := newClearPendingTxHarness(mock.L1ClientConfig{
		SendTransaction: func(_ context.Context, _ *types.Transaction) error {
			return core.ErrNonceTooLow
		},
	})

	err := drivers.ClearPendingTx(
		"test", context.Background(), h.txMgr, h.l1Client, testWalletAddr,
		testPrivKey, testChainID,
	)
	require.Equal(t, drivers.ErrClearPendingRetry, err)
}

// TestClearPendingTxTimeout asserts that ClearPendingTx returns an
// ErrPublishTimeout if the clearing transaction fails to confirm in a timely
// manner and no prior transaction confirms.
func TestClearPendingTxTimeout(t *testing.T) {
	h := newClearPendingTxHarness(mock.L1ClientConfig{
		SendTransaction: func(_ context.Context, _ *types.Transaction) error {
			return nil
		},
		TransactionReceipt: func(_ context.Context, txHash common.Hash) (*types.Receipt, error) {
			return nil, nil
		},
	})

	err := drivers.ClearPendingTx(
		"test", context.Background(), h.txMgr, h.l1Client, testWalletAddr,
		testPrivKey, testChainID,
	)
	require.Equal(t, txmgr.ErrPublishTimeout, err)
}

// TestClearPendingTxMultipleConfs tests we wait the appropriate number of
// confirmations for the clearing transaction to confirm.
func TestClearPendingTxMultipleConfs(t *testing.T) {
	const numConfs = 2

	// Instantly confirm transaction.
	h := newClearPendingTxHarnessWithNumConfs(mock.L1ClientConfig{
		SendTransaction: func(_ context.Context, _ *types.Transaction) error {
			return nil
		},
		TransactionReceipt: func(_ context.Context, txHash common.Hash) (*types.Receipt, error) {
			return &types.Receipt{
				TxHash:      txHash,
				BlockNumber: big.NewInt(int64(testBlockNumber)),
			}, nil
		},
	}, numConfs)

	// The txmgr should timeout waiting for the txn to confirm.
	err := drivers.ClearPendingTx(
		"test", context.Background(), h.txMgr, h.l1Client, testWalletAddr,
		testPrivKey, testChainID,
	)
	require.Equal(t, txmgr.ErrPublishTimeout, err)

	// Now set the chain height to the earliest the transaction will be
	// considered sufficiently confirmed.
	h.l1Client.SetBlockNumberFunc(func(_ context.Context) (uint64, error) {
		return testBlockNumber + numConfs - 1, nil
	})

	// Publishing should succeed.
	err = drivers.ClearPendingTx(
		"test", context.Background(), h.txMgr, h.l1Client, testWalletAddr,
		testPrivKey, testChainID,
	)
	require.Nil(t, err)
}
