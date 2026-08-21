# `lokahi-interop`

The lokahi interop verifier: the timestamp-lockstep round loop, and the stores it keeps.

## The round loop

[`Verifier`] verifies one L2 timestamp per round, in lockstep across the whole chain set. A round
is four steps and no others:

1. **observe** — read every chain and the L1 once, into a [`RoundObservation`];
2. **decide** — reach a [`Decision`] from that value alone, through [`check_preconditions`] and
   [`decide_verified_result`], with no further reads;
3. **write ahead** — record an effectful decision in the verified store's WAL slot, durably,
   before any of its side effects begins; and
4. **apply** — perform the side effects, then clear the slot.

Reading once and deciding from the result is what makes a round reproducible: replaying a recorded
observation through the decision functions must reach the decision the live node reached. Writing
ahead before acting is what makes a crash mid-apply recoverable — every side effect an applied
decision has is idempotent, so a restart re-applies the slot it finds.

The message-validity rules are not implemented here. They are
[`kona_interop::MessageRules`](kona_interop::MessageRules), shared with the fault-proof
consolidation path, so a node's verdict and a proof's verdict on the same message cannot differ.
What this crate adds is the two things the rules cannot know by themselves: which blocks are in the
round, and whether a referenced initiating message exists.

Only [`Decision::Wait`] and [`Decision::Advance`] are applied today. A round that reaches
[`Decision::Invalidate`] or [`Decision::Rewind`] records the decision and holds the verified
frontier where it is, so cross-safety stops rather than promoting a block the verifier believes is
wrong.

## The stores

Three, one per kind of state the verifier keeps on disk:

- [`VerifiedStore`] — the verified frontier, one record per verified timestamp, plus the
  single-slot write-ahead log.
- [`LogStore`] — one per chain: sealed blocks with their log hashes and executing messages, the
  store the verifier asks "does this initiating message exist?".
- [`OutputArchive`] — the output preimages of blocks that were invalidated and replaced. Its own
  database, because it is the only store here whose loss is unrecoverable.

Every store is written against the [`Kv`] backend seam. The on-disk backend is `RocksKv`, behind
the off-by-default `rocksdb` feature so consumers that only need the traits, the record encodings
or [`MemoryKv`] avoid the rocksdb/libclang build dependency.

## The seams it observes through

[`InteropChain`] is one L2 chain as the verifier reads it: a chain controller for the
local-safe-at-timestamp pairing, and a read-only execution layer for logs and output roots.
[`L1Canonical`] is the L1 they all derive from. Both are read-only — the round loop's writes go to
the stores above and, in a later phase, to the single cross-safe promotion entry point.
