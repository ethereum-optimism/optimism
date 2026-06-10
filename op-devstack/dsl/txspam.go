package dsl

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txspam"
)

// FundableEL is the subset of the L1/L2 EL DSL needed to fund and spam EOAs. Both *L1ELNode and
// *L2ELNode satisfy it, so FundEOAs works against either layer.
type FundableEL interface {
	ELNode
	EthClient() apis.EthClient
}

// FundEOAs funds numAccounts spammer EOAs and returns a txspam.SyncEOA for each.
//
// The devstack funder isn't reliable when funding many distinct accounts directly, so we fund a
// single intermediate account from the faucet and let it fund all the others concurrently. budget
// is the total to distribute (a small portion is held back to pay for the funding transactions
// themselves).
func FundEOAs(t devtest.T, budget eth.ETH, numAccounts uint64, blockTime time.Duration, el FundableEL, wallet *HDWallet, faucet *Faucet) []*txspam.SyncEOA {
	funderEOA := NewFunder(wallet, faucet, el).NewFundedEOA(budget)
	funder := txspam.NewSyncEOAFromPrivKey(funderEOA.Key().Priv(), el.ChainID(), blockTime, el.EthClient())
	budget = budget.Sub(budget.Div(50)) // Reserve 2% of the balance for gas.
	eoas, err := funder.FundEOAs(t.Ctx(), el.EthClient(), el.ChainID(), budget.Div(numAccounts), numAccounts, blockTime)
	t.Require().NoError(err)
	return eoas
}
