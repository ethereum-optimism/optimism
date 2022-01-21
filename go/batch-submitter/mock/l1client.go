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

	// NonceAt returns the account nonce of the given account. The block number
	// can be nil, in which case the nonce is taken from the latest known block.
	NonceAt func(context.Context, common.Address, *big.Int) (uint64, error)

	// SendTransaction injects a signed transaction into the pending pool for
	// execution.
	//
	// If the transaction was a contract creation use the TransactionReceipt
	// method to get the contract address after the transaction has been mined.
	SendTransaction func(context.Context, *types.Transaction) error

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

// EstimateGas executes the mock EstimateGas method.
func (c *L1Client) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.EstimateGas(ctx, call)
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

// SetEstimateGasFunc overrwrites the mock EstimateGas method.
func (c *L1Client) SetEstimateGasFunc(
	f func(context.Context, ethereum.CallMsg) (uint64, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.EstimateGas = f
}

// SetNonceAtFunc overrwrites the mock NonceAt method.
func (c *L1Client) SetNonceAtFunc(
	f func(context.Context, common.Address, *big.Int) (uint64, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.NonceAt = f
}

// SetSendTransactionFunc overrwrites the mock SendTransaction method.
func (c *L1Client) SetSendTransactionFunc(
	f func(context.Context, *types.Transaction) error) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.SendTransaction = f
}

// SetTransactionReceiptFunc overwrites the mock TransactionReceipt method.
func (c *L1Client) SetTransactionReceiptFunc(
	f func(context.Context, common.Hash) (*types.Receipt, error)) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.TransactionReceipt = f
}
