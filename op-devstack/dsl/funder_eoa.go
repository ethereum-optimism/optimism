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

const (
	// funderBlockTime is the receipt-poll / resubmit cadence for the funder's
	// reliable EL. It only affects responsiveness, not correctness, so a small
	// fixed value works across chains.
	funderBlockTime = time.Second
	// funderMaxConcurrentTxs matches the load tests: reth caps in-flight txs per
	// account at 16.
	funderMaxConcurrentTxs = 16
)

// FunderEOA is a prefunded account that hands out ETH to test accounts. It
// replaces the former op-faucet service and dsl.Funder: it owns a
// genesis-prefunded key and mints/funds child EOAs from a test wallet.
//
// Funding transactions go through a SyncEOA, whose txinclude.Includer manages
// nonces, so many funding transactions can be issued concurrently — e.g.
// NewFundedEOAs, or several test goroutines sharing one FunderEOA. Concurrency
// safety holds only within a single FunderEOA: do not fund the same account
// concurrently through two FunderEOAs (see AsFunder).
type FunderEOA struct {
	commonImpl
	eoa    *EOA
	inner  *SyncEOA
	wallet *HDWallet
}

// NewFunderEOA wraps a prefunded EOA as a funder that mints child EOAs from wallet.
func NewFunderEOA(eoa *EOA, wallet *HDWallet) *FunderEOA {
	reliableEL := txinclude.NewReliableEL(eoa.el.stackEL().EthClient(), funderBlockTime)
	signer := txinclude.NewPkSigner(eoa.Key().Priv(), eoa.ChainID().ToBig())
	includer := txinclude.NewLimit(
		txinclude.NewPersistent(signer, reliableEL, txinclude.WithStartNonce(eoa.PendingNonce())),
		funderMaxConcurrentTxs,
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

// AsFunder returns a funder for the same account bound to a different EL node on
// the same chain. The returned funder tracks nonces independently, so do not fund
// concurrently through both the original and the returned funder — create one and
// share it.
func (f *FunderEOA) AsFunder(el ELNode) *FunderEOA {
	f.require.Equal(f.ChainID(), el.ChainID(), "funder EL must be on the same chain")
	return NewFunderEOA(f.eoa.AsEL(el), f.wallet)
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
