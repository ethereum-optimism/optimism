# NUT Bundles

Network Upgrade Transaction (NUT) bundles define the L2 deposit transactions that activate a hardfork. Each bundle is a JSON file containing ordered transactions (implementation deployments, proxy upgrades, etc.) that the rollup node embeds and executes at the fork activation block.

## Files

| File | Purpose |
|------|---------|
| `fork_lock.toml` | Lock file mapping fork names to bundle paths, sha256 hashes, and source commits |
| `bundles/<fork>_nut_bundle.json` | Embedded bundle consumed by op-node and kona-node at fork activation |

## Workflow

Updating a fork's bundle is a **two-PR flow**:

### PR 1 — Contracts change

Change the Solidity source, then regenerate the in-repo snapshots:

```bash
cd packages/contracts-bedrock
just generate-nut-bundle
```

This updates:
- `packages/contracts-bedrock/snapshots/semver-lock.json` (if any predeploy bytecode changed)
- `packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json` (the candidate bundle)

Commit these alongside your contracts change. **Merge this PR to `develop` before proceeding.**

### PR 2 — Snapshot the bundle for a fork

From a branch based on the updated `develop`:

```bash
just nut-snapshot-for <fork>
```

This copies `current-upgrade-bundle.json` to `op-core/nuts/bundles/<fork>_nut_bundle.json` and updates `fork_lock.toml` with the sha256 hash and the merge-base commit with `origin/develop`.

**Why merge-base, not HEAD?** The recorded commit is the [merge-base](https://git-scm.com/docs/git-merge-base) with `develop`. By ensuring that the recorded commit is a recent ancestor of `develop`, we can ensure that the reference will survives a squash-merge and persist in the history of the `develop` branch. This is why PR 1 must be merged first.

### Verifying a bundle

```bash
just nut-provenance-verify <fork>
```

Checks that:
1. The bundle file exists and its sha256 matches the lock
2. Creates a temporary worktree at the recorded commit, regenerates the bundle, and compares byte-for-byte

Requires `forge` for the provenance check (step 2).

### CI checks

- **`check-nut-locks`** — Verifies all bundle hashes match their lock entries, all entries have a commit, and every `*_nut_bundle.json` file has a corresponding lock entry. Runs in CI on every PR.

## fork_lock.toml schema

```toml
[<fork-name>]
bundle = "op-core/nuts/bundles/<fork>_nut_bundle.json"  # repo-relative path
hash = "sha256:<hex>"                                      # sha256 of bundle contents
commit = "<full-sha>"                                      # commit that produced the bundle
```
