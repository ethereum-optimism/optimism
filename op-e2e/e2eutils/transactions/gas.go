package transactions

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

// TxBuilder creates and sends a transaction using the supplied bind.TransactOpts.
// Returns the created transaction and any error reported.
type TxBuilder func(opts *bind.TransactOpts) (*types.Transaction, error)

// PadGasEstimate multiplies the gas estimate for a transaction by the specified paddingFactor before sending the
// actual transaction. Useful for cases where the gas required is variable.
// The builder will be invoked twice, first with NoSend=true to estimate the gas and the second time with
// NoSend=false and GasLimit including the requested padding.
func PadGasEstimate(opts *bind.TransactOpts, paddingFactor float64, builder TxBuilder) (*types.Transaction, error) {
	// Take a copy of the opts to avoid mutating the original
	oCopy := *opts
	o := &oCopy
	o.NoSend = true
	tx, err := builder(o)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %w", err)
	}
	gas := float64(tx.Gas()) * paddingFactor
	o.GasLimit = uint64(gas)
	o.NoSend = false
	return builder(o)
}
