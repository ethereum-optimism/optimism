# `kona-chainview`

An incrementally maintained chain view for `kona-node`: the L1 blocks the derivation
pipeline traverses, the unsafe-block signer read at each L1 head, engine-confirmed derived
L2 blocks with the L1 block they were derived from, and the views that follow from them
(finalized L2, safe head per L1 block, the current gossip signer).

The view is a SQL program (`src/chainview.sql`) compiled at build time by upstream
Feldera's `sql-to-dbsp` into a [`dbsp`](https://crates.io/crates/dbsp) circuit that runs
in-process on its own thread. Actors push facts (inserts and retractions) into typed input
handles; the circuit maintains the views incrementally; the host integrates the per-step
deltas into a snapshot and a safe-head history. A reorg is a retraction, not a reset.

## What it replaced in kona-node

| Before | With the chain view |
|---|---|
| `L2Finalizer`: a map from L1 number to derived L2 number, cleared on every pipeline reset, finalizing by number | `finalized_l2`: the newest canonical derived block whose derived-from L1 block is at or below finalized L1, finalized by hash, surviving resets |
| `optimism_syncStatus` making three live L1 RPCs per call | L1 fields read from the snapshot |
| `optimism_safeHeadAtL1Block` unsupported | served from the integrated `safe_head_updates` history (op-node `SafeDB` semantics: nearest entry at or below the requested L1 block) |
| unsafe-block signer scanned from the polled head block's logs only, losing rotations in skipped blocks | the signer read from the `SystemConfig` contract's storage at every polled head, so it is right at the head even while catching up |

## Tables (facts the host pushes)

| Table | Pushed by | Notes |
|---|---|---|
| `l1_blocks` | derivation actor | every L1 origin its pipeline advances through, one block per height; a re-walked height with another hash (a reorg) replaces the old block and its update rows; rows below finalized L1 are dropped |
| `l1_status` | `ChainViewActor` (`head`, `safe`, `finalized` from the watcher's tags), derivation actor (`current`, the pipeline origin) | one row per kind |
| `l2_status` | `ChainViewActor` from `EngineState` | one row per engine label |
| `l2_safe_blocks` | derivation actor | one row per engine-confirmed derived block, with `derived_from`; `LATENESS 4096` on `derived_from_number` (`kona_chainview::LATENESS`; the node refuses to start the chain view for a chain whose `seq_window_size + channel_timeout` reaches it) |
| `unsafe_block_signer` | `ChainViewActor` | one row: the signer read from the `SystemConfig` contract at the latest L1 head |

Status rows are replaced by retracting the previous row; the driver remembers what it
pushed so callers only send new values.

## Views

| View | Meaning |
|---|---|
| `l2_safe_canonical` | derived blocks whose derived-from L1 hash is not contradicted by the tracked canonical hash at that height |
| `safe_head_updates` | per derived-from L1 block, the most recently asserted safe block (by host `seq`) |
| `finalized_l2` | at most one row: the block the engine may finalize next |
| `current_signer` | the unsafe-block signer at the latest L1 head |
| `error_view` | compiler-injected: rows the circuit rejected (LATENESS violations), counted as `lateness_drops` |

## Host integration

- `spawn(ChainViewConfig)` builds the circuit on a `kona-chainview` thread and returns a
  `ChainViewHandle`: a cloneable `ChainViewClient` (push facts, read the snapshot, query
  `safe_head_at_l1`) and the thread's exit signal.
- `ChainViewActor` (in `kona-node-service`) feeds engine heads, forwards the signer to the
  network actor, and turns the thread's exit into an actor error.
- The L1 watcher's `ChainViewL1Sync` walks every L1 block between polls, detects reorgs by
  parent hash, walks back to the common ancestor, re-anchors when a reorg reaches below the
  window, and fails closed on a reorg at or below finalized L1.
- The derivation actor pairs each confirmed safe head with the pending attributes'
  derived-from block, retracts derived blocks above the safe head on `Signal::Reset`, and
  sends `FinalizeBlockId::ByHash` when `finalized_l2` names a new block at or below the
  confirmed safe head.

Nothing is persisted: on restart the window is backfilled from L1 and the history starts
empty, matching kona's principle that local state is recoverable.

## Building

The SQL compiler is a Java program shipped as a release asset of the pinned Feldera
version. It is never downloaded by `build.rs`:

```bash
cd rust
just chainview-fetch-compiler   # downloads and sha256-verifies the pinned JAR
just chainview-doctor           # java version, JAR location
cargo check -p kona-chainview
```

`build.rs` looks for the JAR at `$KONA_SQL2DBSP_JAR`, then at
`${XDG_CACHE_HOME:-~/.cache}/kona-chainview/`, and fails with instructions otherwise.
A JDK 19–21 must be on `PATH` or at `$JAVA_HOME`. The generated code lives only in
`OUT_DIR`; Rust-only edits never invoke the compiler.

The Feldera crate pins in the workspace `Cargo.toml` and the JAR version in `build.rs`
must match; `build.rs` refuses to build otherwise. `src/handles.rs` pins the type and
position of every input and output handle, so a change to the SQL that reorders or retypes
a relation fails to compile there.

## Running kona-node with it

The chain view is part of every kona-node: finality, `optimism_syncStatus` and
`optimism_safeHeadAtL1Block` have no other implementation, and it has no flags. Its L1
blocks are the derivation pipeline's own traversal, so it costs one `eth_getLogs` per L1
block the pipeline advances through and one storage read per new L1 head.
