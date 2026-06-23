# `kona-safedb`

A derivation safe-head database for the OP Stack rollup node.

`SafeDb` records, for each L1 block that advanced the L2 safe head, a mapping from the L1
block to the L2 safe head derived as of that L1 block. It is a Rust port of the `op-node`
`safedb` package and is used by systems — such as fault proofs and interop — that need a
high-quality record of derivation progress.

The crate exposes:

- [`SafeDbV1`] — the database interface, backed by either a real store or a no-op.
- [`DisabledDb`] — a no-op implementation for hosts that do not record derivation.
- `SafeDb` — a [`rocksdb`]-backed implementation that persists records to disk, available
  behind the non-default `rocksdb` feature (off by default so that consumers needing only the
  trait avoid the rocksdb/libclang build dependency).

[`SafeDbV1`]: crate::SafeDbV1
[`DisabledDb`]: crate::DisabledDb
[`rocksdb`]: https://docs.rs/rocksdb
