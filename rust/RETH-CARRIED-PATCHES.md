# Reth fork carried patches

The patches the current `ethereum-optimism/reth` branch carries beyond its upstream base. The
step 2 audit of [UPDATING-RETH](UPDATING-RETH.md) decides on each update which of them the
new base already contains; this file says what the audit should expect and how to carry the
patches that need more than a cherry-pick. Update it in the monorepo PR that pins a fork
change: add a row for each new pick, and drop a row once the pinned base contains its fix.

| Patch | Upstream | First release | Fork PRs |
| --- | --- | --- | --- |
| [RPC replay through the block executor](#rpc-replay-through-the-block-executor) | paradigmxyz/reth#28266 (backport) | v2.11.0 | ethereum-optimism/reth#37 (`op-v2.6.0`), ethereum-optimism/reth#41 (`op-v2.7.0`), ethereum-optimism/reth#56 (`op-v2.8.1`) |
| Trie iterator seek boundary | paradigmxyz/reth#28264 | v2.10.0 | ethereum-optimism/reth#32 (`op-v2.6.0`), ethereum-optimism/reth#40 (`op-v2.7.0`), ethereum-optimism/reth#56 (`op-v2.8.1`) |
| [Forward fixes](#forward-fixes) | paradigmxyz/reth#28148 | v2.9.0 | ethereum-optimism/reth#44 (`op-v2.7.0`), ethereum-optimism/reth#56 (`op-v2.8.1`) |
| | paradigmxyz/reth#28253 (hand-ported) | v2.10.0 | ethereum-optimism/reth#44, ethereum-optimism/reth#56 |
| | paradigmxyz/reth#28314 (hand-ported) | v2.11.0 | ethereum-optimism/reth#44, ethereum-optimism/reth#56 |
| | paradigmxyz/reth#27716 | v2.11.0 | ethereum-optimism/reth#44, ethereum-optimism/reth#56 |
| | paradigmxyz/reth#28370 | v2.11.0 | ethereum-optimism/reth#44, ethereum-optimism/reth#56 |
| | paradigmxyz/reth#28403 | v2.11.0 | ethereum-optimism/reth#44, ethereum-optimism/reth#56 |
| | paradigmxyz/reth#28423 | v2.11.0 | ethereum-optimism/reth#44, ethereum-optimism/reth#56 |
| | paradigmxyz/reth#28431 | v2.11.0 | ethereum-optimism/reth#44, ethereum-optimism/reth#56 |
| `--txpool.enforce-tracked-nonce` flag | paradigmxyz/reth#28526 | none yet (2026-11-25) | ethereum-optimism/reth#47 (`op-v2.7.0`), ethereum-optimism/reth#56 (`op-v2.8.1`) |
| url 2.5.9 (RUSTSEC-2026-0592) | paradigmxyz/reth#28654 | none yet (2026-11-25) | ethereum-optimism/reth#53 (`op-v2.7.0`), ethereum-optimism/reth#56 (`op-v2.8.1`) |
| Storage v2 at a nonzero genesis | paradigmxyz/reth#28691 | none yet (2026-11-25) | ethereum-optimism/reth#55 (`op-v2.7.0`), ethereum-optimism/reth#56 (`op-v2.8.1`) |
| Drop anchor-relative nodes from prefix-set overlays | paradigmxyz/reth#28249 | v2.10.0 | ethereum-optimism/reth#56 (`op-v2.8.1`) |
| Skip the empty senders static file when sender recovery is fully pruned | paradigmxyz/reth#28598 | none yet (2026-11-25) | ethereum-optimism/reth#56 (`op-v2.8.1`) |
| [Masked state trie storage-root test](#masked-state-trie-storage-root-test) | fork-only (`d70cc04`) | never | `op-v2.6.0`, `op-v2.7.0`, ethereum-optimism/reth#56 (`op-v2.8.1`) |
| [Reverts-required overlay shape test](#reverts-required-overlay-shape-test) | fork-only | never | ethereum-optimism/reth#56 (`op-v2.8.1`) |

Patches without a section are plain `-x` picks from upstream: cherry-pick them onto a new
base that predates their first release.

## RPC replay through the block executor

Op-reth's SDM refund settlement is only visible to `debug_*`, `trace_*` and `ots_*` replays
through the block executor, and `rust/op-reth/crates/node/tests/it/rpc_replay_settlement/`
fails without it.

- **v2.11.0 and later:** drop the backport; the release contains paradigmxyz/reth#28266.
  Upstream's `inspect_transaction_without_commit` installs the tracer through the EVM's
  inspector slot, which op-reth's `PostExecCompositeInspector` occupies, so
  `OpBlockExecutor` needs an override or composition for it. Upstream's BAL branch skips
  per-transaction settlement; op-reth rejects computed BALs, keep that invariant under test.
- **v2.9.0 to v2.10.1** (evm2): drop the backport and cherry-pick paradigmxyz/reth#28266
  itself, with the same `OpBlockExecutor` work as above.
- **v2.7.0 to v2.8.1** (alloy-evm 0.39): cherry-pick the backport from the current branch;
  ethereum-optimism/reth#41 shows the conflicts it resolved onto v2.7.0.
  On v2.8.0 and v2.8.1 the plain replay loops that the backport replaces carry
  paradigmxyz/reth#27462's `is_cancelled()` checks: move them into the executor loops, as
  ethereum-optimism/reth#44 did, or replays lose cancellation with no compile error.
  ethereum-optimism/reth#56 did this on v2.8.1 by merging against v2.7.0 plus paradigmxyz/reth#27462.

## Forward fixes

On an alloy-evm 0.39 base (up to v2.8.1), cherry-pick them from the current branch, where
paradigmxyz/reth#28253 and paradigmxyz/reth#28314 are hand-ported from evm2. On an evm2 base,
cherry-pick the upstream commits. Op-reth keeps its own copies of the symbols
paradigmxyz/reth#27716 and paradigmxyz/reth#28431 patch (`UPSTREAM-MIRROR` tags in
`rust/op-reth`), so check those mirrors against each pick.

## Masked state trie storage-root test

A fork-only test, not an upstream pick: it pins that `OverlayStateProvider::storage_root` reads
trie nodes through the overlay when partial persistence leaves database storage-trie nodes older
than the persisted hashed storage. The Isthmus withdrawals root depends on it. Its subject has
no PR number, so check its `-x` source (`d70cc04`) by hand in the step 2 audit; carry it onto
every new branch and keep it passing.

## Reverts-required overlay shape test

A fork-only test, added with the paradigmxyz/reth#28249 pick: a prefix-set overlay built below
the persisted tip at masking 0 must carry no anchor-relative storage nodes. Since
paradigmxyz/reth#27651, `OverlayStateProvider::storage_root`, which the Isthmus withdrawals
root goes through, builds that overlay. Keep it on every new branch; it fails without #28249.
