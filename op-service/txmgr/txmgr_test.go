package txmgr

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type sendTransactionFunc func(ctx context.Context, tx *types.Transaction) error

func testSendState() *SendState {
	return NewSendState(100, time.Hour)
}

// testHarness houses the necessary resources to test the SimpleTxManager.
type testHarness struct {
	cfg       Config
	mgr       *SimpleTxManager
	backend   *mockBackend
	gasPricer *gasPricer
}

// newTestHarnessWithConfig initializes a testHarness with a specific
// configuration.
func newTestHarnessWithConfig(t *testing.T, cfg Config) *testHarness {
	g := newGasPricer(3)
	backend := newMockBackend(g)
	cfg.Backend = backend
	mgr := &SimpleTxManager{
		chainID: cfg.ChainID,
		name:    "TEST",
		cfg:     cfg,
		backend: cfg.Backend,
		l:       testlog.Logger(t, log.LvlCrit),
		metr:    &metrics.NoopTxMetrics{},
	}

	return &testHarness{
		cfg:       cfg,
		mgr:       mgr,
		backend:   backend,
		gasPricer: g,
	}
}

// newTestHarness initializes a testHarness with a default configuration that is
// suitable for most tests.
func newTestHarness(t *testing.T) *testHarness {
	return newTestHarnessWithConfig(t, configWithNumConfs(1))
}

// createTxCandidate creates a mock [TxCandidate].
func (h testHarness) createTxCandidate() TxCandidate {
	inbox := common.HexToAddress("0x42000000000000000000000000000000000000ff")
	return TxCandidate{
		To:       &inbox,
		TxData:   []byte{0x00, 0x01, 0x02},
		GasLimit: uint64(1337),
	}
}

func configWithNumConfs(numConfirmations uint64) Config {
	return Config{
		ResubmissionTimeout:       time.Second,
		ReceiptQueryInterval:      50 * time.Millisecond,
		NumConfirmations:          numConfirmations,
		SafeAbortNonceTooLowCount: 3,
		FeeLimitMultiplier:        5,
		TxNotInMempoolTimeout:     1 * time.Hour,
		Signer: func(ctx context.Context, from common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return tx, nil
		},
		From: common.Address{},
	}
}

type gasPricer struct {
	epoch         int64
	mineAtEpoch   int64
	baseGasTipFee *big.Int
	baseBaseFee   *big.Int
	err           error
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
	epochGasFeeCap := calcGasFeeCap(epochBaseFee, epochGasTipCap)

	return epochGasTipCap, epochGasFeeCap
}

func (g *gasPricer) basefee() *big.Int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return new(big.Int).Mul(g.baseBaseFee, big.NewInt(g.epoch))
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
}

// mockBackend implements ReceiptSource that tracks mined transactions
// along with the gas price used.
type mockBackend struct {
	mu sync.RWMutex

	g    *gasPricer
	send sendTransactionFunc

	// blockHeight tracks the current height of the chain.
	blockHeight uint64

	// minedTxs maps the hash of a mined transaction to its details.
	minedTxs map[common.Hash]minedTxInfo
}

// newMockBackend initializes a new mockBackend.
func newMockBackend(g *gasPricer) *mockBackend {
	return &mockBackend{
		g:        g,
		minedTxs: make(map[common.Hash]minedTxInfo),
	}
}

// setTxSender sets the implementation for the sendTransactionFunction
func (b *mockBackend) setTxSender(s sendTransactionFunc) {
	b.send = s
}

// mine records a (txHash, gasFeeCap) as confirmed. Subsequent calls to
// TransactionReceipt with a matching txHash will result in a non-nil receipt.
// If a nil txHash is supplied this has the effect of mining an empty block.
func (b *mockBackend) mine(txHash *common.Hash, gasFeeCap *big.Int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.blockHeight++
	if txHash != nil {
		b.minedTxs[*txHash] = minedTxInfo{
			gasFeeCap:   gasFeeCap,
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

// Call mocks a call to the EVM.
func (b *mockBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return nil, nil
}

func (b *mockBackend) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return &types.Header{
		BaseFee: b.g.basefee(),
	}, nil
}

func (b *mockBackend) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	if b.g.err != nil {
		return 0, b.g.err
	}
	if msg.GasFeeCap.Cmp(msg.GasTipCap) < 0 {
		return 0, core.ErrTipAboveFeeCap
	}
	return b.g.basefee().Uint64(), nil
}

func (b *mockBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	tip, _ := b.g.sample()
	return tip, nil
}

func (b *mockBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if b.send == nil {
		panic("set sender function was not set")
	}
	return b.send(ctx, tx)
}

func (b *mockBackend) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return 0, nil
}

func (b *mockBackend) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return 0, nil
}

func (*mockBackend) ChainID(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

// TransactionReceipt queries the mockBackend for a mined txHash. If none is
// found, nil is returned for both return values. Otherwise, it returns a
// receipt containing the txHash and the gasFeeCap used in the GasUsed to make
// the value accessible from our test framework.
func (b *mockBackend) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	txInfo, ok := b.minedTxs[txHash]
	if !ok {
		return nil, nil
	}

	// Return the gas fee cap for the transaction in the GasUsed field so that
	// we can assert the proper tx confirmed in our tests.
	return &types.Receipt{
		TxHash:      txHash,
		GasUsed:     txInfo.gasFeeCap.Uint64(),
		BlockNumber: big.NewInt(int64(txInfo.blockNumber)),
	}, nil
}

func (b *mockBackend) Close() {
}

// TestTxMgrConfirmAtMinGasPrice asserts that Send returns the min gas price tx
// if the tx is mined instantly.
func TestTxMgrConfirmAtMinGasPrice(t *testing.T) {
	t.Parallel()

	h := newTestHarness(t)

	gasPricer := newGasPricer(1)

	gasTipCap, gasFeeCap := gasPricer.sample()
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
	})

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		if gasPricer.shouldMine(tx.GasFeeCap()) {
			txHash := tx.Hash()
			h.backend.mine(&txHash, tx.GasFeeCap())
		}
		return nil
	}
	h.backend.setTxSender(sendTx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	receipt, err := h.mgr.sendTx(ctx, tx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// TestTxMgrNeverConfirmCancel asserts that a Send can be canceled even if no
// transaction is mined. This is done to ensure the the tx mgr can properly
// abort on shutdown, even if a txn is in the process of being published.
func TestTxMgrNeverConfirmCancel(t *testing.T) {
	t.Parallel()

	h := newTestHarness(t)

	gasTipCap, gasFeeCap := h.gasPricer.sample()
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
	})
	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		// Don't publish tx to backend, simulating never being mined.
		return nil
	}
	h.backend.setTxSender(sendTx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	receipt, err := h.mgr.sendTx(ctx, tx)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// TestTxMgrConfirmsAtMaxGasPrice asserts that Send properly returns the max gas
// price receipt if none of the lower gas price txs were mined.
func TestTxMgrConfirmsAtHigherGasPrice(t *testing.T) {
	t.Parallel()

	h := newTestHarness(t)

	gasTipCap, gasFeeCap := h.gasPricer.sample()
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
	})
	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		if h.gasPricer.shouldMine(tx.GasFeeCap()) {
			txHash := tx.Hash()
			h.backend.mine(&txHash, tx.GasFeeCap())
		}
		return nil
	}
	h.backend.setTxSender(sendTx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	receipt, err := h.mgr.sendTx(ctx, tx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// errRpcFailure is a sentinel error used in testing to fail publications.
var errRpcFailure = errors.New("rpc failure")

// TestTxMgrBlocksOnFailingRpcCalls asserts that if all of the publication
// attempts fail due to rpc failures, that the tx manager will return
// ErrPublishTimeout.
func TestTxMgrBlocksOnFailingRpcCalls(t *testing.T) {
	t.Parallel()

	h := newTestHarness(t)

	gasTipCap, gasFeeCap := h.gasPricer.sample()
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
	})

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		return errRpcFailure
	}
	h.backend.setTxSender(sendTx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	receipt, err := h.mgr.sendTx(ctx, tx)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// TestTxMgr_CraftTx ensures that the tx manager will create transactions as expected.
func TestTxMgr_CraftTx(t *testing.T) {
	t.Parallel()
	h := newTestHarness(t)
	candidate := h.createTxCandidate()

	// Craft the transaction.
	gasTipCap, gasFeeCap := h.gasPricer.feesForEpoch(h.gasPricer.epoch + 1)
	tx, err := h.mgr.craftTx(context.Background(), candidate)
	require.Nil(t, err)
	require.NotNil(t, tx)

	// Validate the gas tip cap and fee cap.
	require.Equal(t, gasTipCap, tx.GasTipCap())
	require.Equal(t, gasFeeCap, tx.GasFeeCap())

	// Validate the nonce was set correctly using the backend.
	require.Zero(t, tx.Nonce())

	// Check that the gas was set using the gas limit.
	require.Equal(t, candidate.GasLimit, tx.Gas())
}

// TestTxMgr_EstimateGas ensures that the tx manager will estimate
// the gas when candidate gas limit is zero in [CraftTx].
func TestTxMgr_EstimateGas(t *testing.T) {
	t.Parallel()
	h := newTestHarness(t)
	candidate := h.createTxCandidate()

	// Set the gas limit to zero to trigger gas estimation.
	candidate.GasLimit = 0

	// Gas estimate
	gasEstimate := h.gasPricer.baseBaseFee.Uint64()

	// Craft the transaction.
	tx, err := h.mgr.craftTx(context.Background(), candidate)
	require.Nil(t, err)
	require.NotNil(t, tx)

	// Check that the gas was estimated correctly.
	require.Equal(t, gasEstimate, tx.Gas())
}

func TestTxMgr_EstimateGasFails(t *testing.T) {
	t.Parallel()
	h := newTestHarness(t)
	candidate := h.createTxCandidate()

	// Set the gas limit to zero to trigger gas estimation.
	candidate.GasLimit = 0

	// Craft a successful transaction.
	tx, err := h.mgr.craftTx(context.Background(), candidate)
	require.Nil(t, err)
	lastNonce := tx.Nonce()

	// Mock gas estimation failure.
	h.gasPricer.err = fmt.Errorf("execution error")
	_, err = h.mgr.craftTx(context.Background(), candidate)
	require.ErrorContains(t, err, "failed to estimate gas")

	// Ensure successful craft uses the correct nonce
	h.gasPricer.err = nil
	tx, err = h.mgr.craftTx(context.Background(), candidate)
	require.Nil(t, err)
	require.Equal(t, lastNonce+1, tx.Nonce())
}

func TestTxMgr_SigningFails(t *testing.T) {
	t.Parallel()
	errorSigning := false
	cfg := configWithNumConfs(1)
	cfg.Signer = func(ctx context.Context, from common.Address, tx *types.Transaction) (*types.Transaction, error) {
		if errorSigning {
			return nil, fmt.Errorf("signer error")
		} else {
			return tx, nil
		}
	}
	h := newTestHarnessWithConfig(t, cfg)
	candidate := h.createTxCandidate()

	// Set the gas limit to zero to trigger gas estimation.
	candidate.GasLimit = 0

	// Craft a successful transaction.
	tx, err := h.mgr.craftTx(context.Background(), candidate)
	require.Nil(t, err)
	lastNonce := tx.Nonce()

	// Mock signer failure.
	errorSigning = true
	_, err = h.mgr.craftTx(context.Background(), candidate)
	require.ErrorContains(t, err, "signer error")

	// Ensure successful craft uses the correct nonce
	errorSigning = false
	tx, err = h.mgr.craftTx(context.Background(), candidate)
	require.Nil(t, err)
	require.Equal(t, lastNonce+1, tx.Nonce())
}

// TestTxMgrOnlyOnePublicationSucceeds asserts that the tx manager will return a
// receipt so long as at least one of the publications is able to succeed with a
// simulated rpc failure.
func TestTxMgrOnlyOnePublicationSucceeds(t *testing.T) {
	t.Parallel()

	h := newTestHarness(t)

	gasTipCap, gasFeeCap := h.gasPricer.sample()
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
	})

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		// Fail all but the final attempt.
		if !h.gasPricer.shouldMine(tx.GasFeeCap()) {
			return errRpcFailure
		}

		txHash := tx.Hash()
		h.backend.mine(&txHash, tx.GasFeeCap())
		return nil
	}
	h.backend.setTxSender(sendTx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	receipt, err := h.mgr.sendTx(ctx, tx)
	require.Nil(t, err)

	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// TestTxMgrConfirmsMinGasPriceAfterBumping delays the mining of the initial tx
// with the minimum gas price, and asserts that it's receipt is returned even
// though if the gas price has been bumped in other goroutines.
func TestTxMgrConfirmsMinGasPriceAfterBumping(t *testing.T) {
	t.Parallel()

	h := newTestHarness(t)

	gasTipCap, gasFeeCap := h.gasPricer.sample()
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
	})

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
	h.backend.setTxSender(sendTx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	receipt, err := h.mgr.sendTx(ctx, tx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// TestTxMgrDoesntAbortNonceTooLowAfterMiningTx
func TestTxMgrDoesntAbortNonceTooLowAfterMiningTx(t *testing.T) {
	t.Parallel()

	h := newTestHarnessWithConfig(t, configWithNumConfs(2))

	gasTipCap, gasFeeCap := h.gasPricer.sample()
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
	})

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
	h.backend.setTxSender(sendTx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	receipt, err := h.mgr.sendTx(ctx, tx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// TestWaitMinedReturnsReceiptOnFirstSuccess insta-mines a transaction and
// asserts that waitMined returns the appropriate receipt.
func TestWaitMinedReturnsReceiptOnFirstSuccess(t *testing.T) {
	t.Parallel()

	h := newTestHarnessWithConfig(t, configWithNumConfs(1))

	// Create a tx and mine it immediately using the default backend.
	tx := types.NewTx(&types.LegacyTx{})
	txHash := tx.Hash()
	h.backend.mine(&txHash, new(big.Int))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	receipt, err := h.mgr.waitMined(ctx, tx, testSendState())
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.TxHash, txHash)
}

// TestWaitMinedCanBeCanceled ensures that waitMined exits of the passed context
// is canceled before a receipt is found.
func TestWaitMinedCanBeCanceled(t *testing.T) {
	t.Parallel()

	h := newTestHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create an unimined tx.
	tx := types.NewTx(&types.LegacyTx{})

	receipt, err := h.mgr.waitMined(ctx, tx, NewSendState(10, time.Hour))
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// TestWaitMinedMultipleConfs asserts that waitMined will properly wait for more
// than one confirmation.
func TestWaitMinedMultipleConfs(t *testing.T) {
	t.Parallel()

	const numConfs = 2

	h := newTestHarnessWithConfig(t, configWithNumConfs(numConfs))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Create an unimined tx.
	tx := types.NewTx(&types.LegacyTx{})
	txHash := tx.Hash()
	h.backend.mine(&txHash, new(big.Int))

	receipt, err := h.mgr.waitMined(ctx, tx, NewSendState(10, time.Hour))
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Mine an empty block, tx should now be confirmed.
	h.backend.mine(nil, nil)
	receipt, err = h.mgr.waitMined(ctx, tx, NewSendState(10, time.Hour))
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, txHash, receipt.TxHash)
}

// TestManagerErrsOnZeroCLIConfs ensures that the NewSimpleTxManager will error
// when attempting to configure with NumConfirmations set to zero.
func TestManagerErrsOnZeroCLIConfs(t *testing.T) {
	t.Parallel()

	_, err := NewSimpleTxManager("TEST", testlog.Logger(t, log.LvlCrit), &metrics.NoopTxMetrics{}, CLIConfig{})
	require.Error(t, err)
}

// TestManagerErrsOnZeroConfs ensures that the NewSimpleTxManager will error
// when attempting to configure with NumConfirmations set to zero.
func TestManagerErrsOnZeroConfs(t *testing.T) {
	t.Parallel()

	_, err := NewSimpleTxManagerFromConfig("TEST", testlog.Logger(t, log.LvlCrit), &metrics.NoopTxMetrics{}, Config{})
	require.Error(t, err)
}

// failingBackend implements ReceiptSource, returning a failure on the
// first call but a success on the second call. This allows us to test that the
// inner loop of WaitMined properly handles this case.
type failingBackend struct {
	returnSuccessBlockNumber bool
	returnSuccessReceipt     bool
	baseFee, gasTip          *big.Int
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
	ctx context.Context, txHash common.Hash,
) (*types.Receipt, error) {
	if !b.returnSuccessReceipt {
		b.returnSuccessReceipt = true
		return nil, errRpcFailure
	}

	return &types.Receipt{
		TxHash:      txHash,
		BlockNumber: big.NewInt(1),
	}, nil
}

func (b *failingBackend) HeaderByNumber(_ context.Context, _ *big.Int) (*types.Header, error) {
	return &types.Header{
		BaseFee: b.baseFee,
	}, nil
}

func (b *failingBackend) CallContract(_ context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (b *failingBackend) SendTransaction(_ context.Context, _ *types.Transaction) error {
	return errors.New("unimplemented")
}

func (b *failingBackend) SuggestGasTipCap(_ context.Context) (*big.Int, error) {
	return b.gasTip, nil
}

func (b *failingBackend) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	return b.baseFee.Uint64(), nil
}

func (b *failingBackend) NonceAt(_ context.Context, _ common.Address, _ *big.Int) (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (b *failingBackend) PendingNonceAt(_ context.Context, _ common.Address) (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (b *failingBackend) ChainID(ctx context.Context) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (b *failingBackend) Close() {
}

// TestWaitMinedReturnsReceiptAfterFailure asserts that WaitMined is able to
// recover from failed calls to the backend. It uses the failedBackend to
// simulate an rpc call failure, followed by the successful return of a receipt.
func TestWaitMinedReturnsReceiptAfterFailure(t *testing.T) {
	t.Parallel()

	var borkedBackend failingBackend

	mgr := &SimpleTxManager{
		cfg: Config{
			ResubmissionTimeout:       time.Second,
			ReceiptQueryInterval:      50 * time.Millisecond,
			NumConfirmations:          1,
			SafeAbortNonceTooLowCount: 3,
		},
		name:    "TEST",
		backend: &borkedBackend,
		l:       testlog.Logger(t, log.LvlCrit),
		metr:    &metrics.NoopTxMetrics{},
	}

	// Don't mine the tx with the default backend. The failingBackend will
	// return the txHash on the second call.
	tx := types.NewTx(&types.LegacyTx{})
	txHash := tx.Hash()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	receipt, err := mgr.waitMined(ctx, tx, testSendState())
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.TxHash, txHash)
}

func doGasPriceIncrease(t *testing.T, txTipCap, txFeeCap, newTip, newBaseFee int64) (*types.Transaction, *types.Transaction) {
	borkedBackend := failingBackend{
		gasTip:  big.NewInt(newTip),
		baseFee: big.NewInt(newBaseFee),
	}

	mgr := &SimpleTxManager{
		cfg: Config{
			ResubmissionTimeout:       time.Second,
			ReceiptQueryInterval:      50 * time.Millisecond,
			NumConfirmations:          1,
			SafeAbortNonceTooLowCount: 3,
			FeeLimitMultiplier:        5,
			Signer: func(ctx context.Context, from common.Address, tx *types.Transaction) (*types.Transaction, error) {
				return tx, nil
			},
			From: common.Address{},
		},
		name:    "TEST",
		backend: &borkedBackend,
		l:       testlog.Logger(t, log.LvlCrit),
		metr:    &metrics.NoopTxMetrics{},
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: big.NewInt(txTipCap),
		GasFeeCap: big.NewInt(txFeeCap),
	})
	newTx, err := mgr.increaseGasPrice(context.Background(), tx)
	require.NoError(t, err)
	return tx, newTx
}

func TestIncreaseGasPrice(t *testing.T) {
	// t.Parallel()
	require.Equal(t, int64(10), priceBump, "test must be updated if priceBump is adjusted")
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "bump at least 1",
			run: func(t *testing.T) {
				tx, newTx := doGasPriceIncrease(t, 1, 3, 1, 1)
				require.True(t, newTx.GasFeeCap().Cmp(tx.GasFeeCap()) > 0, "new tx fee cap must be larger")
				require.True(t, newTx.GasTipCap().Cmp(tx.GasTipCap()) > 0, "new tx tip must be larger")
			},
		},
		{
			name: "enforces min bump",
			run: func(t *testing.T) {
				tx, newTx := doGasPriceIncrease(t, 100, 1000, 101, 460)
				require.True(t, newTx.GasFeeCap().Cmp(tx.GasFeeCap()) > 0, "new tx fee cap must be larger")
				require.True(t, newTx.GasTipCap().Cmp(tx.GasTipCap()) > 0, "new tx tip must be larger")
			},
		},
		{
			name: "enforces min bump on only tip incrase",
			run: func(t *testing.T) {
				tx, newTx := doGasPriceIncrease(t, 100, 1000, 101, 440)
				require.True(t, newTx.GasFeeCap().Cmp(tx.GasFeeCap()) > 0, "new tx fee cap must be larger")
				require.True(t, newTx.GasTipCap().Cmp(tx.GasTipCap()) > 0, "new tx tip must be larger")
			},
		},
		{
			name: "enforces min bump on only basefee incrase",
			run: func(t *testing.T) {
				tx, newTx := doGasPriceIncrease(t, 100, 1000, 99, 460)
				require.True(t, newTx.GasFeeCap().Cmp(tx.GasFeeCap()) > 0, "new tx fee cap must be larger")
				require.True(t, newTx.GasTipCap().Cmp(tx.GasTipCap()) > 0, "new tx tip must be larger")
			},
		},
		{
			name: "uses L1 values when larger",
			run: func(t *testing.T) {
				_, newTx := doGasPriceIncrease(t, 10, 100, 50, 200)
				require.True(t, newTx.GasFeeCap().Cmp(big.NewInt(450)) == 0, "new tx fee cap must be equal L1")
				require.True(t, newTx.GasTipCap().Cmp(big.NewInt(50)) == 0, "new tx tip must be equal L1")
			},
		},
		{
			name: "uses L1 tip when larger and threshold FC",
			run: func(t *testing.T) {
				_, newTx := doGasPriceIncrease(t, 100, 2200, 120, 1050)
				require.True(t, newTx.GasTipCap().Cmp(big.NewInt(120)) == 0, "new tx tip must be equal L1")
				require.True(t, newTx.GasFeeCap().Cmp(big.NewInt(2420)) == 0, "new tx fee cap must be equal to the threshold value")
			},
		},
		{
			name: "uses L1 FC when larger and threshold tip",
			run: func(t *testing.T) {
				_, newTx := doGasPriceIncrease(t, 100, 2200, 100, 2000)
				require.True(t, newTx.GasTipCap().Cmp(big.NewInt(110)) == 0, "new tx tip must be equal the threshold value")
				t.Log("Vals:", newTx.GasFeeCap())
				require.True(t, newTx.GasFeeCap().Cmp(big.NewInt(4110)) == 0, "new tx fee cap must be equal L1")
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, test.run)
	}
}

// TestIncreaseGasPriceLimits asserts that if the L1 basefee & tip remain the
// same, repeated calls to IncreaseGasPrice eventually hit a limit.
func TestIncreaseGasPriceLimits(t *testing.T) {
	t.Run("no-threshold", func(t *testing.T) {
		testIncreaseGasPriceLimit(t, gasPriceLimitTest{
			expTipCap: 46,
			expFeeCap: 354, // just below 5*100
		})
	})
	t.Run("with-threshold", func(t *testing.T) {
		testIncreaseGasPriceLimit(t, gasPriceLimitTest{
			thr:       big.NewInt(params.GWei),
			expTipCap: 131_326_987,
			expFeeCap: 933_286_308, // just below 1 gwei
		})
	})
}

type gasPriceLimitTest struct {
	thr                  *big.Int
	expTipCap, expFeeCap int64
}

// testIncreaseGasPriceLimit runs a gas bumping test that increases the gas price until it hits an error.
// It starts with a tx that has a tip cap of 10 wei and fee cap of 100 wei.
func testIncreaseGasPriceLimit(t *testing.T, lt gasPriceLimitTest) {
	t.Parallel()

	borkedTip := int64(10)
	borkedFee := int64(45)
	borkedBackend := failingBackend{
		gasTip:  big.NewInt(borkedTip),
		baseFee: big.NewInt(borkedFee),
	}

	mgr := &SimpleTxManager{
		cfg: Config{
			ResubmissionTimeout:       time.Second,
			ReceiptQueryInterval:      50 * time.Millisecond,
			NumConfirmations:          1,
			SafeAbortNonceTooLowCount: 3,
			FeeLimitMultiplier:        5,
			FeeLimitThreshold:         lt.thr,
			Signer: func(ctx context.Context, from common.Address, tx *types.Transaction) (*types.Transaction, error) {
				return tx, nil
			},
			From: common.Address{},
		},
		name:    "TEST",
		backend: &borkedBackend,
		l:       testlog.Logger(t, log.LvlCrit),
		metr:    &metrics.NoopTxMetrics{},
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(100),
	})

	// Run IncreaseGasPrice a bunch of times in a row to simulate a very fast resubmit loop.
	ctx := context.Background()
	for {
		newTx, err := mgr.increaseGasPrice(ctx, tx)
		if err != nil {
			break
		}
		tx = newTx
	}

	lastTip, lastFee := tx.GasTipCap(), tx.GasFeeCap()
	// Confirm that fees only rose until expected threshold
	require.Equal(t, lt.expTipCap, lastTip.Int64())
	require.Equal(t, lt.expFeeCap, lastFee.Int64())
	_, err := mgr.increaseGasPrice(ctx, tx)
	require.Error(t, err)
}

func TestErrStringMatch(t *testing.T) {
	tests := []struct {
		err    error
		target error
		match  bool
	}{
		{err: nil, target: nil, match: true},
		{err: errors.New("exists"), target: nil, match: false},
		{err: nil, target: errors.New("exists"), match: false},
		{err: errors.New("exact match"), target: errors.New("exact match"), match: true},
		{err: errors.New("partial: match"), target: errors.New("match"), match: true},
	}

	for i, test := range tests {
		i := i
		test := test
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			require.Equal(t, test.match, errStringMatch(test.err, test.target))
		})
	}
}

func TestNonceReset(t *testing.T) {
	conf := configWithNumConfs(1)
	conf.SafeAbortNonceTooLowCount = 1
	h := newTestHarnessWithConfig(t, conf)

	index := -1
	var nonces []uint64
	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		index++
		nonces = append(nonces, tx.Nonce())
		// fail every 3rd tx
		if index%3 == 0 {
			return core.ErrNonceTooLow
		}
		txHash := tx.Hash()
		h.backend.mine(&txHash, tx.GasFeeCap())
		return nil
	}
	h.backend.setTxSender(sendTx)

	ctx := context.Background()
	for i := 0; i < 8; i++ {
		_, err := h.mgr.Send(ctx, TxCandidate{
			To: &common.Address{},
		})
		// expect every 3rd tx to fail
		if i%3 == 0 {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}

	// internal nonce tracking should be reset every 3rd tx
	require.Equal(t, []uint64{0, 0, 1, 2, 0, 1, 2, 0}, nonces)
}
