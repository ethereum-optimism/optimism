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

**Why merge-base, not HEAD?** The recorded commit is the merge-base with `develop`, not HEAD, so the reference survives squash-merge. That only works if the contracts source that produced the bundle is already on `develop` — which is why PR 1 must land first.

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

## Reading a bundle diff

A single predeploy edit can cascade into multiple bundle entries. For example, a patch bump to `L1Block.sol` typically produces four diffs:

1. **Deploy L1Block Implementation** — new bytecode (source changed).
2. **Deploy L1BlockCGT Implementation** — new bytecode (inherits the new version string via `super.version()`).
3. **Deploy L2ContractsManager Implementation** — new bytecode (hardcodes predeploy impl addresses, which shift when their bytecode changes).
4. **L2ProxyAdmin Upgrade Predeploys** — new calldata (batch upgrade now points at the new impl addresses).

When reviewing a bundle diff, check that every changed entry traces back to an intentional source change. Unexpected entries indicate a deeper impact than the diff suggests.

## fork_lock.toml schema

```toml
[<fork-name>]
bundle = "op-core/nuts/bundles/<fork>_nut_bundle.json"  # repo-relative path
hash = "sha256:<hex>"                                      # sha256 of bundle contents
commit = "<full-sha>"                                      # commit that produced the bundle
```

## Troubleshooting

### `FAIL: provenance verification: regenerated bundle at commit <X> differs from committed bundle`

The commit recorded in `fork_lock.toml` does not, when checked out, produce the bundle that is currently committed to the repo. The most common cause is running `just nut-snapshot-for <fork>` on a branch whose contracts source has not yet landed on `develop` — the merge-base with `origin/develop` resolves to a commit that predates the change being bundled.

**Fix:** merge the contracts PR to `develop` first, then create a new branch from the updated `develop` and re-run `just nut-snapshot-for <fork>`.

