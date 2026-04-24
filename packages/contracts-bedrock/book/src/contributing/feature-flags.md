# Feature Flags

This doc covers *how* we introduce and maintain feature flags in the contracts codebase. The broader principles — why feature toggles exist and the distinction between a toggle and configuration — live in the internal [Feature Toggles](https://www.notion.so/oplabs/Feature-Toggles-236f153ee16280f7932ce70f3956bda4) doc (oplabs workspace). If anything here conflicts with that doc, that doc wins.

## Categories

We use two categories of flag. Picking the right one up front avoids rework.

### Dev feature flag (`DEV_FEATURE__*`)

**Use when:** an in-flight feature is being built incrementally and must stay disabled in production until ready. These are short-lived.

**Where they live.** A `DEV_FEATURE__*` flag must be enabled in **both** of these files, and they must agree:

- `packages/contracts-bedrock/test/setup/FeatureFlags.sol` — the test-side reader
- `packages/contracts-bedrock/scripts/libraries/Config.sol` — the script-side reader

There is currently no mechanical check that these stay in sync; drift is caught only by the tests that happen to exercise both paths. If you add a new `DEV_FEATURE__*` and only update one of these files, CI will pass on the side you updated and silently skip the other.

**Default:** off.

**Lifetime:** removed in the release after the feature ships, or sooner.

### System Configuration Feature

**Use when:** different chains run with different feature sets, and the decision is made per chain at genesis or via governance. Custom gas tokens are the canonical example.

**Where it lives.** This is on-chain state, not a code-level toggle. It is read at runtime via `SystemConfig.isFeatureEnabled(bit)` on L1 and `L2ContractsManager.isFeatureEnabled(bit)` on L2. Both methods return the same underlying configuration — two projections of one state.

**How tests toggle it.** Tests do not wait for governance to mutate real chain state. Instead, the `SYS_FEATURE__*` environment variables set the configuration at test-harness time, so the same feature can be exercised with and without the bit enabled. The env var is not a separate flag — it is the test-side lever for the on-chain bit.

**Default.** Whatever the chain's existing state says. **Never assume the flag is unset.**

**Lifetime.** Permanent. Removing a System Configuration Feature requires a chain-level migration and is not something individual PRs do.

## Choosing the right category

Answer these questions in order:

1. Will this gate be removed within one or two releases? → **Dev feature flag**
2. Anything else → **System Configuration Feature**

The decision is binary because the toggle/configuration split is binary. If you find yourself wanting a third option — "permanent but also removable" — you are almost certainly describing a System Configuration Feature whose lifetime you wish were shorter than it actually is.

## Adding a new flag

The PR that introduces a flag must include:

- The flag definition in its canonical location (Solidity feature bit, env var, or both)
- CI matrix updated to exercise each reachable state of the flag — not only on/off
- A default that preserves current behavior — either off, or "unset = legacy"
- Tests for each reachable state (see next section)
- A tracking issue or PR-description line for removal if the flag is a dev feature. For a System Configuration Feature, say so explicitly: no removal criteria, and document who owns it.

## Writing code that reads a flag

**Flags are state machines, not booleans.**

Before writing code that branches on a flag, enumerate the reachable states of that flag, including the history the state encodes. Then confirm each state is handled.

A dev feature flag usually has two reachable states: on and off. A System Configuration Feature often has more:

- Off, never set
- On, freshly set this transaction
- On, set in a previous upgrade
- On, set and some downstream state has also changed (e.g. owner renounced)

The CGT incident that triggered this doc (PR [#20095](https://github.com/ethereum-optimism/optimism/pull/20095)) shipped code that handled the "off" and "freshly set" cases but reverted on "already set." The fix was one line. The line was not what was missing — the state enumeration was.

## Removing a flag

**Dev feature flags.** Remove in the release after the feature ships, along with the tests that only exercised the toggled-off branch. Stale dev flags are tech debt: the longer they sit, the more surrounding code grows to depend on their interaction.

**System Configuration Features.** These do not get removed casually. If a feature outlives its purpose, a separate migration is required — this is a chain-level operation.

## References

- [Feature Toggles (internal, oplabs workspace)](https://www.notion.so/oplabs/Feature-Toggles-236f153ee16280f7932ce70f3956bda4)
- PR #20095 — trigger incident for the state-machine guidance above
