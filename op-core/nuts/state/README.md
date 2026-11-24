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
| `lagoon_multichain_state.json` | lagoon, multi-chain | (next fork, multi-chain case) |

Each state composes from the previous one: `karst_state = jovian_state + (karst
bundle applied)`, and so on. The chain's seed is `jovian_state.json`, which has no
predecessor bundle and is built from jovian-era source.

## Multi-chain states

From Lagoon onward, the state depends on the size of the chain's dependency set.
The Lagoon activation block activates the interop contracts only when the
dependency set holds more than one chain. It then sets the `INTEROP` feature,
gives the interop proxies (CrossL2Inbox and others) an implementation and funds
ETHLiquidity.

`<fork>_multichain_state.json` is the state as of `<fork>` for a chain with a
dependency set of two chains. The second chain does not exist; only the size of
the dependency set changes the state. The multi-chain states compose the
same way: `<fork>_multichain_state = <prev>_multichain_state + (frozen <fork>
bundle applied)`. Karst has no multi-chain state, so the Lagoon multi-chain state
composes from `karst_state.json`.

The activation test runs a `-multichain` case for every fork from Lagoon onward.
That case boots from `<forks.Prev(F)>_multichain_state.json`, or from
`karst_state.json` for Lagoon.

## Generating the seed: `jovian_state.json`

`jovian_state` is the chain's seed — jovian has no predecessor NUT bundle, so it's
built from jovian-era source:

```bash
ops/scripts/gen-seed-state.sh
```

**Why jovian's own toolchain (not the current op-deployer):** the current
op-deployer cannot consume jovian era contracts. Building
inside the worktree pairs the era's contracts with the era's tooling.

## Generating subsequent states: compose

Every state past the seed is the **post-activation state of its fork**:
`<fork>_state = <prev>_state + (frozen <fork> bundle applied)`:

```bash
# karst_state.json = jovian_state + karst bundle
just nut-prefork-state-for karst
```

It boots the predecessor state, applies the frozen bundle at the activation block,
and dumps the post-activation predeploy-scoped state to `<fork>_state.json`. From
Lagoon onward, it also writes `<fork>_multichain_state.json`.

Under the hood this runs the env-gated `TestGenerateForkState`, which reuses the
validation test's `activateFork` so generation and validation stay in lockstep —
hence it lives in the proofs test suite rather than as a standalone binary.
Compose needs **no fork-era contract build** — it only re-runs the already-frozen
bundle, so current tooling is fine despite the ABI drift above.

**Generate one fork at a time, committing each first:** the loader embeds states at
compile time, so `<prev>_state.json` must be on disk before generating
`<fork>_state.json`.

To verify every committed fork state is up to date, run:

```bash
just check-nut-prefork-states
```

This regenerates committed non-seed fork states, including the multi-chain
states, and fails if the committed JSON changes.

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
