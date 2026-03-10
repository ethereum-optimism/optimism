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

This runs `GenerateNUTBundle.s.sol` and writes `current-upgrade-bundle.json`.

### Snapshotting a bundle for a fork

```bash
just update-nuts <fork>
```

This copies `current-upgrade-bundle.json` to `op-node/rollup/derive/<fork>_nut_bundle.json` and updates `fork_lock.toml` with the sha256 hash and current git commit.

Typical usage after modifying contracts:

```bash
cd packages/contracts-bedrock && just generate-nut-bundle && cd ../..
just update-nuts karst
```

### Verifying a bundle

```bash
just verify-nuts <fork>
```

Checks that:
1. The bundle file exists and its sha256 matches the lock
2. If a `commit` is recorded, creates a temporary worktree at that commit, regenerates the bundle, and compares byte-for-byte

Requires `forge` for the provenance check (step 2).

### CI checks

- **`check-nut-locks`** — Verifies all bundle hashes match their lock entries, and that every `*_nut_bundle.json` file has a corresponding lock entry. Runs in CI on every PR.
- **`nut-bundle-check`** — Verifies `current-upgrade-bundle.json` is up-to-date with the contracts. Runs as part of `just check` in `packages/contracts-bedrock/`.

## fork_lock.toml schema

```toml
[<fork-name>]
bundle = "op-node/rollup/derive/<fork>_nut_bundle.json"  # repo-relative path
hash = "sha256:<hex>"                                      # sha256 of bundle contents
commit = "<full-sha>"                                      # commit that produced the bundle (optional)
```

## Adding a new fork's bundle

1. Implement fork-specific logic in `GenerateNUTBundle.s.sol`
2. `cd packages/contracts-bedrock && just generate-nut-bundle`
3. `just update-nuts <fork>`
4. Add `//go:embed <fork>_nut_bundle.json` and switch case in `op-node/rollup/derive/upgrade_transaction.go`
5. `just check-nut-locks` to verify
