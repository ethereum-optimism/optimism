# Pre-fork state artifacts

`<fork>_state.json` is the frozen L2 **predeploy** state (proxies + their
implementations, with full storage) as of that fork — i.e. the state a chain is
in once `<fork>` has activated, before the next fork. The NUT bundle activation
test (`rust/kona/tests/proofs/nut_bundle_activation_test.go`) for fork **F** boots
from `<forks.Prev(F)>_state.json` instead of building genesis from current source,
so it exercises the immutable, locked bundle against the predeploy versions it was
actually designed to upgrade — not whatever versions HEAD happens to be at.

See `docs/proposals/nut-bundle-test-prestate.md` for the full rationale.

## Naming & workflow

State files are named after the fork they **represent**, not the bundle that
consumes them. The intent is to generate each state when its fork ships, so a new
fork's bundle can reuse the state produced by the previous fork:

| File | State as of | Consumed by |
|------|-------------|-------------|
| `jovian_state.json` | jovian | karst bundle test |
| `karst_state.json`  | karst  | interop bundle test |
| `interop_state.json`| interop | (next fork) |

Each state composes from the previous one: `karst_state = jovian_state + (karst
bundle applied)`, and so on. The chain's seed is `jovian_state.json`, which has no
predecessor bundle and is built from jovian-era source.

## `jovian_state.json` — the seed

### How it was generated

```bash
ops/scripts/gen-jovian-prestate-seed.sh
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

**Why jovian's own toolchain (not the `nut-prestate-gen` tool / current op-deployer):**
the current op-deployer cannot consume jovian-era contracts — their ABIs have
drifted (e.g. `DeployImplementations` dropped its `protocolVersionsProxy` input
field since jovian, so the current Go expects 15 input fields where jovian's
contract has 16). Generating inside the worktree pairs jovian contracts with
jovian tooling, which are mutually consistent.

### Caveat: L1-derived slots are environment-specific

Jovian's op-deployer randomizes the CREATE2 salt, which varies the L1 counterpart
addresses embedded in a few L2 predeploys (`L2CrossDomainMessenger`,
`L2StandardBridge`, `L2ERC721Bridge` — their `otherMessenger`/`otherBridge`
slots). The committed file is therefore **one specific instance**, and the script
does **not** reproduce it byte-for-byte. The activation test is expected to reset
these L1-derived slots to test-controlled values on entry, so the exact values
here are not load-bearing.
