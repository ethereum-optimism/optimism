#!/usr/bin/env bash
#
# Regenerates op-core/nuts/state/jovian_state.json — the frozen L2 predeploy
# state as of the jovian fork. It's the seed of the pre-fork state chain: the
# karst NUT bundle activation test boots from it (jovian is karst's predecessor).
# State files are named after the fork they represent, not the bundle that
# consumes them.
#
# Why this is a script and not the `nut-prestate-gen` tool:
# karst's predecessor is jovian, which predates the NUT lock system, so the seed
# must be built from jovian-era contracts. The CURRENT op-deployer cannot consume
# jovian-era contracts (ABI drift — e.g. DeployImplementations dropped its
# `protocolVersionsProxy` input field since jovian). So we generate the seed with
# jovian's OWN toolchain, run inside a worktree checked out at the jovian commit,
# where op-deployer and the contracts are mutually consistent.
#
# CAVEAT (non-determinism): jovian's op-deployer randomizes the CREATE2 salt, so
# re-running this produces cosmetically different L1-derived slots (the L1
# counterpart addresses embedded in L2CrossDomainMessenger / L2StandardBridge /
# L2ERC721Bridge). The committed seed is one specific instance; the activation
# test is expected to reset those L1-derived slots on entry. Do not expect this
# script to reproduce the committed file byte-for-byte.
#
# Usage: ops/scripts/gen-jovian-prestate-seed.sh
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"

# Jovian-era monorepo commit. Jovian mainnet activation was 2025-12-02
# (L1 timestamp 1764691201); this commit is jovian-era source, before karst's
# contract changes.
COMMIT="79cee4ec028db485150db71e64d0921a78960f70"
OUT="$ROOT/op-core/nuts/state/jovian_state.json"
WT="$(mktemp -d "${TMPDIR:-/tmp}/jovian-seed-wt.XXXXXX")"

cleanup() {
  git -C "$ROOT" worktree remove --force "$WT" 2>/dev/null || true
  rm -rf "$WT" 2>/dev/null || true
}
trap cleanup EXIT

echo ">>> [1/3] worktree at $COMMIT"
git -C "$ROOT" worktree add --detach "$WT" "$COMMIT"

echo ">>> [2/3] build jovian-era forge-artifacts"
( cd "$WT/packages/contracts-bedrock" && just build-no-tests )

echo ">>> [3/3] dump predeploy-scoped jovian state with jovian's toolchain"
mkdir -p "$WT/ops/scripts/jovian-prestate-dump"
cat > "$WT/ops/scripts/jovian-prestate-dump/main.go" <<'GO'
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

// predeployScoped keeps only predeploy proxies + the implementations they point
// at via their EIP-1967 implementation slot, preserving full per-account storage.
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
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: jovian-prestate-dump <out.json>")
		os.Exit(1)
	}
	// The jovian fork level is the pre-karst predeploy state.
	allocs := config.L2Allocs(config.DefaultAllocType, genesis.L2AllocsJovian)
	scoped := predeployScoped(allocs)
	data, err := json.MarshalIndent(scoped, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(os.Args[1], data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s (%d accounts)\n", os.Args[1], len(scoped.Accounts))
}
GO

mkdir -p "$ROOT/op-core/nuts/state"
( cd "$WT" && go run ./ops/scripts/jovian-prestate-dump "$OUT" )

echo ">>> done"
ls -lh "$OUT"
