// Package state provides the frozen pre-fork L2 predeploy state artifacts that
// the NUT bundle activation test boots from.
//
// It is deliberately a separate package from op-core/nuts: op-node and kona-node
// import op-core/nuts for the embedded bundles, and the state JSON is large
// (hundreds of KB per fork). Keeping the embed here means the state ships only in
// test binaries that import this package, not in the node binaries.
package state

import (
	"embed"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/forks"
)

// State files are named after the fork they represent (the state as of that
// fork): the jovian seed via ops/scripts/gen-seed-state.sh, and later states by
// composing each fork's bundle onto the previous (TestGenerateForkState in
// rust/kona/tests/proofs). From Lagoon onward, <fork>_multichain_state.json is the state of a
// chain with a dependency set of more than one chain. See README.md.
//
//go:embed *_state.json
var stateFS embed.FS

// PreForkState returns the frozen L2 predeploy state a chain has immediately
// before fork activates — i.e. the state as of forks.Prev(fork) — as a set of
// accounts to overlay onto the genesis predeploy set. The activation test requires
// one, so a missing state is an error: either fork has no predecessor, or no
// artifact is committed for that predecessor yet.
func PreForkState(fork forks.Name) (types.GenesisAlloc, error) {
	prev := forks.Prev(fork)
	if prev == forks.None {
		return nil, fmt.Errorf("fork %s has no predecessor fork", fork)
	}
	return loadState(prev, fmt.Sprintf("%s_state.json", prev))
}

// HasMultiChainState reports whether the state as of fork has a multi-chain variant. A chain
// activates the interop contracts at Lagoon only when its dependency set holds more than one
// chain, so the states from Lagoon onward differ by dependency set size.
func HasMultiChainState(fork forks.Name) bool {
	return slices.Contains(forks.From(forks.Lagoon), fork)
}

// PreForkMultiChainState returns the frozen L2 predeploy state immediately before fork activates
// on a chain with a dependency set of more than one chain. Before Lagoon, the dependency set has
// no effect, so it returns the same state as PreForkState.
func PreForkMultiChainState(fork forks.Name) (types.GenesisAlloc, error) {
	prev := forks.Prev(fork)
	if !HasMultiChainState(prev) {
		return PreForkState(fork)
	}
	return loadState(prev, fmt.Sprintf("%s_multichain_state.json", prev))
}

// loadState loads the committed state file name, which holds the state as of fork.
func loadState(fork forks.Name, name string) (types.GenesisAlloc, error) {
	data, err := stateFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("no committed pre-fork state for %s (%s): %w", fork, name, err)
	}
	var alloc types.GenesisAlloc
	if err := json.Unmarshal(data, &alloc); err != nil {
		return nil, fmt.Errorf("parsing pre-fork state %s: %w", name, err)
	}
	return alloc, nil
}
