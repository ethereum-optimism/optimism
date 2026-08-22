# `lokahi-interop`

Persistent stores for the lokahi interop actor.

Three stores, one per kind of state the timestamp-lockstep verifier keeps on disk:

- [`VerifiedStore`] — the verified frontier, one record per verified timestamp, plus a
  single-slot write-ahead log for the effectful decision currently being applied.
- [`LogStore`] — one per chain: sealed blocks with their log hashes and executing messages,
  the store the verifier asks "does this initiating message exist?".
- [`OutputArchive`] — the output preimages of blocks that were invalidated and replaced. Its
  own database, because it is the only store here whose loss is unrecoverable.

Every store is written against the [`Kv`] backend seam. The on-disk backend is `RocksKv`, behind
the off-by-default `rocksdb` feature so consumers that only need the traits, the record encodings
or [`MemoryKv`] avoid the rocksdb/libclang build dependency.
