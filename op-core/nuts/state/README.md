# Pre-fork state artifacts

`<fork>_state.json` is the frozen L2 **predeploy** state (proxies + their
implementations, with full storage) as of that fork — i.e. the state a chain is
in once `<fork>` has activated, before the next fork. The NUT bundle activation
test (`rust/kona/tests/proofs/nut_bundle_activation_test.go`) for fork **F** boots
from `<forks.Prev(F)>_state.json` instead of building genesis from current source,
so it exercises the immutable, locked bundle against the predeploy versions it was
actually designed to upgrade.

## Naming & workflow

State files are named after the fork they **represent**, not the bundle that
consumes them. The reason is that the future forks' name are not known in advance,
and the intent is to generate each state when its fork ships:

| File | State as of | Consumed by |
|------|-------------|-------------|
| `jovian_state.json` | jovian | karst bundle test |
| `karst_state.json`  | karst  | lagoon bundle test |
| `lagoon_state.json` | lagoon | (next fork) |

(The fork after karst is `lagoon`; the bundle it activates is still named
`interop_nut_bundle.json` — the fork was renamed, the bundle wasn't.)

Each state composes from the previous one: `karst_state = jovian_state + (karst
bundle applied)`, and so on. The chain's seed is `jovian_state.json`, which has no
predecessor bundle and is built from jovian-era source.

## Generating the seed: `jovian_state.json`

`jovian_state` is the chain's seed — jovian has no predecessor NUT bundle, so it's
built from jovian-era source:

```bash
ops/scripts/gen-seed-state.sh jovian
```

That script:

1. Creates a git worktree at the pinned **jovian-era commit**
   `79cee4ec028db485150db71e64d0921a78960f70` (jovian mainnet activation was
   2025-12-02, L1 timestamp `1764691201`; this commit is jovian-era source,
   before karst's contract changes).
2. Builds the jovian-era contracts there.
3. Runs a small dumper **inside the worktree** that calls jovian's own
   `op-e2e/config.L2Allocs(DefaultAllocType, L2AllocsJovian)`, filters to
   predeploy proxies + their implementations, and writes this file.

**Why jovian's own toolchain (not the current op-deployer):** the current
op-deployer cannot consume jovian/karst-era contracts — their ABIs have drifted
(e.g. `DeployImplementations` dropped its `protocolVersionsProxy` input field, so
the current Go expects 15 input fields where the old contract has 16). Building
inside the worktree pairs the era's contracts with the era's tooling.

## Generating subsequent states: compose

Every state past the seed is the **post-activation state of its fork**:
`<fork>_state = <prev>_state + (frozen <fork> bundle applied)`. The activation test
executes exactly that, so it doubles as the generator — set
`OP_E2E_GEN_PRESTATE=<fork>` and run it:

```bash
# karst_state.json = jovian_state + karst bundle
OP_E2E_GEN_PRESTATE=karst go test -run 'TestActivationBlockNUTBundle/karst' ./rust/kona/tests/proofs/
```

It boots the predecessor state, applies the frozen bundle at the activation block,
dumps the post-activation predeploy-scoped state to `<fork>_state.json`, and stops
before the fault-proof step. Compose needs **no fork-era contract build** — it only
re-runs the already-frozen bundle, so current tooling is fine despite the ABI drift
above.

**Generate one fork at a time, committing each first:** the loader embeds states at
compile time, so `<prev>_state.json` must be on disk before generating
`<fork>_state.json`.

## Determinism

**Composed states** (`karst_state` and later) are byte-reproducible given the
committed seed. `<fork>_state = <prev>_state + (frozen bundle)` is deterministic,
and the generator zeroes **L1Block's L1-attribute slots** (integer slots 0..8 —
timestamp, L1 hash, etc.; see `canonicalizeL1Block`). Those are written by the
per-block L1-info deposit, so they'd otherwise vary with the test's wall-clock L1
genesis time and are re-set anyway when the state is consumed.

The **seed** (`jovian_state`) is **not** byte-reproducible: its worktree
op-deployer run randomizes the CREATE2 salt, so the L1 counterpart addresses baked
into the cross-domain bridge predeploys (`otherMessenger`/`otherBridge`) differ per
generation. It's committed as one specific instance. Composed states inherit those
addresses verbatim; they're arbitrary and the activation test doesn't depend on
them, so this doesn't affect what the test validates.
