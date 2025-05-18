package dsl

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// EOA is an Externally-Owned-Account:
// an account on a specific chain that is operated by a single private key.
type EOA struct {
	commonImpl

	key *Key

	// el is the execution-layer node that this user operates against.
	// This may be a L1 or L2 EL node.
	el ELNode

	// Set to true when a cleanup has been registered
	cleanupRegistered bool
}

func NewEOA(key *Key, el ELNode) *EOA {
	eoa := &EOA{
		commonImpl: commonFromT(key.t),
		el:         el,
		key:        key,
	}

	// If this address is in the fundedEOARegistry, register a cleanup
	if _, funded := fundedEOARegistry[eoa.Address()]; funded && !eoa.cleanupRegistered {
		eoa.registerCleanup()
	}

	return eoa
}

// registerCleanup sets up a cleanup function to refund any unused ETH at the end of the test
func (u *EOA) registerCleanup() {
	if u.cleanupRegistered {
		return
	}

	u.cleanupRegistered = true

	// Register cleanup to refund unused ETH
	u.t.Cleanup(func() {
		// Get the current balance
		balance := u.balance()
		if balance.IsZero() {
			return
		}

		// For now, use a hardcoded zero address as a refund address
		// In a real implementation, you would retrieve the faucet address
		refundAddress := common.HexToAddress("0x0000000000000000000000000000000000000000")

		// Log that we're refunding
		u.log.Info("Refunding unused ETH", "address", u.Address(), "amount", balance, "refund_to", refundAddress)

		// Leave a small amount for gas fees (0.01 ETH)
		gasReserve := eth.Ether(0).Add(eth.WeiU64(10_000_000_000_000_000)) // 0.01 ETH

		// Only refund if we have more than the gas reserve
		if !balance.Gt(gasReserve) {
			u.log.Info("Balance too low to refund", "address", u.Address(), "balance", balance, "min_required", gasReserve)
			return
		}

		// Calculate amount to refund (balance - gas reserve)
		refundAmount := balance.Sub(gasReserve)

		// Transfer the funds
		tx := u.Transfer(refundAddress, refundAmount)

		// Check for errors using the Success field and eval which returns an error
		_, err := tx.Success.Eval(u.ctx)
		if err != nil {
			u.log.Warn("Failed to refund ETH", "address", u.Address(), "amount", refundAmount, "err", err)
		} else {
			u.log.Info("Successfully refunded ETH", "address", u.Address(), "amount", refundAmount, "refund_to", refundAddress)
		}
	})
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
		txplan.WithTransactionSubmitter(elClient),
		txplan.WithRetryInclusion(elClient, 5, retry.Exponential()),
		txplan.WithBlockInclusionInfo(elClient),
	)
}

// PlanTransfer creates the tx-plan options to perform a transfer
// of the given amount of ETH to the given account.
func (u *EOA) PlanTransfer(to common.Address, amount eth.ETH) txplan.Option {
	return txplan.Combine(
		u.Plan(),
		txplan.WithTo(&to),
		txplan.WithValue(amount.ToBig()),
		txplan.WithGasLimit(params.TxGas),
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

func (u *EOA) DeployEventLogger() common.Address {
	tx := txplan.NewPlannedTx(u.Plan(), txplan.WithData(common.FromHex(bindings.EventloggerBin)))
	res, err := tx.Included.Eval(u.ctx)
	u.t.Require().NoError(err, "failed to deploy EventLogger")
	eventLoggerAddress := res.ContractAddress
	u.log.Info("deployed EventLogger", "chainID", tx.ChainID.Value(), "address", eventLoggerAddress)
	return eventLoggerAddress
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
