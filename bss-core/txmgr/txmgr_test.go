package txmgr_test

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/bss-core/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// testHarness houses the necessary resources to test the SimpleTxManager.
type testHarness struct {
	cfg       txmgr.Config
	mgr       txmgr.TxManager
	backend   *mockBackend
	gasPricer *gasPricer
}

// newTestHarnessWithConfig initializes a testHarness with a specific
// configuration.
func newTestHarnessWithConfig(cfg txmgr.Config) *testHarness {
	backend := newMockBackend()
	mgr := txmgr.NewSimpleTxManager("TEST", cfg, backend)

	return &testHarness{
		cfg:       cfg,
		mgr:       mgr,
		backend:   backend,
		gasPricer: newGasPricer(3),
	}
}

// newTestHarness initializes a testHarness with a default configuration that is
// suitable for most tests.
func newTestHarness() *testHarness {
	return newTestHarnessWithConfig(configWithNumConfs(1))
}

func configWithNumConfs(numConfirmations uint64) txmgr.Config {
	return txmgr.Config{
		ResubmissionTimeout:       time.Second,
		ReceiptQueryInterval:      50 * time.Millisecond,
		NumConfirmations:          numConfirmations,
		SafeAbortNonceTooLowCount: 3,
	}
}

type gasPricer struct {
	epoch         int64
	mineAtEpoch   int64
	baseGasTipFee *big.Int
	baseBaseFee   *big.Int
	mu            sync.Mutex
}

func newGasPricer(mineAtEpoch int64) *gasPricer {
	return &gasPricer{
		mineAtEpoch:   mineAtEpoch,
		baseGasTipFee: big.NewInt(5),
		baseBaseFee:   big.NewInt(7),
	}
}

func (g *gasPricer) expGasFeeCap() *big.Int {
	_, gasFeeCap := g.feesForEpoch(g.mineAtEpoch)
	return gasFeeCap
}

func (g *gasPricer) shouldMine(gasFeeCap *big.Int) bool {
	return g.expGasFeeCap().Cmp(gasFeeCap) == 0
}

func (g *gasPricer) feesForEpoch(epoch int64) (*big.Int, *big.Int) {
	epochBaseFee := new(big.Int).Mul(g.baseBaseFee, big.NewInt(epoch))
	epochGasTipCap := new(big.Int).Mul(g.baseGasTipFee, big.NewInt(epoch))
	epochGasFeeCap := txmgr.CalcGasFeeCap(epochBaseFee, epochGasTipCap)

	return epochGasTipCap, epochGasFeeCap
}

func (g *gasPricer) sample() (*big.Int, *big.Int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.epoch++
	epochGasTipCap, epochGasFeeCap := g.feesForEpoch(g.epoch)

	return epochGasTipCap, epochGasFeeCap
}

type minedTxInfo struct {
	gasFeeCap   *big.Int
	blockNumber uint64
	reverted    bool
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

// mine records a (txHash, gasFeeCap) as confirmed. Subsequent calls to
// TransactionReceipt with a matching txHash will result in a non-nil receipt.
// If a nil txHash is supplied this has the effect of mining an empty block.
func (b *mockBackend) mine(txHash *common.Hash, gasFeeCap *big.Int) {
	b.mineWithStatus(txHash, gasFeeCap, false)
}

// mineWithStatus records a (txHash, gasFeeCap) pair as confirmed, but also
// includes the option to specify whether or not the transaction reverted.
// Subsequent calls to TransactionReceipt with a matching txHash will result in
// a non-nil receipt. If a nil txHash is supplied this has the effect of mining
// an empty block.
func (b *mockBackend) mineWithStatus(
	txHash *common.Hash,
	gasFeeCap *big.Int,
	revert bool,
) {

	b.mu.Lock()
	defer b.mu.Unlock()

	b.blockHeight++
	if txHash != nil {
		b.minedTxs[*txHash] = minedTxInfo{
			gasFeeCap:   gasFeeCap,
			blockNumber: b.blockHeight,
			reverted:    revert,
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
// receipt containing the txHash and the gasFeeCap used in the GasUsed to make
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

	var status = types.ReceiptStatusSuccessful
	if txInfo.reverted {
		status = types.ReceiptStatusFailed
	}

	// Return the gas fee cap for the transaction in the GasUsed field so that
	// we can assert the proper tx confirmed in our tests.
	return &types.Receipt{
		TxHash:      txHash,
		GasUsed:     txInfo.gasFeeCap.Uint64(),
		BlockNumber: big.NewInt(int64(txInfo.blockNumber)),
		Status:      status,
	}, nil
}

// TestTxMgrConfirmAtMinGasPrice asserts that Send returns the min gas price tx
// if the tx is mined instantly.
func TestTxMgrConfirmAtMinGasPrice(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	gasPricer := newGasPricer(1)

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		if gasPricer.shouldMine(tx.GasFeeCap()) {
			txHash := tx.Hash()
			h.backend.mine(&txHash, tx.GasFeeCap())
		}
		return nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// TestTxMgrFailsForRevertedTxn asserts that Send returns ErrReverted if the
// confirmed transaction reverts during execution, and returns the resulting
// receipt.
func TestTxMgrFailsForRevertedTxn(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	gasPricer := newGasPricer(1)

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		if gasPricer.shouldMine(tx.GasFeeCap()) {
			txHash := tx.Hash()
			h.backend.mineWithStatus(&txHash, tx.GasFeeCap(), true)
		}
		return nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Equal(t, txmgr.ErrReverted, err)
	require.NotNil(t, receipt)
	require.Equal(t, gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// TestTxMgrNeverConfirmCancel asserts that a Send can be canceled even if no
// transaction is mined. This is done to ensure the the tx mgr can properly
// abort on shutdown, even if a txn is in the process of being published.
func TestTxMgrNeverConfirmCancel(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		// Don't publish tx to backend, simulating never being mined.
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// TestTxMgrConfirmsAtMaxGasPrice asserts that Send properly returns the max gas
// price receipt if none of the lower gas price txs were mined.
func TestTxMgrConfirmsAtHigherGasPrice(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		if h.gasPricer.shouldMine(tx.GasFeeCap()) {
			txHash := tx.Hash()
			h.backend.mine(&txHash, tx.GasFeeCap())
		}
		return nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// errRpcFailure is a sentinel error used in testing to fail publications.
var errRpcFailure = errors.New("rpc failure")

// TestTxMgrBlocksOnFailingRpcCalls asserts that if all of the publication
// attempts fail due to rpc failures, that the tx manager will return
// txmgr.ErrPublishTimeout.
func TestTxMgrBlocksOnFailingRpcCalls(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		return errRpcFailure
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// TestTxMgrOnlyOnePublicationSucceeds asserts that the tx manager will return a
// receipt so long as at least one of the publications is able to succeed with a
// simulated rpc failure.
func TestTxMgrOnlyOnePublicationSucceeds(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		// Fail all but the final attempt.
		if !h.gasPricer.shouldMine(tx.GasFeeCap()) {
			return errRpcFailure
		}

		txHash := tx.Hash()
		h.backend.mine(&txHash, tx.GasFeeCap())
		return nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)

	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// TestTxMgrConfirmsMinGasPriceAfterBumping delays the mining of the initial tx
// with the minimum gas price, and asserts that it's receipt is returned even
// though if the gas price has been bumped in other goroutines.
func TestTxMgrConfirmsMinGasPriceAfterBumping(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		// Delay mining the tx with the min gas price.
		if h.gasPricer.shouldMine(tx.GasFeeCap()) {
			time.AfterFunc(5*time.Second, func() {
				txHash := tx.Hash()
				h.backend.mine(&txHash, tx.GasFeeCap())
			})
		}
		return nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// TestTxMgrDoesntAbortNonceTooLowAfterMiningTx
func TestTxMgrDoesntAbortNonceTooLowAfterMiningTx(t *testing.T) {
	t.Parallel()

	h := newTestHarnessWithConfig(configWithNumConfs(2))

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		switch {

		// If the txn's gas fee cap is less than the one we expect to mine,
		// accept the txn to the mempool.
		case tx.GasFeeCap().Cmp(h.gasPricer.expGasFeeCap()) < 0:
			return nil

		// Accept and mine the actual txn we expect to confirm.
		case h.gasPricer.shouldMine(tx.GasFeeCap()):
			txHash := tx.Hash()
			h.backend.mine(&txHash, tx.GasFeeCap())
			time.AfterFunc(5*time.Second, func() {
				h.backend.mine(nil, nil)
			})
			return nil

		// For gas prices greater than our expected, return ErrNonceTooLow since
		// the prior txn confirmed and will invalidate subsequent publications.
		default:
			return core.ErrNonceTooLow
		}
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
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
		Status:      types.ReceiptStatusSuccessful,
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
