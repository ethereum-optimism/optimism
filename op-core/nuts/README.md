# NUT Bundles

Network Upgrade Transaction (NUT) bundles define the L2 deposit transactions that activate a hardfork. Each bundle is a JSON file containing ordered transactions (implementation deployments, proxy upgrades, etc.) that the rollup node embeds and executes at the fork activation block.

## Files

| File | Purpose |
|------|---------|
| `fork_lock.toml` | Lock file mapping fork names to bundle paths, sha256 hashes, and source commits |
| `op-node/rollup/derive/<fork>_nut_bundle.json` | Embedded bundle consumed by op-node at fork activation |

## Workflow

### Generating a bundle

```bash
cd packages/contracts-bedrock
just generate-nut-bundle
```

### Snapshotting a bundle for a fork

```bash
just nut-snapshot-for <fork>
```

This copies `current-upgrade-bundle.json` to `op-node/rollup/derive/<fork>_nut_bundle.json` and updates `fork_lock.toml` with the sha256 hash and the merge-base commit with `origin/develop`.

**Important:** The recorded commit is the merge-base with develop, not HEAD. This ensures the commit survives squash-merge. Contract changes must be merged to develop in a separate PR *before* snapshotting the bundle.


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
bundle = "op-node/rollup/derive/<fork>_nut_bundle.json"  # repo-relative path
hash = "sha256:<hex>"                                      # sha256 of bundle contents
commit = "<full-sha>"                                      # commit that produced the bundle
```

