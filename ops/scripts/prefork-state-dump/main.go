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
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	config "github.com/ethereum-optimism/optimism/op-e2e/config"
)

// predeployNamespaceSize mirrors Predeploys.PREDEPLOY_COUNT: the L2 predeploy
// namespace spans 0x4200…0000 through 0x4200…07FF, and genesis etches a Proxy at
// every slot in it. We scan the full range rather than a name list so the dump
// captures every proxy the era genesis produces — including the bare proxies at
// slots that only become active predeploys in a LATER fork. The fork-era
// predeploys package cannot name those slots, but the frozen state must still
// carry their bare proxies: the state is overlaid per-account onto a
// current-source base genesis, so a captured bare proxy authoritatively resets a
// slot to its era shape instead of inheriting a future implementation.
const predeployNamespaceSize = 2048

var namespaceBase = common.HexToAddress("0x4200000000000000000000000000000000000000")

// predeployScoped keeps every predeploy and preinstall, plus the impls they point
// at via their EIP-1967 slot — with full per-account storage. Everything else
// (e.g. prefunded EOAs) is dropped.
func predeployScoped(full *foundry.ForgeAllocs) *foundry.ForgeAllocs {
	out := &foundry.ForgeAllocs{Accounts: make(types.GenesisAlloc)}

	capture := func(addr common.Address) {
		acct, ok := full.Accounts[addr]
		if !ok {
			return
		}
		if _, seen := out.Accounts[addr]; seen {
			return
		}
		out.Accounts[addr] = acct

		// Follow the EIP-1967 implementation pointer, if set, and keep the impl too.
		// A bare proxy (impl slot zero) or a non-proxied preinstall has none to follow.
		implAddr := common.BytesToAddress(acct.Storage[genesis.ImplementationSlot].Bytes())
		if implAddr == (common.Address{}) {
			return
		}
		if implAcct, ok := full.Accounts[implAddr]; ok {
			out.Accounts[implAddr] = implAcct
		}
	}

	// Full predeploy namespace: captures every proxy (bare or active) and the impls
	// active proxies point at.
	base := new(big.Int).SetBytes(namespaceBase.Bytes())
	for i := int64(0); i < predeployNamespaceSize; i++ {
		capture(common.BigToAddress(new(big.Int).Add(base, big.NewInt(i))))
	}

	// Non-namespace preinstalls (Create2Deployer, Safe, EntryPoint, Permit2, …) live
	// outside the 0x4200 namespace; pick them up from the predeploys registry.
	for _, p := range predeploys.Predeploys {
		capture(p.Address)
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
