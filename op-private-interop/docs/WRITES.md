# Private write publication

Claim wire version 2 publishes opaque write records in the leading claim transaction of each
sequencer range. This supersedes the version-1 wire in older design notes. The execution proof
mode remains **attested**: the batcher's signature authenticates the records, while their
completeness and correctness remain an operator assertion. A future batch execution verifier
must bind and verify the entire write set as well as the existing claim fields.

## Records

Each record is 72 bytes: a 32-byte tag, a 32-byte value commitment, and an unsigned 8-byte
big-endian private block number. Records are strictly sorted by tag without duplicates.
For fixed-width fields:

```
tag = keccak256("optimism.private-interop.write-key.v1" || uint64be(chainID)
                || uint8(kind) || address20 || slot32)
valueCommitment = keccak256("optimism.private-interop.write-value.v1" || tag || value32)
```

Kinds are existence (0), balance (1), nonce (2), code hash (3), storage reset (4), and storage (5).
Non-storage kinds use slot zero. Integers are 32-byte big-endian values. Missing account fields
normalize to zero balance/nonce and the empty-code hash; existence distinguishes a missing
account from an existing empty one. Reset uses value zero and invalidates previous slot values.
Tags and value commitments are deterministic, not encryption: guessed addresses, slots and
low-entropy values can be checked. No plaintext addresses, keys, or values enter the write payload.

The execution client reexecutes each requested block against its exact parent hash, includes all
system operations, deposits and fee accounting, and checks the resulting state root against the
block header. The RPC returns **block-boundary net changes**, not intermediate accesses. Reverted
storage writes and values restored within the block disappear; surviving nonce/fees remain.
A storage wipe emits an account-wide reset marker, including for slots never loaded by the block.
Surviving nonzero slots after recreation are explicitly emitted even if they match their old values.

The batcher folds blocks in order, keeping the most recent record for every changed tag. A key
changed and restored across different blocks remains in the range record. A record names its
last change, not necessarily the range's last block. This permits interpreting deletion/recreation:
slot records older than the latest reset are invalid; a surviving slot record at the reset's same
block is the final recreated value. Every slot dependency must track existence and reset too.

## Publication and reading

`debug_privateBlockWrites(blockHash)` is available in op-reth's operator/debug RPC namespace.
It requires historical parent state and runs bounded concurrent full-block execution. The private
batcher requires this endpoint: errors, missing responses, wrong block hashes, and malformed or
out-of-order records fail closed. There is no fallback to an empty write set.

The claim's ABI adds `bytes writes` after `bytes proof`, and its version is 2. `ClaimRegistry`
checks record shape, ordering and block bounds. Its running claim hash includes every write byte.
It emits no logs, preserving the original projected message positions. The calldata rides inside
normal sequencer batches to L1. Consumers obtain the write data from those batches or the derived
projection transactions, not from an unavailable sidecar commitment.

`codec.CheckUnchanged` checks a complete sequence of already-authenticated claims between range
checkpoints. It refuses missing ranges; a forward gap allowed by the registry is not evidence of
no private writes. It must not be fed merely decoded, unverified operator data. Callers still
need membership evidence for the original values. Value commitments let a user authenticate a
candidate update but do not supply that value or a refreshed Merkle branch.

## Capacity and activation

Production closes a range before adding a block would exceed 128 KiB of write records (1,820
records) or its configured estimated range-byte budget. The next block remains queued for the
next range. An individually oversized block is rejected explicitly without truncation. Chunked
publication is required before supporting arbitrary larger private blocks; this version does not
claim that capacity. The general codec/contract reject payloads over 4 MiB as an allocation bound.
Claim transaction gas includes its data size and per-record checks.

This changes the projection genesis, claim selector and accepted wire version. Start a fresh
projection with matching Go/Rust binaries and contracts; there is no live-chain migration here.
Version-1 fixture files remain as historical data, and the current version-2 corpus is separate.

This change publishes sequencer writes. It does not implement an independent private execution
prover, forced-operation read/write conflict consumption, operator reconciliation after such an
operation, or recovery of withheld values. The existing proof-carrying event export remains an
already-attested historical-event transport. A successful write-publication test does not establish
that an arbitrary new withdrawal can be proved during an operator outage.
