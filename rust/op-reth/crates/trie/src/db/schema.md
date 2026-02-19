# Proof History Database Schema

> Location: `crates/optimism/trie/src/db`
> Backend: **MDBX** (via `reth-db`)
> Purpose: Serve **historical `eth_getProof`** by storing versioned trie data in a bounded window.

---

## Design Overview

This database is a **versioned, append-only history store** for Ethereum state tries.

Each logical key is stored with **multiple historical versions**, each tagged by a **block number**. Reads select the latest version whose block number is **≤ the requested block**.

### Core principles

* History tables are **DupSort** tables
* Each entry is versioned by `block_number`
* Deletions are encoded as **tombstones**
* A reverse index (`BlockChangeSet`) enables **range pruning**
* Proof window bounds are tracked explicitly

---

## Version Encoding

All historical values are wrapped in `VersionedValue<T>`.

### `VersionedValue<T>`

| Field          | Type              | Encoding             |
| -------------- | ----------------- | -------------------- |
| `block_number` | `u64`             | big-endian (8 bytes) |
| `value`        | `MaybeDeleted<T>` | see below            |

```
VersionedValue := block_number || maybe_deleted_value
```

---

### `MaybeDeleted<T>`

Encodes value presence or deletion:

| Logical value | Encoding                     |
| ------------- | ---------------------------- |
| `Some(T)`     | `T::compress()`              |
| `None`        | empty byte slice (`len = 0`) |

An empty value represents **deletion at that block**.

---

## Tables

---

## 1. `AccountTrieHistory` (DupSort)

Historical **branch nodes** of the **account trie**.

### Purpose

Reconstruct account trie structure at any historical block.

### Schema

| Component | Type                                |
| --------- | ----------------------------------- |
| Key       | `StoredNibbles`                     |
| SubKey    | `u64` (block number)                |
| Value     | `VersionedValue<BranchNodeCompact>` |

### Key encoding

* `StoredNibbles` = compact-encoded trie path

### Semantics

For a given trie path:

* Multiple versions may exist
* Reader selects highest `block_number ≤ target_block`

---

## 2. `StorageTrieHistory` (DupSort)

Historical **branch nodes** of **per-account storage tries**.

### Schema

| Component | Type                                |
| --------- | ----------------------------------- |
| Key       | `StorageTrieKey`                    |
| SubKey    | `u64`                               |
| Value     | `VersionedValue<BranchNodeCompact>` |

### `StorageTrieKey` encoding

```
StorageTrieKey :=
  hashed_address (32 bytes)
  || StoredNibbles::encode(path)
```

Ordering:

1. `hashed_address`
2. trie path bytes

---

## 3. `HashedAccountHistory` (DupSort)

Historical **account leaf values**.

### Schema

| Component | Type                      |
| --------- | ------------------------- |
| Key       | `B256` (hashed address)   |
| SubKey    | `u64`                     |
| Value     | `VersionedValue<Account>` |

### Semantics

Stores nonce, balance, code hash, and storage root per account per block.

---

## 4. `HashedStorageHistory` (DupSort)

Historical **storage slot values**.

### Schema

| Component | Type                           |
| --------- | ------------------------------ |
| Key       | `HashedStorageKey`             |
| SubKey    | `u64`                          |
| Value     | `VersionedValue<StorageValue>` |

### `HashedStorageKey` encoding

Fixed 64 bytes:

```
hashed_address (32 bytes) || hashed_storage_key (32 bytes)
```

### `StorageValue` encoding

* Wraps `U256`
* Encoded as **32-byte big-endian**

---

## 5. `BlockChangeSet`

Reverse index of **which keys were modified in a block**.

### Purpose

Efficient pruning by block range.

### Schema

| Component | Type                 |
| --------- | -------------------- |
| Key       | `u64` (block number) |
| Value     | `ChangeSet`          |

### `ChangeSet` structure

```rust
pub struct ChangeSet {
  pub account_trie_keys: Vec<StoredNibbles>,
  pub storage_trie_keys: Vec<StorageTrieKey>,
  pub hashed_account_keys: Vec<B256>,
  pub hashed_storage_keys: Vec<HashedStorageKey>,
}
```

### Encoding

* Serialized using **bincode**

---

## 6. `ProofWindow`

Tracks active proof window bounds.

### Schema

| Component | Type              |
| --------- | ----------------- |
| Key       | `ProofWindowKey`  |
| Value     | `BlockNumberHash` |

### `ProofWindowKey`

| Variant         | Encoding |
| --------------- | -------- |
| `EarliestBlock` | `0u8`    |
| `LatestBlock`   | `1u8`    |

### `BlockNumberHash` encoding

```
block_number (u64 BE, 8 bytes)
|| block_hash (B256, 32 bytes)
```

Total size: **40 bytes**

---
Here is a **short, clean, professional** version suitable for `SCHEMA.md`:

---

## Reads: Hashed & Trie Cursors

Historical reads are performed using **hashed cursors** and **trie cursors**, both operating on versioned history tables.

All reads follow the same rule:

> Select the newest entry whose block number is **≤ the requested block**.

---

### Hashed Cursors

Hashed cursors read **leaf values** from:

* `HashedAccountHistory`
* `HashedStorageHistory`

They answer:

> *What was the value of this account or storage slot at block B?*

For a given key, the cursor scans historical versions and returns the latest valid value. Tombstones indicate deletion and are treated as non-existence.

---

### Trie Cursors

Trie cursors read **trie branch nodes** from:

* `AccountTrieHistory`
* `StorageTrieHistory`

They answer:

> *Which trie nodes existed at this path at block B?*

These cursors enable reconstruction of Merkle paths required for proof generation.

---

### Combined Usage

When serving `eth_getProof`:

* Trie cursors reconstruct the Merkle path
* Hashed cursors supply the leaf values

Both are evaluated at the same target explained block to produce deterministic historical proofs.


---
## Writes: `store_trie_updates` (Append-Only)

`store_trie_updates` persists all state changes introduced by a block using a strictly **append-only** write model.


### Purpose

The function records **historical trie updates** so that state and proofs can be reconstructed at any later block.

---

### What is written

For a processed block `B`, the following data is appended:

* Account trie branch nodes → `AccountTrieHistory`
* Storage trie branch nodes → `StorageTrieHistory`
* Account leaf updates → `HashedAccountHistory`
* Storage slot updates → `HashedStorageHistory`
* Modified keys → `BlockChangeSet[B]`

All entries are tagged with the same `block_number`.

---

### How writes work

For each updated item:

1. A `VersionedValue` is created with:

    * `block_number = B`
    * the encoded node or value

2. The entry is appended to the corresponding history table.

No existing entries are modified or replaced.


---
Here is a **concise, professional, SCHEMA.md-style** explanation aligned exactly with your clarification:

---

## Initial State Backfill

### Source database (Reth)

The initial state is sourced from **Reth’s main execution database**, which only contains data for the **current canonical state**.

The backfill reads from the following Reth tables:

* `HashedAccounts` – current account leaf values
* `HashedStorages` – current storage slot values
* `AccountsTrie` – current account trie branch nodes
* `StoragesTrie` – current storage trie branch nodes

These tables do **not** contain historical versions; they represent a single finalized state snapshot.

---

### Destination database (Proofs storage)

The data is copied into the **proofs history database** (`OpProofsStore`), which is a **versioned, append-only** store designed for historical proof generation.

---

### How the initial state is created

During backfill:

1. The current state is fully scanned from Reth tables using read-only cursors.
2. All entries are written into the proofs storage as versioned records.
3. This creates a complete **baseline state** inside the proofs DB.

The backfill runs only once and is skipped if the proofs DB already has an `earliest_block` set.

---

### Why block `0` is used

Since Reth tables only represent the **current state**, the copied data must be assigned a synthetic version.

Block **`0`** is used as the baseline version because:

* It is ≤ any real block number
* It establishes a stable initial version for all keys
* Later block updates naturally override it using higher block numbers

This makes block `0` the canonical **initial state anchor** for versioned reads.

---
