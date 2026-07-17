package pipeline

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// BuildInteropDepSet converts the intent chain list into an op-core static dependency set.
func BuildInteropDepSet(chains []*state.ChainIntent) (*depset.StaticConfigDependencySet, error) {
	return buildInteropDepSet(chains, depset.NewStaticConfigDependencySet)
}

func buildInteropDepSet(
	chains []*state.ChainIntent,
	newDepSet func(map[eth.ChainID]*depset.StaticConfigDependency) (*depset.StaticConfigDependencySet, error),
) (*depset.StaticConfigDependencySet, error) {
	deps := make(map[eth.ChainID]*depset.StaticConfigDependency)
	for _, chain := range chains {
		id := eth.ChainIDFromBytes32(chain.ID)
		deps[id] = &depset.StaticConfigDependency{}
	}
	return newDepSet(deps)
}

// ValidateInteropDepSetMatchesIntent verifies that the current intent contains
// exactly the chain IDs that were recorded by prepare.
func ValidateInteropDepSetMatchesIntent(chains []*state.ChainIntent, prepared *depset.StaticConfigDependencySet) error {
	if prepared == nil {
		return fmt.Errorf("prepared interop dependency set is missing; rerun op-deployer prepare")
	}

	seen := make(map[common.Hash]struct{}, len(chains))
	duplicateSet := make(map[common.Hash]struct{})
	for _, chain := range chains {
		if _, ok := seen[chain.ID]; ok {
			duplicateSet[chain.ID] = struct{}{}
			continue
		}
		seen[chain.ID] = struct{}{}
	}
	if len(duplicateSet) > 0 {
		duplicates := make([]common.Hash, 0, len(duplicateSet))
		for id := range duplicateSet {
			duplicates = append(duplicates, id)
		}
		sortChainIDs(duplicates)
		return fmt.Errorf("intent contains duplicate chain IDs %s; rerun op-deployer prepare", formatChainIDs(duplicates))
	}

	intentIDs := make(map[common.Hash]struct{}, len(chains))
	for _, chain := range chains {
		intentIDs[chain.ID] = struct{}{}
	}
	preparedIDs := make(map[common.Hash]struct{}, len(prepared.Chains()))
	for _, id := range prepared.Chains() {
		preparedIDs[common.Hash(id.Bytes32())] = struct{}{}
	}

	added := make([]common.Hash, 0)
	for id := range intentIDs {
		if _, ok := preparedIDs[id]; !ok {
			added = append(added, id)
		}
	}
	removed := make([]common.Hash, 0)
	for id := range preparedIDs {
		if _, ok := intentIDs[id]; !ok {
			removed = append(removed, id)
		}
	}
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}

	sortChainIDs(added)
	sortChainIDs(removed)
	return fmt.Errorf(
		"intent chain set does not match prepared chain set: added chain IDs %s; removed chain IDs %s; rerun op-deployer prepare",
		formatChainIDs(added),
		formatChainIDs(removed),
	)
}

func sortChainIDs(ids []common.Hash) {
	sort.Slice(ids, func(i, j int) bool {
		return strings.Compare(ids[i].Hex(), ids[j].Hex()) < 0
	})
}

func formatChainIDs(ids []common.Hash) string {
	formatted := make([]string, len(ids))
	for i, id := range ids {
		formatted[i] = id.Hex()
	}
	return "[" + strings.Join(formatted, ", ") + "]"
}

func GenerateInteropDepset(_ context.Context, pEnv *Env, globalIntent *state.Intent, st *state.State) error {
	lgr := pEnv.Logger.New("stage", "generate-interop-depset")

	lgr.Info("Creating interop dependency set...")
	interopDepSet, err := BuildInteropDepSet(globalIntent.Chains)
	if err != nil {
		return fmt.Errorf("failed to create interop dependency set: %w", err)
	}
	st.InteropDepSet = interopDepSet

	if err := pEnv.StateWriter.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
