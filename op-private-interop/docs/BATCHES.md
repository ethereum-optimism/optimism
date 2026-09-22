# Projection batch admission and private checkpoints

The public projection derives from ordinary L1 channel frames. Published spans carry
private output commitments and message records, not private application transactions
or write sets. Ordinary L1 deposits remain public. On the projection they follow its
existing deposit execution policy; the private chain executes them normally.

This is an experimental, fresh-deployment profile. The only configured verifier is
`insecure-stub-v1`: bounded dummy proof bytes succeed without cryptographic execution
verification. Commitments therefore remain assertions of the authorized L1 publisher.
Neither an output root nor a successful local recovery proves private execution.

## Submitted block layout

Each submitted span has exactly one leading `postClaim` transaction. Its first block
then contains `recordOutput(bytes32)`, followed by zero or more message replay calls.
Every subsequent block starts with exactly one `recordOutput`, followed by replays.
Thus an application-empty submitted block still has an output record. The first
block has both a claim and an output record. Missing, repeated, late or malformed
records reject the entire submitted span before its first new block is released.

The root is the standard private OP OutputV0 commitment:

```text
keccak256(versionZero || privateStateRoot || privateMessagePasserStorageRoot || privateBlockHash)
```

The publisher obtains the message-passer storage root from the private payload's
post-Isthmus `withdrawalsRoot`. The source configuration must activate Isthmus at
genesis. Projection configuration includes the independently computed private
genesis output root as the initial checkpoint. The public projection's own state
root is never used as a private commitment.

Both record methods emit no logs. `recordOutput` deliberately stores nothing: its
canonical admitted calldata is the durable record. Claims in this profile also
remain readable when their EVM call reverts; registry storage and receipt success
are not a second admission gate. Legacy claim-follow mode retains its receipt
policy. Protocol-generated deposits/PostExec transactions cannot create checkpoints,
even if their destination or calldata impersonates a record method.

## Pure admission and proof statement

The rollup config opts into `private_projection` with verifier `insecure-stub-v1`,
a nonzero `genesis_output_root`, and optional `allow_events`. Holocene and interop
must be active at genesis. Ordinary chains omit this config and retain their rules.

After decompression and decoding, Go's `checkSpanBatchHolocene` and Kona's
`SpanBatch::check_batch_holocene_with_context` check range geometry, authenticate the
original span parent and overlap, resolve continuation context, and call
`projection.ValidateProjectionRange` / `validate_projection_range`. This happens
before decomposition releases any new singular batch. The producer uses the same
Go validator. Byte decoding alone cannot authenticate a canonical parent.

```text
resolveContext(canonicalProjectionHistory, originalSpanParent, genesis)
    -> Ready(context) | Unavailable | Invalid
validateSpan(config, context, completeDecodedSpan, pureVerifier)
    -> Valid(statement) | Invalid
```

The validator performs no RPC, clock, database or mutable-cache access. Its input
includes chain ID, block interval, genesis geometry, full canonical public parent
hash and an independently resolved continuation. It enforces:

- 1–65,536 aligned consecutive blocks, excluding genesis, with overflow-safe heights.
- One version-2 claim whose first/last heights cover exactly the entire span.
- The fixed per-block output placement described above, with nonzero roots.
- Canonical, validly signed EIP-1559 envelopes for this chain; zero value and fee caps;
  nonzero gas at most 16,777,216; no creation or operator-supplied deposit envelopes.
- Only `postClaim`, `recordOutput`, `replaySentMessage`, `validateMessage`, and enabled
  `replayEvent` at their designated predeploys. Native bridge replay is forbidden.
- Exact ABI encoding, at most 64 KiB proof bytes, 1 MiB message bytes and four event
  topics; no trailing data, dirty padding or alternate offsets.
- Empty access lists except the exact inbox checksum list derived from each import.
- Claim anchor height/root/recovery hash equal to the independently resolved context.
  Without a recovery interval, the claimed private parent output must equal the
  checkpoint at the public parent, and the recovery hash must be zero.

The proof statement contains the chain ID, full public parent hash, continuation,
normalized claim and a Merkle commitment to every ordered block record. Each leaf is
`H(0x00 || blockTranscript)`. The transcript uses big-endian uint64 heights, timestamps,
L1-origin numbers and transaction counts, then each sender, nonce, gas, destination,
calldata length and canonical calldata. Proof bytes are normalized to empty; signature
bytes are excluded to avoid circularity, while the recovered sender remains bound.
Internal nodes are `H(0x01 || left || right)`, duplicating the last node at odd levels.
The final root is `H("optimism.private-projection.v2\0" || uint64be(leafCount) || treeRoot)`.
The count prevents duplicated leaves from aliasing spans of different lengths.

A final preflight checks ordinary singular scheduling rules over the unexecuted
suffix, including L1-origin timestamps, drift and upgrade blocks. For drift only,
strict checkpoint metadata counts as application-empty; any replay still counts as
nonempty. All empty-batch next-origin timing checks remain enforced. Fork activation
blocks still prohibit submitted transactions: this fresh interop-at-genesis profile
must not assume a future upgrade can be crossed with a published span. Such spans
are rejected and require expiry/recovery, or a separately specified upgrade policy. Structural/proof
failure drops the span and flushes its channel. Submitted singular wire batches
cannot bypass this gate. This is whole-span structural admission, not atomic EVM
execution: a transaction revert does not by itself replace a block. Payload-invalid
execution and invalid interop dependencies retain their ordinary handling.

## Continuation after invalidation or sequencing-window expiry

For a span 1–5 followed by replacement of 3–5 with 3′–5′, the next span 6–10 extends
canonical public 5′. Its surviving private checkpoint is the output record at 2,
not the old submitted endpoint at 5. The claim carries:

- `anchorBlock` and `anchorOutputRoot`: the surviving checkpoint.
- `recoveryHash`: a commitment to the canonical replacement inputs from the
  publication parent back to, but excluding, the anchor.
- `parentOutputRoot`: the private output after executing that recovery interval.

The resolver walks hash-linked canonical projection history backward from the
**parent of the original full span**, including when part of the span overlaps safe
history. It stops at the nearest admitted output record, or at the configured
private genesis output. A block containing sequencer transactions but no valid
output record is invalid context, never silently classified as fallback.

Recovery hashes fold blocks in descending height order, beginning with zero:

```text
H("optimism.private-recovery.v1\0" || previousHash || publicBlockHash || publicParentHash
  || uint64be(number) || uint64be(timestamp) || uint64be(transactionCount)
  || each(uint64be(transactionLength) || rawTransaction))
```

The public block hash additionally binds the complete header. A future execution
proof must authenticate private execution from the surviving output through these
canonical replacement inputs, then through the submitted private range, and bind
all intermediate output records and exported/imported messages. The present stub
only checks the public framing and context bindings; it does **not** verify that the
claimed private parent or intermediate outputs follow from execution. Configuration
and dependency-set claims also remain trusted until the proof relation authenticates
them against its protocol configuration.

Protocol replacements need no publisher proof or private RPC to be derived. Their
private-state correspondence is established later by the recovery segment of the
next real span proof. Deposits replayed during recovery can emit private messages;
those messages are not retroactively exported into deposit-only public replacements.
Requiring a proven replacement root before that recovery proof would deadlock.

## Missing context, overlap and resets

The stateful context collector is outside the pure validator. Each attempt pins the
canonical parent hash, then performs at most 128 backward reads. One constant-sized
cursor persists across attempts; there is no consensus maximum outage length.
Unavailable committed data or unfinished scanning retains one candidate and its
original L1 inclusion block, returning a temporary error. Every decomposed block
retains that inclusion block for its sequencing-window checks. The pipeline does not
flush, emit a prefix, substitute unsafe history, or read a new candidate while waiting.

A changed target parent restarts collection. Reset/flush discards both candidate and
cursor. Malformed context rejects the candidate. Ordinary missing L1 scheduling
information retains existing Holocene handling so waiting cannot prevent L1 traversal.
Valid overlapping public records are checked against canonical history and only the
new suffix is emitted. An overlap matching abandoned history is rejected.

[LightCL recovery](RECOVERY.md) uses a surviving root to identify a matching local
private checkpoint, preserves ancestry/finality and restart safeguards, executes the
canonical replacement interval, and resumes sequencing after the reserved range.

## Verification and deployment limits

Go and Kona share admission/statement vectors and a canonical recovery transcript
vector. Tests cover missing context, long incremental recovery, reorgs, reset, original
inclusion retention, malformed records and changed intermediate roots. Node and
fault-proof derivation use the same Kona gate. Kona's stateless executor and op-reth
share the authenticated-system/user-deposit classifier: public user deposits are
inert, while private execution applies them normally. Both attribute builders keep
projection publication fee-free after L1 SystemConfig updates (zero fee scalars,
operator fees and minimum base fee; maximum gas limit). Batcher authorization
continues to follow L1 updates.

The private super-root lifecycle test exercises native range/consolidation verification,
a challenged correct projection root, and rejection of a corrupted projection root
through the ZK dispute-game mock-verifier path. This is not a cryptographic recursive
super-root proof. The separate experimental SP1 private-execution relation supports
native execution, compiled guest execution and local CPU core proving; see
[`rust/kona/sp1/README.md`](../../../rust/kona/sp1/README.md#experimental-private-projection-relation).
Its journal binds independently authenticated canonical context and private execution;
it does not replace the network's explicit stub verifier or provide an RPC witness collector.

The renderer's conservative gas formula limits export messages to 581,329 bytes,
below the structural 1 MiB wire bound. Larger private messages can stall publication;
private admission and improved gas budgeting remain separate work.

Use a fresh deployment with matching contracts, rollup config, Go services and op-reth.
Claim wire version 2, ClaimRegistry 3.0.0 and the output-root config change projection
genesis and admission rules. This is not a live upgrade for existing private devnets.
