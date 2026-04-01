# NUT Bundles

Network Upgrade Transaction (NUT) bundles define the L2 deposit transactions that activate a hardfork. Each bundle is a JSON file containing ordered transactions (implementation deployments, proxy upgrades, etc.) that the rollup node embeds and executes at the fork activation block.

## Files

| File | Purpose |
|------|---------|
| `fork_lock.toml` | Lock file mapping fork names to bundle paths, sha256 hashes, and source commits |
| `op-node/rollup/derive/<fork>_nut_bundle.json` | Embedded bundle consumed by op-node at fork activation |
| `packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json` | Generated bundle (not committed, regenerated from contracts) |

## Workflow

### Generating a bundle

```bash
cd packages/contracts-bedrock
just generate-nut-bundle
```

### Snapshotting a bundle for a fork

```bash
just update-nuts <fork>
```

This copies `current-upgrade-bundle.json` to `op-node/rollup/derive/<fork>_nut_bundle.json` and updates `fork_lock.toml` with the sha256 hash and current git commit.


### Verifying a bundle

```bash
just verify-nuts <fork>
```

Checks that:
1. The bundle file exists and its sha256 matches the lock
2. Creates a temporary worktree at the recorded commit, regenerates the bundle, and compares byte-for-byte

Requires `forge` for the provenance check (step 2).

### Locking a fork

Once a fork's bundle is finalized (e.g., governance post written), lock it:

1. Ensure the `commit` field is populated (run `just update-nuts <fork>` if needed)
2. Manually edit `fork_lock.toml` to add `locked = true` for the fork
3. CI (`check-nut-locks`) will enforce that the locked fork's hash cannot change

To unlock (e.g., for a critical fix), manually set `locked = false` in `fork_lock.toml`.

### CI checks

- **`check-nut-locks`** — Verifies all bundle hashes match their lock entries, all entries have a commit, locked forks haven't changed vs the base branch, and every `*_nut_bundle.json` file has a corresponding lock entry. Runs in CI on every PR.
- **`nut-bundle-check`** — Verifies `current-upgrade-bundle.json` is up-to-date with the contracts. Runs as part of `just check` in `packages/contracts-bedrock/`.

## fork_lock.toml schema

```toml
[<fork-name>]
bundle = "op-node/rollup/derive/<fork>_nut_bundle.json"  # repo-relative path
hash = "sha256:<hex>"                                      # sha256 of bundle contents
commit = "<full-sha>"                                      # commit that produced the bundle
locked = true                                              # prevents update-nuts from overwriting (optional, default false)
```

