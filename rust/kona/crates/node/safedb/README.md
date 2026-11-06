# `kona-safedb`

A derivation safe-head database for the OP Stack rollup node.

`SafeDatabase` records, for each L1 block that advanced the L2 safe head, a mapping from the L1
block to the L2 safe head derived as of that L1 block. Systems that need a high-quality record of
derivation progress — such as fault proofs and interop — read from it.

Records live in a single [`rocksdb`] column, keyed by L1 block number so that key order is L1
order:

```text
key   (9 bytes):  0x00 | L1 block number (big-endian u64)
value (72 bytes): L1 block hash (32) | L2 block hash (32) | L2 block number (big-endian u64)
```

Correctness depends on the caller: updates must arrive in ascending L1 order and none may be
missed. See [`SafeDb::safe_head_updated`].

The layout carries no compatibility guarantee with any other implementation and is free to change.

## RPC error contract

The node's `optimism_safeHeadAtL1Block` and pre-interop `superroot_atTimestamp` handlers
translate `SafeDB` failures into JSON-RPC errors with code `-32000`. A missing L1 lookup
record has the bare message `not found`. The reverse lookup used by `superroot_atTimestamp`
distinguishes `l1 at safe head not found` (not recorded yet) from
`l1 at safe head history unavailable` (before the available history), with an operation
prefix of `l1AtSafeHead@<blockNumber>`. A disabled database has a message containing
`safe head database not enabled`, prefixed with the attempted operation. These conditions
remain distinct so clients can identify unavailable historical coverage without treating a
disabled database as an empty one. Other storage failures return an error with operation
context rather than a successful historical answer.

The crate exposes:

- [`SafeDb`] — the database trait.
- `SafeDatabase` — a [`rocksdb`]-backed implementation that persists records to disk, available
  behind the non-default `rocksdb` feature (off by default so that consumers needing only the
  trait avoid the rocksdb/libclang build dependency).

[`SafeDb`]: crate::SafeDb
[`SafeDb::safe_head_updated`]: crate::SafeDb::safe_head_updated
[`rocksdb`]: https://docs.rs/rocksdb
