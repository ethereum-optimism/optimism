package txmgr_test

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/go/batch-submitter/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestNextGasPrice asserts that NextGasPrice properly bumps the passed current
// gas price, and clamps it to the max gas price. It also tests that
// NextGasPrice doesn't mutate the passed curGasPrice argument.
func TestNextGasPrice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		curGasPrice       *big.Int
		gasRetryIncrement *big.Int
		maxGasPrice       *big.Int
		expGasPrice       *big.Int
	}{
		{
			name:              "increment below max",
			curGasPrice:       new(big.Int).SetUint64(5),
			gasRetryIncrement: new(big.Int).SetUint64(10),
			maxGasPrice:       new(big.Int).SetUint64(20),
			expGasPrice:       new(big.Int).SetUint64(15),
		},
		{
			name:              "increment equal max",
			curGasPrice:       new(big.Int).SetUint64(5),
			gasRetryIncrement: new(big.Int).SetUint64(10),
			maxGasPrice:       new(big.Int).SetUint64(15),
			expGasPrice:       new(big.Int).SetUint64(15),
		},
		{
			name:              "increment above max",
			curGasPrice:       new(big.Int).SetUint64(5),
			gasRetryIncrement: new(big.Int).SetUint64(10),
			maxGasPrice:       new(big.Int).SetUint64(12),
			expGasPrice:       new(big.Int).SetUint64(12),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Copy curGasPrice, as we will later test for mutation.
			curGasPrice := new(big.Int).Set(test.curGasPrice)

			nextGasPrice := txmgr.NextGasPrice(
				curGasPrice, test.gasRetryIncrement,
				test.maxGasPrice,
			)

			require.Equal(t, nextGasPrice, test.expGasPrice)

			// Ensure curGasPrice hasn't been mutated. This check
			// enforces that NextGasPrice creates a copy internally.
			// Failure to do so could result in gas price bumps
			// being read concurrently from other goroutines, and
			// introduce race conditions.
			require.Equal(t, curGasPrice, test.curGasPrice)
		})
	}
}

// testHarness houses the necessary resources to test the SimpleTxManager.
type testHarness struct {
	cfg     txmgr.Config
	mgr     txmgr.TxManager
	backend *mockBackend
}

// newTestHarnessWithConfig initializes a testHarness with a specific
// configuration.
func newTestHarnessWithConfig(cfg txmgr.Config) *testHarness {
	backend := newMockBackend()
	mgr := txmgr.NewSimpleTxManager("TEST", cfg, backend)

	return &testHarness{
		cfg:     cfg,
		mgr:     mgr,
		backend: backend,
	}
}

// newTestHarness initializes a testHarness with a defualt configuration that is
// suitable for most tests.
func newTestHarness() *testHarness {
	return newTestHarnessWithConfig(configWithNumConfs(1))
}

func configWithNumConfs(numConfirmations uint64) txmgr.Config {
	return txmgr.Config{
		MinGasPrice:          new(big.Int).SetUint64(5),
		MaxGasPrice:          new(big.Int).SetUint64(50),
		GasRetryIncrement:    new(big.Int).SetUint64(5),
		ResubmissionTimeout:  time.Second,
		ReceiptQueryInterval: 50 * time.Millisecond,
		NumConfirmations:     numConfirmations,
	}
}

type minedTxInfo struct {
	gasPrice    *big.Int
	blockNumber uint64
}

// mockBackend implements txmgr.ReceiptSource that tracks mined transactions
// along with the gas price used.
type mockBackend struct {
	mu sync.RWMutex

	// blockHeight tracks the current height of the chain.
	blockHeight uint64

	// minedTxs maps the hash of a mined transaction to its details.
	minedTxs map[common.Hash]minedTxInfo
}

// newMockBackend initializes a new mockBackend.
func newMockBackend() *mockBackend {
	return &mockBackend{
		minedTxs: make(map[common.Hash]minedTxInfo),
	}
}

// mine records a (txHash, gasPrice) as confirmed. Subsequent calls to
// TransactionReceipt with a matching txHash will result in a non-nil receipt.
// If a nil txHash is supplied this has the effect of mining an empty block.
func (b *mockBackend) mine(txHash *common.Hash, gasPrice *big.Int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.blockHeight++
	if txHash != nil {
		b.minedTxs[*txHash] = minedTxInfo{
			gasPrice:    gasPrice,
			blockNumber: b.blockHeight,
		}
	}
}

// BlockNumber returns the most recent block number.
func (b *mockBackend) BlockNumber(ctx context.Context) (uint64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.blockHeight, nil
}

// TransactionReceipt queries the mockBackend for a mined txHash. If none is
// found, nil is returned for both return values. Otherwise, it retruns a
// receipt containing the txHash and the gasPrice used in the GasUsed to make
// the value accessible from our test framework.
func (b *mockBackend) TransactionReceipt(
	ctx context.Context,
	txHash common.Hash,
) (*types.Receipt, error) {

	b.mu.RLock()
	defer b.mu.RUnlock()

	txInfo, ok := b.minedTxs[txHash]
	if !ok {
		return nil, nil
	}

	// Return the gas price for the transaction in the GasUsed field so that
	// we can assert the proper tx confirmed in our tests.
	return &types.Receipt{
		TxHash:      txHash,
		GasUsed:     txInfo.gasPrice.Uint64(),
		BlockNumber: big.NewInt(int64(txInfo.blockNumber)),
	}, nil
}

// TestTxMgrConfirmAtMinGasPrice asserts that Send returns the min gas price tx
// if the tx is mined instantly.
func TestTxMgrConfirmAtMinGasPrice(t *testing.T) {
	t.Parallel()

	h := newTestHarness()
	sendTxFunc := func(
		ctx context.Context,
		gasPrice *big.Int,
	) (*types.Transaction, error) {
		tx := types.NewTx(&types.LegacyTx{
			GasPrice: gasPrice,
		})
		txHash := tx.Hash()
		h.backend.mine(&txHash, gasPrice)
		return tx, nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, sendTxFunc)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.GasUsed, h.cfg.MinGasPrice.Uint64())
}

// TestTxMgrNeverConfirmCancel asserts that a Send can be canceled even if no
// transaction is mined. This is done to ensure the the tx mgr can properly
// abort on shutdown, even if a txn is in the process of being published.
func TestTxMgrNeverConfirmCancel(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	sendTxFunc := func(
		ctx context.Context,
		gasPrice *big.Int,
	) (*types.Transaction, error) {
		// Don't publish tx to backend, simulating never being mined.
		return types.NewTx(&types.LegacyTx{
			GasPrice: gasPrice,
		}), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	receipt, err := h.mgr.Send(ctx, sendTxFunc)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// TestTxMgrConfirmsAtMaxGasPrice asserts that Send properly returns the max gas
// price receipt if none of the lower gas price txs were mined.
func TestTxMgrConfirmsAtMaxGasPrice(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	sendTxFunc := func(
		ctx context.Context,
		gasPrice *big.Int,
	) (*types.Transaction, error) {
		tx := types.NewTx(&types.LegacyTx{
			GasPrice: gasPrice,
		})
		if gasPrice.Cmp(h.cfg.MaxGasPrice) == 0 {
			txHash := tx.Hash()
			h.backend.mine(&txHash, gasPrice)
		}
		return tx, nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, sendTxFunc)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.GasUsed, h.cfg.MaxGasPrice.Uint64())
}

// TestTxMgrConfirmsAtMaxGasPriceDelayed asserts that after the maximum gas
// price tx has been published, and a resubmission timeout has elapsed, that an
// error is returned signaling that even our max gas price is taking too long.
func TestTxMgrConfirmsAtMaxGasPriceDelayed(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	sendTxFunc := func(
		ctx context.Context,
		gasPrice *big.Int,
	) (*types.Transaction, error) {
		tx := types.NewTx(&types.LegacyTx{
			GasPrice: gasPrice,
		})
		// Delay mining of the max gas price tx by more than the
		// resubmission timeout. Default config uses 1 second. Send
		// should still return an error beforehand.
		if gasPrice.Cmp(h.cfg.MaxGasPrice) == 0 {
			time.AfterFunc(2*time.Second, func() {
				txHash := tx.Hash()
				h.backend.mine(&txHash, gasPrice)
			})
		}
		return tx, nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, sendTxFunc)
	require.Equal(t, err, txmgr.ErrPublishTimeout)
	require.Nil(t, receipt)
}

// errRpcFailure is a sentinel error used in testing to fail publications.
var errRpcFailure = errors.New("rpc failure")

// TestTxMgrBlocksOnFailingRpcCalls asserts that if all of the publication
// attempts fail due to rpc failures, that the tx manager will return
// txmgr.ErrPublishTimeout.
func TestTxMgrBlocksOnFailingRpcCalls(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	sendTxFunc := func(
		ctx context.Context,
		gasPrice *big.Int,
	) (*types.Transaction, error) {
		return nil, errRpcFailure
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, sendTxFunc)
	require.Equal(t, err, txmgr.ErrPublishTimeout)
	require.Nil(t, receipt)
}

// TestTxMgrOnlyOnePublicationSucceeds asserts that the tx manager will return a
// receipt so long as at least one of the publications is able to succeed with a
// simulated rpc failure.
func TestTxMgrOnlyOnePublicationSucceeds(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	sendTxFunc := func(
		ctx context.Context,
		gasPrice *big.Int,
	) (*types.Transaction, error) {
		// Fail all but the final attempt.
		if gasPrice.Cmp(h.cfg.MaxGasPrice) != 0 {
			return nil, errRpcFailure
		}

		tx := types.NewTx(&types.LegacyTx{
			GasPrice: gasPrice,
		})
		txHash := tx.Hash()
		h.backend.mine(&txHash, gasPrice)
		return tx, nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, sendTxFunc)
	require.Nil(t, err)

	require.NotNil(t, receipt)
	require.Equal(t, receipt.GasUsed, h.cfg.MaxGasPrice.Uint64())
}

// TestTxMgrConfirmsMinGasPriceAfterBumping delays the mining of the initial tx
// with the minimum gas price, and asserts that it's receipt is returned even
// though if the gas price has been bumped in other goroutines.
func TestTxMgrConfirmsMinGasPriceAfterBumping(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	sendTxFunc := func(
		ctx context.Context,
		gasPrice *big.Int,
	) (*types.Transaction, error) {
		tx := types.NewTx(&types.LegacyTx{
			GasPrice: gasPrice,
		})
		// Delay mining the tx with the min gas price.
		if gasPrice.Cmp(h.cfg.MinGasPrice) == 0 {
			time.AfterFunc(5*time.Second, func() {
				txHash := tx.Hash()
				h.backend.mine(&txHash, gasPrice)
			})
		}
		return tx, nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, sendTxFunc)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.GasUsed, h.cfg.MinGasPrice.Uint64())
}

// TestWaitMinedReturnsReceiptOnFirstSuccess insta-mines a transaction and
// asserts that WaitMined returns the appropriate receipt.
func TestWaitMinedReturnsReceiptOnFirstSuccess(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	// Create a tx and mine it immediately using the default backend.
	tx := types.NewTx(&types.LegacyTx{})
	txHash := tx.Hash()
	h.backend.mine(&txHash, new(big.Int))

	ctx := context.Background()
	receipt, err := txmgr.WaitMined(ctx, h.backend, tx, 50*time.Millisecond, 1)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.TxHash, txHash)
}

// TestWaitMinedCanBeCanceled ensures that WaitMined exits of the passed context
// is canceled before a receipt is found.
func TestWaitMinedCanBeCanceled(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create an unimined tx.
	tx := types.NewTx(&types.LegacyTx{})

	receipt, err := txmgr.WaitMined(ctx, h.backend, tx, 50*time.Millisecond, 1)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// TestWaitMinedMultipleConfs asserts that WaitMiend will properly wait for more
// than one confirmation.
func TestWaitMinedMultipleConfs(t *testing.T) {
	t.Parallel()

	const numConfs = 2

	h := newTestHarnessWithConfig(configWithNumConfs(numConfs))
	ctxt, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Create an unimined tx.
	tx := types.NewTx(&types.LegacyTx{})
	txHash := tx.Hash()
	h.backend.mine(&txHash, new(big.Int))

	receipt, err := txmgr.WaitMined(ctxt, h.backend, tx, 50*time.Millisecond, numConfs)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)

	ctxt, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Mine an empty block, tx should now be confirmed.
	h.backend.mine(nil, nil)
	receipt, err = txmgr.WaitMined(ctxt, h.backend, tx, 50*time.Millisecond, numConfs)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, txHash, receipt.TxHash)
}

// TestManagerPanicOnZeroConfs ensures that the NewSimpleTxManager will panic
// when attempting to configure with NumConfirmations set to zero.
func TestManagerPanicOnZeroConfs(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NewSimpleTxManager should panic when using zero conf")
		}
	}()

	_ = newTestHarnessWithConfig(configWithNumConfs(0))
}

// failingBackend implements txmgr.ReceiptSource, returning a failure on the
// first call but a success on the second call. This allows us to test that the
// inner loop of WaitMined properly handles this case.
type failingBackend struct {
	returnSuccessBlockNumber bool
	returnSuccessReceipt     bool
}

// BlockNumber for the failingBackend returns errRpcFailure on the first
// invocation and a fixed block height on subsequent calls.
func (b *failingBackend) BlockNumber(ctx context.Context) (uint64, error) {
	if !b.returnSuccessBlockNumber {
		b.returnSuccessBlockNumber = true
		return 0, errRpcFailure
	}

	return 1, nil
}

// TransactionReceipt for the failingBackend returns errRpcFailure on the first
// invocation, and a receipt containing the passed TxHash on the second.
func (b *failingBackend) TransactionReceipt(
	ctx context.Context, txHash common.Hash) (*types.Receipt, error) {

	if !b.returnSuccessReceipt {
		b.returnSuccessReceipt = true
		return nil, errRpcFailure
	}

	return &types.Receipt{
		TxHash:      txHash,
		BlockNumber: big.NewInt(1),
	}, nil
}

// TestWaitMinedReturnsReceiptAfterFailure asserts that WaitMined is able to
// recover from failed calls to the backend. It uses the failedBackend to
// simulate an rpc call failure, followed by the successful return of a receipt.
func TestWaitMinedReturnsReceiptAfterFailure(t *testing.T) {
	t.Parallel()

	var borkedBackend failingBackend

	// Don't mine the tx with the default backend. The failingBackend will
	// return the txHash on the second call.
	tx := types.NewTx(&types.LegacyTx{})
	txHash := tx.Hash()

	ctx := context.Background()
	receipt, err := txmgr.WaitMined(ctx, &borkedBackend, tx, 50*time.Millisecond, 1)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.TxHash, txHash)
}
