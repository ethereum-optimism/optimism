// Command prefork-state-dump writes the predeploy-scoped L2 state for a given fork
// level to a JSON file.
//
// It is invoked by ops/scripts/gen-seed-state.sh, which copies this file into a git
// worktree checked out at the fork's commit and runs it there. Running inside the
// worktree is the whole point: it pairs the fork-era contracts with the fork-era
// op-deployer (via op-e2e/config), avoiding the ABI drift that breaks the current
// op-deployer against older contracts. For that reason it depends only on the
// long-stable public op-e2e/config.L2Allocs API, which exists across these
// commits — do not add dependencies on newer helpers.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	config "github.com/ethereum-optimism/optimism/op-e2e/config"
)

// predeployScoped keeps every predeploy and preinstall, plus the
// impls they point at via their EIP-1967 slot — with full per-account storage.
// Everything else (e.g. prefunded EOAs) is dropped.
func predeployScoped(full *foundry.ForgeAllocs) *foundry.ForgeAllocs {
	out := &foundry.ForgeAllocs{Accounts: make(types.GenesisAlloc)}
	for _, p := range predeploys.Predeploys {
		acct, ok := full.Accounts[p.Address]
		if !ok {
			continue
		}
		out.Accounts[p.Address] = acct
		if p.ProxyDisabled {
			continue
		}
		implAddr := common.BytesToAddress(acct.Storage[genesis.ImplementationSlot].Bytes())
		if implAddr == (common.Address{}) {
			continue
		}
		if implAcct, ok := full.Accounts[implAddr]; ok {
			out.Accounts[implAddr] = implAcct
		}
	}
	return out
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: prefork-state-dump <fork> <out.json>")
		os.Exit(1)
	}
	fork, outPath := os.Args[1], os.Args[2]

	// The <fork> alloc mode is the predeploy state as of that fork.
	allocs := config.L2Allocs(config.DefaultAllocType, genesis.L2AllocsMode(fork))
	scoped := predeployScoped(allocs)

	data, err := json.MarshalIndent(scoped, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(outPath, data, 0o640); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s (%d accounts)\n", outPath, len(scoped.Accounts))
}
