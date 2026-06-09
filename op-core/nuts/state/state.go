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

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/forks"
)

// State files are named after the fork they represent (the state as of that
// fork). They are produced by ops/scripts/gen-jovian-prestate-seed.sh (the jovian
// seed) and, in future, by composing later bundles onto it. See README.md.
//
//go:embed *_state.json
var stateFS embed.FS

// PreForkState returns the frozen L2 predeploy state a chain has immediately
// before fork activates — i.e. the state as of forks.Prev(fork) — as a set of
// accounts to overlay onto the genesis predeploy set. ok is false when no state
// artifact is committed for that predecessor fork yet (callers then fall back to
// building genesis from current source).
func PreForkState(fork forks.Name) (alloc types.GenesisAlloc, ok bool, err error) {
	prev := forks.Prev(fork)
	if prev == forks.None {
		return nil, false, nil
	}
	name := fmt.Sprintf("%s_state.json", prev)
	data, err := stateFS.ReadFile(name)
	if err != nil {
		// No committed state for this predecessor fork.
		return nil, false, nil
	}
	if err := json.Unmarshal(data, &alloc); err != nil {
		return nil, true, fmt.Errorf("parsing pre-fork state %s: %w", name, err)
	}
	return alloc, true, nil
}
