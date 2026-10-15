package op_reth_opgeth_peering

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const (
	// fanOut is how many transactions one account can hold in a reth pool at once (reth's
	// --txpool.max-account-slots default), so how many accounts one account funds per round.
	fanOut = 16
	// feeCapGwei and tipGwei are pinned on every transaction the tests sign offline. The fee cap
	// leaves room for the base fee to climb through the full blocks a burst produces.
	feeCapGwei = 100
	tipGwei    = 1
)

var (
	feeCap = big.NewInt(feeCapGwei * params.GWei)
	tip    = big.NewInt(tipGwei * params.GWei)
	// fundingTxCost is the most one funding transfer can cost its sender.
	fundingTxCost = eth.GWei(feeCapGwei).Mul(params.TxGas)
)

// fundAccounts returns count fresh accounts holding at least amount each, bound to the
// sequencer's EL, and the number of the last block that funded one of them. The funder pays up
// to fanOut distributors in one round, each distributor pays up to fanOut more in the next, and
// so on until a round pays the count accounts, so no account ever has more than fanOut
// transactions in a pool at once. Every round is one batch of offline signed transfers to the
// sequencer, waited on through each payer's last transfer.
func fundAccounts(t devtest.T, sys *pairSystem, count int, amount eth.ETH) ([]*dsl.EOA, uint64) {
	levels := fundingLevels(count)
	amounts := fundingAmounts(levels, amount)
	t.Logger().Info("Funding accounts through distributors", "count", count, "levels", levels, "amounts", amounts)
	accounts := sys.FunderL2.NewFundedEOAs(levels[0], amounts[0])
	fundedAt := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number
	for level := 1; level < len(levels); level++ {
		accounts, fundedAt = fundRound(t, sys, accounts, levels[level], amounts[level])
	}
	return accounts, fundedAt
}

// fundingLevels returns how many accounts each funding round creates, the last being count, with
// every earlier round creating enough payers for the next.
func fundingLevels(count int) []int {
	levels := []int{count}
	for levels[0] > fanOut {
		payers := (levels[0] + fanOut - 1) / fanOut
		levels = append([]int{payers}, levels...)
	}
	return levels
}

// fundingAmounts returns what an account of each round is funded with: the last round's accounts
// get amount, and a payer gets what it pays out plus the cost of paying it.
func fundingAmounts(levels []int, amount eth.ETH) []eth.ETH {
	amounts := make([]eth.ETH, len(levels))
	amounts[len(levels)-1] = amount
	for level := len(levels) - 2; level >= 0; level-- {
		amounts[level] = amounts[level+1].Add(fundingTxCost).Mul(fanOut)
	}
	return amounts
}

// fundRound has each payer fund up to fanOut fresh accounts with amount, in one batch of offline
// signed transfers, and returns the funded accounts once every payer's last transfer is included,
// with the number of the last block that included one.
func fundRound(t devtest.T, sys *pairSystem, payers []*dsl.EOA, count int, amount eth.ETH) ([]*dsl.EOA, uint64) {
	t.Require().GreaterOrEqual(len(payers)*fanOut, count, "not enough payers for the round")
	accounts := make([]*dsl.EOA, 0, count)
	transfers := make([]*types.Transaction, 0, count)
	lastTransfers := make([]*types.Transaction, 0, len(payers))
	for i, payer := range payers {
		var last *types.Transaction
		for nonce := uint64(0); nonce < fanOut && len(accounts) < count; nonce++ {
			account := sys.Wallet.NewEOA(sys.Sequencer.EL)
			to := account.Address()
			last = payer.Sign(
				payer.PlanOffline(),
				txplan.WithStaticNonce(nonce),
				txplan.WithTo(&to),
				txplan.WithValue(amount),
				txplan.WithGasLimit(params.TxGas),
				txplan.WithGasFeeCap(feeCap),
				txplan.WithGasTipCap(tip),
			)
			accounts = append(accounts, account)
			transfers = append(transfers, last)
		}
		t.Require().NotNil(last, "payer %d of %d had nothing to pay", i, len(payers))
		lastTransfers = append(lastTransfers, last)
	}
	sys.Sequencer.EL.SendTransactions(transfers)
	var fundedAt uint64
	for _, last := range lastTransfers {
		// A payer's earlier transfers are included no later than its last one.
		fundedAt = max(fundedAt, sys.Sequencer.EL.WaitForReceipt(last.Hash()).BlockNumber.Uint64())
	}
	t.Logger().Info("Funded a round of accounts", "count", len(accounts), "payers", len(payers), "amount", amount, "funded_at", fundedAt)
	return accounts, fundedAt
}
