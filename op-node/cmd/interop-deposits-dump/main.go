// Command interop-deposits-dump prints Interop activation deposits and upgrade gas; paired with the kona-hardforks `interop-deposits-dump` example for cross-language diffing.
package main

import (
	"fmt"
	"math/big"
	"os"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
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
			dep, err := optypes.UnmarshalDepositTx([]byte(raw))
			if err != nil {
				fmt.Fprintf(os.Stderr, "go: tx %d is not a deposit tx: %v\n", i, err)
				os.Exit(1)
			}
			to := "create"
			if dep.To != nil {
				to = fmt.Sprintf("0x%x", dep.To.Bytes())
			}
			fmt.Printf("--- tx %d ---\n", i)
			fmt.Printf("source_hash=0x%x\n", dep.SourceHash.Bytes())
			fmt.Printf("from=0x%x\n", dep.From.Bytes())
			fmt.Printf("to=%s\n", to)
			fmt.Printf("mint=0x%032x\n", bigOrZero(dep.Mint))
			fmt.Printf("value=0x%064x\n", bigOrZero(dep.Value))
			fmt.Printf("gas_limit=0x%016x\n", dep.Gas)
			fmt.Printf("is_system_tx=%v\n", dep.IsSystemTransaction)
			fmt.Printf("data=0x%x\n", dep.Data)
		}
	}
}

func bigOrZero(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return v
}
