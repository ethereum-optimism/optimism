# Private write publication

Claim wire version 3 publishes opaque write records in the leading claim transaction of each
sequencer range. This supersedes the older version-1 wire and the draft version-2 stable-tag scheme.
The execution proof
mode remains **attested**: the batcher's signature authenticates the records, while their
completeness and correctness remain an operator assertion. A future batch execution verifier
must bind and verify the entire write set as well as the existing claim fields.

## Records

Each record is 72 bytes: a 32-byte tag, a 32-byte value commitment, and an unsigned 8-byte
big-endian private block number. Records are strictly sorted by tag without duplicates.
The private RPC uses stable identifiers for aggregation. For fixed-width fields:

```
privateTag = keccak256("optimism.private-interop.write-key.v1" || uint64be(chainID)
                || uint8(kind) || address20 || slot32)
privateValueCommitment = keccak256("optimism.private-interop.write-value.v1" || privateTag || value32)

bounds = uint64be(claim.firstBlock) || uint64be(claim.lastBlock)
tag = keccak256("optimism.private-interop.range-key.v1" || bounds || privateTag)
valueCommitment = keccak256("optimism.private-interop.range-value.v1" || bounds || privateValueCommitment)
```

Only the range-scoped tag and value commitment are published. The builder applies this transform
after aggregating by stable private key, then sorts the public tags again. A user who knows a key
can derive its lookup tag for each range. Equal keys and equal values have different public hashes
in different ranges; a batch does not reveal how many times a key changed within that range.
Range bounds are public salts, not secrets. Counts, last-write block numbers and other message
metadata remain visible, and keys that can be guessed can still be followed across ranges.
The same bounds on competing forks reuse the scope; this is not a fork-unlinkability mechanism.

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

The claim's ABI adds `bytes writes` after `bytes proof`, and its version is 3. `ClaimRegistry`
checks record shape, ordering and block bounds. Its running claim hash includes every write byte.
It emits no logs, preserving the original projected message positions. The calldata rides inside
normal sequencer batches to L1. Consumers obtain the write data from those batches or the derived
projection transactions, not from an unavailable sidecar commitment.

`codec.CheckUnchanged` takes stable private dependency tags and derives the lookup tags separately
for every range. It checks a complete sequence of already-authenticated claims between range
checkpoints. It refuses missing ranges; a forward gap allowed by the registry is not evidence of
no private writes. It must not be fed merely decoded, unverified operator data. Callers still
need membership evidence for the original values. Value commitments let a user authenticate a
candidate update but do not supply that value or a refreshed Merkle branch. Value commitments and
per-key block numbers are optional for freshness alone: a complete set of range-scoped tags is
sufficient to conservatively invalidate an old witness. They are retained to support checking
candidate new values and ordering resets.

## Outage recovery

The private LightCL retains normal L1 confirmation settings. Its claimed follow endpoint
provides a private checkpoint and the projection's fully scanned local-safe frontier.
After fallback or replacement, the LightCL pauses sequencing and uses the ordinary
attributes handler to execute canonical deposit-only inputs against private state. It
checks the projection's timestamp, L1 origin and sequence number; projection hashes
identify inputs and are never used as private forkchoice hashes.

Local-safe execution, cross-safety and finality remain separate. The adapter maps the
projection's safety frontiers through the authenticated private ancestry. A projection
reorg revokes affected private checkpoints; finalized private history cannot be revoked.
For a claim whose carrier survives but whose suffix is replaced, the original private
terminal commitment authenticates the surviving prefix by hash-linked ancestry. The
supernode's durable deny list and retained denied headers reconstruct that information
after restart. Missing headers or private state stop recovery.

Private batchers require `--private-interop.public-projection-rollup-rpc` alongside the
projection execution RPC. The publication cursor uses the projection's local-safe head
and waits until private canonical execution has reconciled it. A surviving claim carrier
still reserves its original range, so recovery executes actual canonical replacements
through that range before publication resumes. If canonical derivation drops the
remaining span, those positions wait for ordinary sequencing-window expiry. Range publication currently waits for the
previous projection parent. Catch-up therefore requires enough blocks per range to
outpace L1 inclusion plus follow polling; undersized ranges can remain at the expiry
frontier even when reconciliation is correct. The outage test retains an eight-block
catch-up limit and uses the devstack/op-up default of twelve-block ranges with its
accelerated L1.

An invalidated range's aggregate write records do not provide freshness coverage for its
surviving prefix: aggregation may have discarded earlier writes to a key. Missing or
invalidated coverage cannot prove that a user's state was unchanged. This mechanism also
does not recover withheld values or prove a new forced withdrawal's state transition.

Recovery consumes replacements produced by canonical derivation and cross-safety checks.
A reverted ClaimRegistry call alone does not invalidate its block or suppress separate
replay calls; adding a new proof verifier requires a consensus rule or atomic replay
authorization that makes invalid proof handling effective. This adapter does not provide
that rule.

## Capacity and activation

Production closes a range before adding a block would exceed 128 KiB of write records (1,820
records) or its configured estimated range-byte budget. The next block remains queued for the
next range. An individually oversized block is rejected explicitly without truncation. Chunked
publication is required before supporting arbitrary larger private blocks; this version does not
claim that capacity. The general codec/contract reject payloads over 4 MiB as an allocation bound.
Claim transaction gas includes its data size and per-record checks.

This changes the projection genesis and accepted wire version. The selector changed when upgrading
from version 1 to version 2; version 3 retains version 2's ABI shape and selector. Start a fresh
projection with matching Go/Rust binaries and contracts; there is no live-chain migration here.
Version-1 and version-2 fixture files remain as historical data; the current corpus is version 3.
Version 2 must be rejected: interpreting stable tags as range-scoped tags (or vice versa) could
silently report changed state as unchanged. Discard earlier development history and start fresh.

This change publishes sequencer writes. It does not implement an independent private execution
prover, forced-operation read/write conflict consumption, operator reconciliation after such an
operation, or recovery of withheld values. The existing proof-carrying event export remains an
already-attested historical-event transport. A successful write-publication test does not establish
that an arbitrary new withdrawal can be proved during an operator outage.
