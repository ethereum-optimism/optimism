package mock

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// L1ClientConfig houses the internal methods that are executed by the mock
// L1Client. Any members left as nil will panic on execution.
type L1ClientConfig struct {
	// BlockNumber returns the most recent block number.
	BlockNumber func(context.Context) (uint64, error)

	// EstimateGas tries to estimate the gas needed to execute a specific
	// transaction based on the current pending state of the backend blockchain.
	// There is no guarantee that this is the true gas limit requirement as
	// other transactions may be added or removed by miners, but it should
	// provide a basis for setting a reasonable default.
	EstimateGas func(context.Context, ethereum.CallMsg) (uint64, error)

	// HeaderByNumber returns a block header from the current canonical chain.
	// If number is nil, the latest known header is returned.
	HeaderByNumber func(context.Context, *big.Int) (*types.Header, error)

	// NonceAt returns the account nonce of the given account. The block number
	// can be nil, in which case the nonce is taken from the latest known block.
	NonceAt func(context.Context, common.Address, *big.Int) (uint64, error)

	// SendTransaction injects a signed transaction into the pending pool for
	// execution.
	//
	// If the transaction was a contract creation use the TransactionReceipt
	// method to get the contract address after the transaction has been mined.
	SendTransaction func(context.Context, *types.Transaction) error

	// SuggestGasTipCap retrieves the currently suggested gas tip cap after 1559
	// to allow a timely execution of a transaction.
	SuggestGasTipCap func(context.Context) (*big.Int, error)

	// TransactionReceipt returns the receipt of a transaction by transaction
	// hash. Note that the receipt is not available for pending transactions.
	TransactionReceipt func(context.Context, common.Hash) (*types.Receipt, error)
}

// L1Client represents a mock L1Client.
type L1Client struct {
	cfg L1ClientConfig
	mu  sync.RWMutex
}

// NewL1Client returns a new L1Client using the mocked methods in the
// L1ClientConfig.
func NewL1Client(cfg L1ClientConfig) *L1Client {
	return &L1Client{
		cfg: cfg,
	}
}

// BlockNumber returns the most recent block number.
func (c *L1Client) BlockNumber(ctx context.Context) (uint64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.BlockNumber(ctx)
}

// EstimateGas tries to estimate the gas needed to execute a specific
// transaction based on the current pending state of the backend blockchain.
// There is no guarantee that this is the true gas limit requirement as other
// transactions may be added or removed by miners, but it should provide a basis
// for setting a reasonable default.
func (c *L1Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.EstimateGas(ctx, msg)
}

// HeaderByNumber returns a block header from the current canonical chain. If
// number is nil, the latest known header is returned.
func (c *L1Client) HeaderByNumber(ctx context.Context, blockNumber *big.Int) (*types.Header, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.HeaderByNumber(ctx, blockNumber)
}

// NonceAt executes the mock NonceAt method.
func (c *L1Client) NonceAt(ctx context.Context, addr common.Address, blockNumber *big.Int) (uint64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.NonceAt(ctx, addr, blockNumber)
}

// SendTransaction executes the mock SendTransaction method.
func (c *L1Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.SendTransaction(ctx, tx)
}

// SuggestGasTipCap retrieves the currently suggested gas tip cap after 1559 to
// allow a timely execution of a transaction.
func (c *L1Client) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.SuggestGasTipCap(ctx)
}

// TransactionReceipt executes the mock TransactionReceipt method.
func (c *L1Client) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.TransactionReceipt(ctx, txHash)
}

// SetBlockNumberFunc overwrites the mock BlockNumber method.
func (c *L1Client) SetBlockNumberFunc(
	f func(context.Context) (uint64, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.BlockNumber = f
}

// SetEstimateGasFunc overwrites the mock EstimateGas method.
func (c *L1Client) SetEstimateGasFunc(
	f func(context.Context, ethereum.CallMsg) (uint64, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.EstimateGas = f
}

// SetHeaderByNumberFunc overwrites the mock HeaderByNumber method.
func (c *L1Client) SetHeaderByNumberFunc(
	f func(ctx context.Context, blockNumber *big.Int) (*types.Header, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.HeaderByNumber = f
}

// SetNonceAtFunc overwrites the mock NonceAt method.
func (c *L1Client) SetNonceAtFunc(
	f func(context.Context, common.Address, *big.Int) (uint64, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.NonceAt = f
}

// SetSendTransactionFunc overwrites the mock SendTransaction method.
func (c *L1Client) SetSendTransactionFunc(
	f func(context.Context, *types.Transaction) error) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.SendTransaction = f
}

// SetSuggestGasTipCapFunc overwrites themock SuggestGasTipCap method.
func (c *L1Client) SetSuggestGasTipCapFunc(
	f func(context.Context) (*big.Int, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.SuggestGasTipCap = f
}

// SetTransactionReceiptFunc overwrites the mock TransactionReceipt method.
func (c *L1Client) SetTransactionReceiptFunc(
	f func(context.Context, common.Hash) (*types.Receipt, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.TransactionReceipt = f
}
