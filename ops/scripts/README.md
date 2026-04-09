# ops/scripts

## Component release checkers

These scripts help answer a narrow question for selected release artifacts:

> Did any tracked build input change between the last stable release tag and `develop`?

Currently supported components:
- `op-node`
- `op-batcher`
- `kona-node`

### Scripts

- `check-component-release.py` — check one component
- `check-component-releases.sh` — wrapper that checks a default set of components

### How matching works

The checker is dependency-based, not a broad path-glob heuristic.

- **Go components** (`op-node`, `op-batcher`): uses `go list -deps -json` from the binary entrypoint and tracks local source/build inputs in the transitive package graph.
- **Rust components** (`kona-node`): uses `cargo metadata` from the package and tracks local manifests, build scripts, and source trees in the transitive workspace crate graph.

This means the output is best read as:
- `artifact_changed: yes` → tracked build inputs changed since the last stable release tag
- `artifact_changed: no` → no tracked build inputs changed

## Usage

### Check one component

```bash
python3 ops/scripts/check-component-release.py op-node
python3 ops/scripts/check-component-release.py op-batcher
python3 ops/scripts/check-component-release.py kona-node
```

Useful flags:

```bash
python3 ops/scripts/check-component-release.py op-node --ref origin/develop --fetch
python3 ops/scripts/check-component-release.py op-node --include-merges
python3 ops/scripts/check-component-release.py op-node --json
python3 ops/scripts/check-component-release.py op-node -vvv
```

By default the output is concise. Use `-vvv` for full details, including matched packages/crates and changed tracked files.

### Check the default component set

```bash
ops/scripts/check-component-releases.sh
```

Pass checker flags directly through the wrapper:

```bash
ops/scripts/check-component-releases.sh -vvv
ops/scripts/check-component-releases.sh --ref origin/develop --fetch
ops/scripts/check-component-releases.sh op-batcher --include-merges
```

You can also use explicit passthrough with `--`:

```bash
ops/scripts/check-component-releases.sh op-node -- --json
```

### Default wrapper components

The wrapper checks these by default:
- `op-node`
- `op-batcher`
- `kona-node`

To change that set, edit `DEFAULT_COMPONENTS` in:

- `ops/scripts/check-component-releases.sh`
