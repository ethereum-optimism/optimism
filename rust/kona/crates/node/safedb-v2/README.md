# `kona-safedb-v2`

A derivation safe-head database for the OP Stack rollup node.

`SafeDatabaseV2` records, for each L1 block that advanced the L2 safe head, a mapping from the L1
block to the L2 safe head derived as of that L1 block. It is a Rust port of the `op-node`
`safedb` package and is used by systems — such as fault proofs and interop — that need a
high-quality record of derivation progress.

The crate exposes:

- [`SafeDbV2`] — the database interface, backed by either a real store or a no-op.
- [`DisabledDatabaseV2`] — a no-op implementation for hosts that do not record derivation.
- `SafeDatabaseV2` — a [`rocksdb`]-backed implementation that persists records to disk, available
  behind the non-default `rocksdb` feature (off by default so that consumers needing only the
  trait avoid the rocksdb/libclang build dependency).

[`SafeDbV2`]: crate::SafeDbV2
[`DisabledDatabaseV2`]: crate::DisabledDatabaseV2
[`rocksdb`]: https://docs.rs/rocksdb
