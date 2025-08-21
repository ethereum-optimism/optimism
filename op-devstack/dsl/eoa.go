package dsl

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	e2eBindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	txIntentBindings "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EOA is an Externally-Owned-Account:
// an account on a specific chain that is operated by a single private key.
type EOA struct {
	commonImpl

	key *Key

	// el is the execution-layer node that this user operates against.
	// This may be a L1 or L2 EL node.
	el ELNode
}

func NewEOA(key *Key, el ELNode) *EOA {
	return &EOA{
		commonImpl: commonFromT(key.t),
		el:         el,
		key:        key,
	}
}

func (u *EOA) AsEL(el ELNode) *EOA {
	return NewEOA(u.key, el)
}

func (u *EOA) String() string {
	return fmt.Sprintf("EOA(%s @ %s)", u.key.Address(), u.el.ChainID())
}

func (u *EOA) Address() common.Address {
	return u.key.Address()
}

// Key returns the cross-chain user identity/key,
// i.e. the user but detached it from the EL node.
func (u *EOA) Key() *Key {
	return u.key
}

func (u *EOA) ChainID() eth.ChainID {
	return u.el.ChainID()
}

// Plan creates the default tx-planning options,
// to perform a transaction with this Key,
// against the connected EL node and its chain.
func (u *EOA) Plan() txplan.Option {
	elClient := u.el.stackEL().EthClient()
	return txplan.Combine(
		txplan.WithChainID(elClient),
		u.key.Plan(),
		txplan.WithPendingNonce(elClient),
		txplan.WithAgainstLatestBlock(elClient),
		txplan.WithEstimator(elClient, true),
		txplan.WithRetrySubmission(elClient, 5, retry.Exponential()),
		txplan.WithRetryInclusion(elClient, 5, retry.Exponential()),
		txplan.WithBlockInclusionInfo(elClient),
	)
}

func (u *EOA) PlanAuth(code common.Address) txplan.Option {
	toAddr := u.Address()
	return txplan.Combine(
		u.Plan(),
		txplan.WithType(types.SetCodeTxType),
		txplan.WithTo(&toAddr),
		txplan.WithAuthorizationTo(code),
		// Set a fixed gas limit because eth_estimateGas doesn't consider authorizations yet.
		txplan.WithGasLimit(75_000),
	)
}

// PlanTransfer creates the tx-plan options to perform a transfer
// of the given amount of ETH to the given account.
func (u *EOA) PlanTransfer(to common.Address, amount eth.ETH) txplan.Option {
	return txplan.Combine(
		u.Plan(),
		txplan.WithTo(&to),
		txplan.WithValue(amount),
		// Don't set gas explicitly since the transfer might be to a contract
	)
}

// Transfer transfers the given amount of ETH to the given account, immediately.
func (u *EOA) Transfer(to common.Address, amount eth.ETH) *txplan.PlannedTx {
	return u.Transact(u.PlanTransfer(to, amount))
}

// Transact plans and executes a tx.
// The success-state, as defined by the tx-plan options, is required.
// The resulting evaluated tx is returned.
func (u *EOA) Transact(opts ...txplan.Option) *txplan.PlannedTx {
	opt := txplan.Combine(opts...)
	tx := txplan.NewPlannedTx(opt)
	_, err := tx.Success.Eval(u.ctx)
	u.require.NoError(err, "must transact")
	return tx
}

// balance looks up the user balance in the latest block.
// It is not exposed publicly in DSL: see methods like VerifyBalance instead.
func (u *EOA) balance() eth.ETH {
	result, err := retry.Do(u.ctx, 3, retry.Exponential(), func() (*big.Int, error) {
		return u.el.stackEL().EthClient().BalanceAt(u.ctx, u.Address(), nil)
	})
	u.t.Require().NoError(err, "must lookup balance")
	return eth.WeiBig(result)
}

// Try to avoid using this method where possible, use the VerifyBalance* methods instead.
func (u *EOA) GetBalance() eth.ETH {
	return u.balance()
}

// VerifyBalanceLessThan verifies balance < v
func (u *EOA) VerifyBalanceLessThan(v eth.ETH) {
	actual := u.balance()
	u.t.Require().True(actual.Lt(v), "got %s, expecting less than %s", actual, v)
}

// VerifyBalanceExact verifies balance == v
func (u *EOA) VerifyBalanceExact(v eth.ETH) {
	actual := u.balance()
	u.t.Require().Equal(v, actual, "must have expected balance")
}

// VerifyBalanceAtLeast verifies balance >= v
func (u *EOA) VerifyBalanceAtLeast(v eth.ETH) {
	actual := u.balance()
	u.t.Require().GreaterOrEqual(actual, v, "got %s, expecting at least %s", actual, v)
}

func (u *EOA) WaitForBalance(v eth.ETH) {
	u.t.Require().Eventually(func() bool {
		u.VerifyBalanceExact(v)
		return true
	}, u.el.stackEL().TransactionTimeout(), time.Second, "awaiting balance to be updated")
}

func (u *EOA) DeployEventLogger() common.Address {
	tx := txplan.NewPlannedTx(u.Plan(), txplan.WithData(common.FromHex(bindings.EventloggerBin)))
	res, err := tx.Included.Eval(u.ctx)
	u.t.Require().NoError(err, "failed to deploy EventLogger")
	eventLoggerAddress := res.ContractAddress
	u.log.Info("deployed EventLogger", "chainID", tx.ChainID.Value(), "address", eventLoggerAddress)
	return eventLoggerAddress
}

func (u *EOA) DeployWETH() common.Address {
	// Use the e2e bindings which contain the WETH bytecode
	tx := txplan.NewPlannedTx(u.Plan(), txplan.WithData(common.FromHex(e2eBindings.WETHBin)))
	res, err := tx.Included.Eval(u.ctx)
	u.t.Require().NoError(err, "failed to deploy WETH")
	wethAddress := res.ContractAddress
	u.log.Info("deployed WETH", "chainID", tx.ChainID.Value(), "address", wethAddress)
	return wethAddress
}

func (u *EOA) SendInitMessage(trigger *txintent.InitTrigger) (*txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput], *types.Receipt) {
	tx := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](u.Plan())
	tx.Content.Set(trigger)
	receipt, err := tx.PlannedTx.Included.Eval(u.ctx)
	u.t.Require().NoError(err, "init msg receipt not found")
	u.log.Info("init message included", "chain", u.ChainID(), "block", receipt.BlockNumber)
	return tx, receipt
}

func (u *EOA) SendExecMessage(initIntent *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput], eventIdx int) (*txintent.IntentTx[*txintent.ExecTrigger, *txintent.InteropOutput], *types.Receipt) {
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](u.Plan())
	tx.Content.DependOn(&initIntent.Result)
	tx.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &initIntent.Result, eventIdx))
	receipt, err := tx.PlannedTx.Included.Eval(u.ctx)
	u.t.Require().NoError(err, "exec msg receipt not found")
	u.log.Info("exec message included", "chain", u.ChainID(), "block", receipt.BlockNumber)
	// Check single ExecutingMessage triggered
	u.t.Require().Equal(1, len(receipt.Logs))
	return tx, receipt
}

// SendPackedRandomInitMessages batches random messages and initiates them via a single multicall
func (u *EOA) SendPackedRandomInitMessages(rng *rand.Rand, eventLoggerAddress common.Address) (*txintent.IntentTx[*txintent.MultiTrigger, *txintent.InteropOutput], *types.Receipt, error) {
	// Intent to initiate messages
	eventCnt := 1 + rng.Intn(9)
	initCalls := make([]txintent.Call, eventCnt)
	for index := range eventCnt {
		initCalls[index] = interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(100))
	}
	tx := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](u.Plan())
	tx.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: initCalls})
	receipt, err := tx.PlannedTx.Included.Eval(u.ctx)
	if err != nil {
		return nil, nil, err
	}
	return tx, receipt, nil
}

// SendPackedExecMessages batches every message and validates them via a single multicall
func (u *EOA) SendPackedExecMessages(dependOn *txintent.IntentTx[*txintent.MultiTrigger, *txintent.InteropOutput]) (*txintent.IntentTx[*txintent.MultiTrigger, *txintent.InteropOutput], *types.Receipt, error) {
	// Intent to validate message
	tx := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](u.Plan())
	tx.Content.DependOn(&dependOn.Result)
	indexes := []int{}
	result, err := dependOn.Result.Eval(u.ctx)
	if err != nil {
		return nil, nil, err
	}
	for idx := range len(result.Entries) {
		indexes = append(indexes, idx)
	}
	tx.Content.Fn(txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &dependOn.Result, indexes))
	receipt, err := tx.PlannedTx.Included.Eval(u.ctx)
	if err != nil {
		return nil, nil, err
	}
	return tx, receipt, nil
}

// PendingNonce looks up the user nonce in the pending state.
func (u *EOA) PendingNonce() uint64 {
	result, err := retry.Do(u.ctx, 3, retry.Exponential(), func() (uint64, error) {
		return u.el.stackEL().EthClient().PendingNonceAt(u.ctx, u.Address())
	})
	u.t.Require().NoError(err, "must lookup balance")
	return result
}

// WaitForTokenBalance waits for a specific token balance to be reached
func (u *EOA) WaitForTokenBalance(tokenAddr common.Address, expectedBalance eth.ETH) {
	u.t.Require().Eventually(func() bool {
		balance := u.GetTokenBalance(tokenAddr)
		return balance.ToBig().Cmp(expectedBalance.ToBig()) == 0
	}, u.el.stackEL().TransactionTimeout(), time.Second, "awaiting token balance to be updated")
}

// GetTokenBalance returns the token balance for this EOA
func (u *EOA) GetTokenBalance(tokenAddr common.Address) eth.ETH {
	// Use the txintent bindings for contract calls
	tokenContract := txIntentBindings.NewBindings[txIntentBindings.OptimismMintableERC20](
		txIntentBindings.WithTest(u.t),
		txIntentBindings.WithClient(u.el.stackEL().EthClient()),
		txIntentBindings.WithTo(tokenAddr),
	)

	balance, err := contractio.Read(tokenContract.BalanceOf(u.Address()), u.ctx)
	u.t.Require().NoError(err, "must lookup token balance")
	return balance
}

// VerifyTokenBalance verifies the token balance matches expected amount
func (u *EOA) VerifyTokenBalance(tokenAddr common.Address, expectedBalance eth.ETH) {
	actual := u.GetTokenBalance(tokenAddr)
	u.t.Require().Equal(expectedBalance, actual, "must have expected token balance")
}

// ApproveToken approves a spender to spend tokens on behalf of this EOA
func (u *EOA) ApproveToken(tokenAddr common.Address, spender common.Address, amount eth.ETH) {
	tokenContract := txIntentBindings.NewBindings[txIntentBindings.WETH](
		txIntentBindings.WithTest(u.t),
		txIntentBindings.WithClient(u.el.stackEL().EthClient()),
		txIntentBindings.WithTo(tokenAddr),
	)

	approveCall := tokenContract.Approve(spender, amount)
	_, err := contractio.Write(approveCall, u.ctx, u.Plan())
	u.t.Require().NoError(err, "failed to approve token")
}
