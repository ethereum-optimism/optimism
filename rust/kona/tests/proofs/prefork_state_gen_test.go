package proofs

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
)

// genPreForkStateEnv names the fork whose pre-fork state artifact to (re)generate,
// e.g. OP_E2E_GEN_PREFORK_STATE=karst. It gates TestGenerateForkState, and the engine
// records trie-key preimages whenever it is set (see
// op-e2e/actions/helpers/l2_engine.go) so the post-activation state can be
// enumerated and dumped.
//
// Generate one fork at a time, committing each before the next: the loader embeds
// states at COMPILE time, so <prev>_state.json must be on disk (and the test
// binary rebuilt) before generating <fork>_state.json.
const genPreForkStateEnv = "OP_E2E_GEN_PREFORK_STATE"

// TestGenerateForkState (re)generates op-core/nuts/state/<fork>_state.json for the
// fork named by OP_E2E_GEN_PREFORK_STATE, and is skipped otherwise (so it never runs in
// normal CI). It reuses the exact activation flow via activateFork — the
// post-activation state IS the artifact (compose: <fork>_state = <prev>_state +
// the frozen <fork> bundle) — so generation stays in lockstep with the validation
// test rather than duplicating setup.
func TestGenerateForkState(gt *testing.T) {
	target := os.Getenv(genPreForkStateEnv)
	if target == "" {
		gt.Skipf("set %s=<fork> to (re)generate <fork>_state.json", genPreForkStateEnv)
	}

	fork := forks.Name(target)
	preFork := forks.Prev(fork)
	require.NotEqualf(gt, forks.None, preFork, "fork %s has no preceding fork", fork)
	preHelper := lookupHardforkHelper(preFork)
	require.NotNilf(gt, preHelper, "no pre-fork helper registered for %s (prior fork %s)", fork, preFork)

	matrix := helpers.NewMatrix[forks.Name]()
	matrix.AddDefaultTestCasesWithName(string(fork), fork, helpers.NewForkMatrix(preHelper), generateForkState)
	matrix.Run(gt)
}

func generateForkState(gt *testing.T, testCfg *helpers.TestCfg[forks.Name]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env, actHeader := activateFork(t, testCfg)
	writePreForkState(t, env, testCfg.Custom, actHeader)
}

// writePreForkState dumps the post-activation predeploy-scoped state to
// op-core/nuts/state/<fork>_state.json. It requires the engine to have recorded
// trie-key preimages (enabled by genPreForkStateEnv) so the state can be enumerated.
func writePreForkState(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, fork forks.Name, actHeader *types.Header) {
	stateDB, err := env.Engine.L2Chain().StateAt(actHeader.Root)
	require.NoError(t, err, "open state at %s activation block", fork)
	var full foundry.ForgeAllocs
	full.FromState(stateDB)
	scoped := predeployScoped(&full)
	require.NotEmptyf(t, scoped.Accounts,
		"dumped state is empty — are trie-key preimages enabled? (set %s)", genPreForkStateEnv)
	canonicalizeL1Block(scoped)

	root, err := opservice.FindMonorepoRoot(".")
	require.NoError(t, err)
	outPath := filepath.Join(root, "op-core", "nuts", "state", string(fork)+"_state.json")
	data, err := json.MarshalIndent(scoped, "", "  ")
	require.NoError(t, err)
	data = append(data, '\n')
	require.NoError(t, os.WriteFile(outPath, data, 0o640))
	fmt.Printf("wrote %s (%d accounts)\n", outPath, len(scoped.Accounts))
}

// canonicalizeL1Block drops L1Block's L1-attribute storage — integer slots 0..8
// (number, timestamp, basefee, hash, …) per snapshots/storageLayout/L1Block.json.
// Those are written by the per-block L1-info deposit, so they're run-specific
// (timestamp, L1 hash) and get re-set when the state is consumed; zeroing them
// keeps the artifact byte-deterministic without affecting behavior. The EIP-1967
// impl/admin slots and the isFeatureEnabled mapping (keccak-keyed, not slot 9
// literally) are left intact.
func canonicalizeL1Block(allocs *foundry.ForgeAllocs) {
	l1b, ok := allocs.Accounts[predeploys.L1BlockAddr]
	if !ok {
		return
	}
	for slot := range l1b.Storage {
		if slot.Big().Cmp(big.NewInt(9)) < 0 { // L1-attribute slots 0..8
			delete(l1b.Storage, slot)
		}
	}
}

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
