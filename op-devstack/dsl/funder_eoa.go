package dsl

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// FunderEOA is a funded account that hands out ETH to test accounts.
type FunderEOA struct {
	commonImpl
	eoa    *EOA
	inner  *SyncEOA
	wallet *HDWallet
}

// NewFunderEOA wraps a prefunded EOA as a funder that mints child EOAs from wallet.
func NewFunderEOA(eoa *EOA, wallet *HDWallet) *FunderEOA {
	// We are safe to estimate blockTime. Even if it is actually higher or lower it will
	// merely result in slightly less optimal resubmission strategies in the includer.
	const blockTime = 2 * time.Second
	el := txinclude.NewReliableEL(eoa.el.stackEL().EthClient(), blockTime)
	signer := txinclude.NewPkSigner(eoa.Key().Priv(), eoa.ChainID().ToBig())
	includer := txinclude.NewLimit(
		txinclude.NewPersistent(signer, el, txinclude.WithStartNonce(eoa.PendingNonce())),
		16, // reth caps in-flight txs per account at 16.
	)
	return &FunderEOA{
		commonImpl: commonFromT(eoa.t),
		eoa:        eoa,
		inner:      NewSyncEOA(includer, eoa.Plan()),
		wallet:     wallet,
	}
}

func (f *FunderEOA) String() string {
	return fmt.Sprintf("FunderEOA(%s @ %s)", f.eoa.Address(), f.ChainID())
}

func (f *FunderEOA) Address() common.Address {
	return f.eoa.Address()
}

func (f *FunderEOA) ChainID() eth.ChainID {
	return f.eoa.ChainID()
}

// AsFunder returns a view of the same funder bound to a different EL node on the
// same chain. All views share the account's includer and nonce state, so they are
// safe to use concurrently. Funded EOAs are created and observed through el;
// funding transactions continue to use the original funder's EL.
func (f *FunderEOA) AsFunder(el ELNode) *FunderEOA {
	f.require.Equal(f.ChainID(), el.ChainID(), "funder EL must be on the same chain")
	view := *f
	view.eoa = f.eoa.AsEL(el)
	return &view
}

// fund sends amount to addr, waiting for the funding tx to be included. It is safe
// to call concurrently. Zero amounts are a no-op.
func (f *FunderEOA) fund(addr common.Address, amount eth.ETH) {
	if amount.IsZero() {
		return
	}
	res, err := f.inner.Include(f.t, txplan.WithTo(&addr), txplan.WithValue(amount))
	f.require.NoError(err, "must fund %s with %s", addr, amount)
	f.require.Equal(types.ReceiptStatusSuccessful, res.Receipt.Status, "funding tx to %s reverted", addr)
}

// NewFundedEOA mints a fresh child EOA on the funder's chain and funds it with at
// least the given amount. A zero amount yields a fresh, unfunded EOA.
func (f *FunderEOA) NewFundedEOA(amount eth.ETH) *EOA {
	eoa := f.wallet.NewEOA(f.eoa.el)
	f.FundAtLeast(eoa, amount)
	return eoa
}

// NewFundedEOAs mints and funds count fresh child EOAs concurrently.
func (f *FunderEOA) NewFundedEOAs(count int, amount eth.ETH) []*EOA {
	eoas := make([]*EOA, count)
	var wg sync.WaitGroup
	for i := range eoas {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eoas[i] = f.NewFundedEOA(amount)
		}()
	}
	wg.Wait()
	for _, eoa := range eoas {
		// Surface funding failures early rather than as confusing nil derefs later.
		f.require.NotNil(eoa)
	}
	return eoas
}

// Fund transfers amount to the target EOA and waits until its balance reflects it.
func (f *FunderEOA) Fund(to *EOA, amount eth.ETH) eth.ETH {
	current := to.balance()
	f.fund(to.Address(), amount)
	final := current.Add(amount)
	to.WaitForBalance(final)
	return final
}

// FundAtLeast tops the target EOA up so that its balance is at least amount.
func (f *FunderEOA) FundAtLeast(to *EOA, amount eth.ETH) eth.ETH {
	current := to.balance()
	if current.Lt(amount) {
		f.fund(to.Address(), amount.Sub(current))
		to.WaitForBalance(amount)
		return amount
	}
	return current
}
