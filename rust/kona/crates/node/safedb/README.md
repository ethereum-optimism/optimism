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

The layout carries no compatibility guarantee with any other implementation and is free to change.

The crate exposes:

- [`SafeDb`] — the database interface, backed by either a real store or a no-op.
- [`DisabledDatabase`] — a no-op implementation for hosts that do not record derivation.
- `SafeDatabase` — a [`rocksdb`]-backed implementation that persists records to disk, available
  behind the non-default `rocksdb` feature (off by default so that consumers needing only the
  trait avoid the rocksdb/libclang build dependency).

[`SafeDb`]: crate::SafeDb
[`DisabledDatabase`]: crate::DisabledDatabase
[`rocksdb`]: https://docs.rs/rocksdb
