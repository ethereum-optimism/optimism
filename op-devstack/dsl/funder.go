package dsl

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Funder struct {
	commonImpl
	wallet *HDWallet
	faucet *Faucet
	el     ELNode
}

func NewFunder(w *HDWallet, f *Faucet, el ELNode) *Funder {
	f.t.Require().Equal(f.inner.ID().ChainID(), el.ChainID(), "faucet and EL must be on same chain")
	return &Funder{
		commonImpl: commonFromT(w.t),
		wallet:     w,
		faucet:     f,
		el:         el,
	}
}

func (f *Funder) NewFundedEOA(amount eth.ETH) *EOA {
	eoa := f.wallet.NewEOA(f.el)
	f.faucet.Fund(eoa.Address(), amount)
	return eoa
}

func (f *Funder) NewFundedEOAs(count int, amount eth.ETH) []*EOA {
	eoas := func() []*EOA {
		eoas := make([]*EOA, count)
		var wg sync.WaitGroup
		defer wg.Wait()
		for idx := range len(eoas) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				eoas[idx] = f.NewFundedEOA(amount)
			}()
		}
		return eoas
	}()
	for _, eoa := range eoas {
		// For a large count, the faucet may fail.
		// This sanity check prevents surprising errors down the line.
		f.require.NotNil(eoa)
	}
	return eoas
}

func (f *Funder) Fund(wallet *EOA, amount eth.ETH) eth.ETH {
	currentBalance := wallet.balance()
	f.faucet.Fund(wallet.Address(), amount)
	finalBalance := currentBalance.Add(amount)
	wallet.VerifyBalanceExact(finalBalance)
	return finalBalance
}

func (f *Funder) FundAtLeast(wallet *EOA, amount eth.ETH) eth.ETH {
	currentBalance := wallet.balance()
	if currentBalance.Lt(amount) {
		missing := amount.Sub(currentBalance)
		f.faucet.Fund(wallet.Address(), missing)
		currentBalance = currentBalance.Add(missing)
		wallet.WaitForBalance(currentBalance)
	}
	return currentBalance
}
