// Command interop-deposits-dump prints Interop activation deposits and upgrade gas; paired with the kona-hardforks `interop-deposits-dump` example for cross-language diffing.
package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/core/types"
)

func main() {
	for _, activate := range []bool{false, true} {
		txs, gas, err := derive.InteropActivationUpgradeTransactions(activate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "go: error for activate=%v: %v\n", activate, err)
			os.Exit(1)
		}
		fmt.Printf("activate=%v\n", activate)
		fmt.Printf("gas=0x%016x\n", gas)
		for i, raw := range txs {
			var tx types.Transaction
			if err := tx.UnmarshalBinary([]byte(raw)); err != nil {
				fmt.Fprintf(os.Stderr, "go: tx %d unmarshal: %v\n", i, err)
				os.Exit(1)
			}
			if !tx.IsDepositTx() {
				fmt.Fprintf(os.Stderr, "go: tx %d is not a deposit tx (type=%d)\n", i, tx.Type())
				os.Exit(1)
			}
			to := "create"
			if t := tx.To(); t != nil {
				to = fmt.Sprintf("0x%x", t.Bytes())
			}
			fmt.Printf("--- tx %d ---\n", i)
			fmt.Printf("source_hash=0x%x\n", tx.SourceHash().Bytes())
			fmt.Printf("from=0x%x\n", tx.From().Bytes())
			fmt.Printf("to=%s\n", to)
			fmt.Printf("mint=0x%032x\n", bigOrZero(tx.Mint()))
			fmt.Printf("value=0x%064x\n", bigOrZero(tx.Value()))
			fmt.Printf("gas_limit=0x%016x\n", tx.Gas())
			fmt.Printf("is_system_tx=%v\n", tx.IsSystemTx())
			fmt.Printf("data=0x%x\n", tx.Data())
		}
	}
}

func bigOrZero(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return v
}
