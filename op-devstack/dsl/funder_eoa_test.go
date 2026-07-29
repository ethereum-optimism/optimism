package dsl_test

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type funderTestEL struct {
	t       devtest.T
	chainID eth.ChainID
	client  apis.EthClient
}

var _ stack.L1ELNode = (*funderTestEL)(nil)

func (el *funderTestEL) T() devtest.T                      { return el.t }
func (el *funderTestEL) Logger() log.Logger                { return el.t.Logger() }
func (el *funderTestEL) Name() string                      { return "funder-test" }
func (el *funderTestEL) Label(string) string               { return "" }
func (el *funderTestEL) SetLabel(string, string)           {}
func (el *funderTestEL) ChainID() eth.ChainID              { return el.chainID }
func (el *funderTestEL) EthClient() apis.EthClient         { return el.client }
func (el *funderTestEL) UserRPC() string                   { return "" }
func (el *funderTestEL) TransactionTimeout() time.Duration { return 5 * time.Second }

func newFunderTestEL(t devtest.T, chainID eth.ChainID, client apis.EthClient) *dsl.L1ELNode {
	return dsl.NewL1ELNode(&funderTestEL{t: t, chainID: chainID, client: client})
}

type funderTestEthClient struct {
	apis.EthClient

	mu            sync.Mutex
	chainID       *big.Int
	pendingNonce  uint64
	sendCalls     int
	sendsReady    chan struct{}
	receiptsReady chan struct{}
	balances      map[common.Address]*big.Int
	transactions  []*types.Transaction
	receipts      map[common.Hash]*types.Receipt
}

var _ apis.EthClient = (*funderTestEthClient)(nil)

func newFunderTestEthClient(chainID *big.Int, pendingNonce uint64) *funderTestEthClient {
	return &funderTestEthClient{
		chainID:       chainID,
		pendingNonce:  pendingNonce,
		sendsReady:    make(chan struct{}),
		receiptsReady: make(chan struct{}),
		balances:      make(map[common.Address]*big.Int),
		receipts:      make(map[common.Hash]*types.Receipt),
	}
}

func (c *funderTestEthClient) ChainID(context.Context) (*big.Int, error) {
	return new(big.Int).Set(c.chainID), nil
}

func (c *funderTestEthClient) InfoByLabel(context.Context, eth.BlockLabel) (eth.BlockInfo, error) {
	return &testutils.MockBlockInfo{InfoBaseFee: big.NewInt(1), InfoGasLimit: 30_000_000}, nil
}

func (c *funderTestEthClient) EstimateGas(context.Context, ethereum.CallMsg) (uint64, error) {
	return 21_000, nil
}

func (c *funderTestEthClient) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	return c.pendingNonce, nil
}

func copyBig(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(v)
}

func (c *funderTestEthClient) BalanceAt(_ context.Context, account common.Address, _ *big.Int) (*big.Int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return copyBig(c.balances[account]), nil
}

func (c *funderTestEthClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	// Hold both submissions at the EL boundary so nonce allocation is concurrent
	// regardless of goroutine scheduling.
	c.mu.Lock()
	c.sendCalls++
	if c.sendCalls == 2 {
		close(c.sendsReady)
	}
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.sendsReady:
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	txHash := tx.Hash()
	if _, submitted := c.receipts[txHash]; submitted {
		return nil
	}
	c.transactions = append(c.transactions, tx)
	to := *tx.To()
	balance := copyBig(c.balances[to])
	c.balances[to] = balance.Add(balance, tx.Value())
	c.receipts[txHash] = &types.Receipt{Status: types.ReceiptStatusSuccessful, TxHash: txHash}
	if len(c.transactions) == 2 {
		close(c.receiptsReady)
	}
	return nil
}

func (c *funderTestEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.receiptsReady:
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if receipt := c.receipts[txHash]; receipt != nil {
		return receipt, nil
	}
	return nil, ethereum.NotFound
}

func (c *funderTestEthClient) submittedTransactions() []*types.Transaction {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*types.Transaction(nil), c.transactions...)
}

func TestFunderEOAViewsFundConcurrently(t *testing.T) {
	dt := devtest.SerialT(t)
	ctx, cancel := context.WithTimeout(dt.Ctx(), 10*time.Second)
	defer cancel()
	dt = dt.WithCtx(ctx)

	chainID := eth.ChainIDFromUInt64(901)
	const startNonce = uint64(7)
	client := newFunderTestEthClient(chainID.ToBig(), startNonce)
	originalEL := newFunderTestEL(dt, chainID, client)
	reboundEL := newFunderTestEL(dt, chainID, client)

	privateKey, err := crypto.GenerateKey()
	dt.Require().NoError(err)
	wallet := dsl.NewRandomHDWallet(dt, 0)
	funder := dsl.NewFunderEOA(dsl.NewEOA(dsl.NewKey(dt, privateKey), originalEL), wallet)
	rebound := funder.AsFunder(reboundEL)
	recipients := [2]*dsl.EOA{wallet.NewEOA(originalEL), wallet.NewEOA(reboundEL)}
	amounts := [2]eth.ETH{eth.OneHundredthEther, eth.OneTenthEther}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		funder.Fund(recipients[0], amounts[0])
	}()
	go func() {
		defer wg.Done()
		rebound.Fund(recipients[1], amounts[1])
	}()
	wg.Wait()

	transactions := client.submittedTransactions()
	dt.Require().Len(transactions, 2)
	dt.Require().ElementsMatch(
		[]uint64{startNonce, startNonce + 1},
		[]uint64{transactions[0].Nonce(), transactions[1].Nonce()},
		"concurrent funding transactions must use distinct nonces",
	)
	recipients[0].VerifyBalanceExact(amounts[0])
	recipients[1].VerifyBalanceExact(amounts[1])
}
