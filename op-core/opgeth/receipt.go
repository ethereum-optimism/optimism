package opgeth

import (
	"github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// FromGethReceipt wraps an op-geth receipt, copying its deposit fields to the
// authoritative outer fields of the wrapper. The receipt struct is shallow-copied
// (reference fields like Logs still alias the source).
//
// Wrapping by hand, as &Receipt{Receipt: *r}, leaves the outer deposit fields nil: a
// deposit receipt wrapped that way encodes without them and derives no deposit
// contract address.
func FromGethReceipt(r *types.Receipt) *optypes.Receipt {
	wrapped := &optypes.Receipt{Receipt: *r}
	wrapped.DepositNonce = r.DepositNonce
	wrapped.DepositReceiptVersion = r.DepositReceiptVersion
	return wrapped
}

// FromGethReceipts wraps op-geth receipts, per FromGethReceipt.
func FromGethReceipts(rs types.Receipts) optypes.Receipts {
	out := make(optypes.Receipts, len(rs))
	for i, r := range rs {
		out[i] = FromGethReceipt(r)
	}
	return out
}
