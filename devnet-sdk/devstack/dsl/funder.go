package dsl

import "github.com/ethereum-optimism/optimism/op-service/eth"

type Funder struct {
	commonImpl
	wallet *HDWallet
	faucet *Faucet
	el     ELNode
}

func NewFunder(w *HDWallet, f *Faucet, el ELNode) *Funder {
	f.t.Require().Equal(f.inner.ID().ChainID, el.ChainID(), "faucet and EL must be on same chain")
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
