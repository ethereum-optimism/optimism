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
	env, actHeader := activateFork(t, testCfg, false)
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

// predeployNamespaceSize mirrors Predeploys.PREDEPLOY_COUNT: the L2 predeploy
// namespace spans 0x4200…0000 through 0x4200…07FF, and genesis etches a Proxy at
// every slot in it. We scan the full range rather than a name list so the dump
// captures every proxy — including the bare proxies at slots that only become
// active predeploys in a LATER fork. The pre-fork state is overlaid per-account
// onto a current-source base genesis, so a captured bare proxy authoritatively
// resets that slot to its era shape instead of inheriting a future implementation
// (which would trip the L2ContractsManager downgrade guard on activation).
const predeployNamespaceSize = 2048

var predeployNamespaceBase = common.HexToAddress("0x4200000000000000000000000000000000000000")

// predeployScoped keeps every predeploy and preinstall, plus the
// impls they point at via their EIP-1967 slot — with full per-account storage.
// Everything else (e.g. prefunded EOAs) is dropped.
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
	base := new(big.Int).SetBytes(predeployNamespaceBase.Bytes())
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
