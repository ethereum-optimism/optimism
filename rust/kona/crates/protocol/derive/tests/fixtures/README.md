# `kona-derive::pure` replay-test fixtures

This directory hosts the synthetic fixtures used by `tests/replay.rs`.

## Status

**Phase 3 ships with synthetic fixtures only.** A recorded-mainnet fixture
requires both an Ethereum L1 RPC (for headers + receipts + tx calldata) and
a beacon-chain endpoint (for blob bodies + KZG proofs) covering the chosen
L1 range, plus the L2 lookups (L2BlockInfo + system config) to bootstrap the
deriver at the safe head. That recording is not feasible in the build
environment where this PR was assembled, so the in-tree replay test instead
constructs synthetic L1Input streams that hit each of the code paths the
plan calls out as load-bearing:

- at least one `SystemConfigUpdated` trace event (sysconfig update path),
- at least one `NeedSpanBatchOverlap`/`add_span_batch_overlap` round-trip
  (overlap content check path),
- a Granite or Isthmus hardfork activation timestamp transition (closes the
  hidden risk that the deleted `Signal::Activation` site was load-bearing
  in the async pipeline).

The synthetic fixtures are byte-for-byte deterministic and reproduce on any
machine.

## Recording real fixtures (future work)

When the recording infrastructure is in place, the script will live behind a
`record-fixtures` cargo feature so CI replays from frozen bytes only. The
flow:

1. Pick an L1 range covering ≥1 syscfg update, ≥1 overlapping span batch,
   ≥1 post-Holocene hardfork activation boundary.
2. Walk the range, fetching headers, receipts, batch-inbox tx calldata, and
   blob bodies (KZG-verified).
3. Snapshot the L2 lookups (`l2_block_info_by_number`,
   `system_config_by_number`) the deriver requests.
4. Serialize everything to `mainnet_replay.bin` here.

Until that work happens, the synthetic fixtures keep the parity test honest.
