# Sequencer batches and ordinary deposits

The public projection advances under normal OP derivation when the private operator
is offline. Its fallback blocks contain the required system and ordinary deposit
transactions, but no synthetic private message replay transactions or new private
range claims. Private commitments advance when the sequencer publishes another
accepted range. This does not provide independent private execution during an outage.

There is one ordinary portal deposit path. The special proof-carrying projection
exporter, verifier hooks and reserved exporter address are removed. The general L1
event oracle remains a separate contract feature; its ordinary export route uses
the standard cross-domain messengers so failed L1 registration can be retried.

The batcher skips projection positions that are already canonical and publishes a
claim at the beginning of each new range, followed by its synthetic message replay
transactions. A range may cover later projection positions whose private blocks have
already executed. It cannot commit unknown future L1 deposits. Claims publish private
terminal block and parent hashes, L1/configuration bindings and a private input hash.
There are no published write sets, key/value commitments or private-writes RPC.

The private input hash covers stock span-batch frames, which exclude deposit payloads.
Canonical L1 headers, receipts, configuration and block origins provide those deposits
through ordinary derivation. An authentic private terminal block hash commits the
executed deposits through transaction roots and ancestor headers. Today the registry
accepts an operator attestation: it does not verify private execution or deposit
completeness. The explicit `insecure-stub-v1` derivation verifier accepts bounded
dummy proof bytes; the registry retains authorization and range accounting. A future execution verifier must
bind the prior private checkpoint, the canonical deposit history, resulting private
state and published message outputs; the input hash alone is insufficient.

[Recovery](RECOVERY.md) remains in place for private reconciliation after fallback or
cross-chain invalidation. Removing write publication and the special exporter does
not remove that mechanism. Existing interop message validation also retains its normal
transaction access-list checks, which are separate from the removed private write sets.

## Whole-range admission before execution

The projection rollup config opts into `private_projection` with verifier
`insecure-stub-v1` and, optionally, `allow_events`. This profile is for fresh chains
with Holocene and interop active at genesis. Ordinary chains omit the field and
retain their existing admission rules. It is consensus configuration, not a runtime
choice of an RPC verifier.

After channel decompression and span decoding, Go's `checkSpanBatchHolocene` and
Kona's `SpanBatch::check_batch_holocene` authenticate the parent and canonical overlap,
then invoke `projection.ValidateProjectionRange` / `validate_projection_range` before
releasing the first singular batch. The batch producer uses the same Go function.
Validation belongs here rather than in byte decoding because parent/overlap context
must already be authenticated. No execution or private-node lookup is required.

The shared validator is a pure function of config, the complete decoded span, its
chain/genesis/interval/parent context, and a deterministic proof verifier. No clock,
RPC, database, mutable cursor, or cross-call cache is consulted. The function:

- Requires 1–65,536 consecutive blocks, aligned to the configured block interval,
  with overflow-safe heights. A range cannot include genesis.
- Requires exactly one canonical `postClaim` call: transaction zero of block zero.
  Its inclusive first/last heights must cover exactly this span. Missing, duplicated,
  misplaced or truncated claims reject the entire range. Empty later blocks are allowed.
- Accepts only canonically encoded, validly signed EIP-1559 transactions for this
  chain, with nonzero gas at most 16,777,216, zero value and zero fee caps; no contract
  creation. Deposit envelopes and every other transaction type are rejected.
- Allows only the ClaimRegistry `postClaim`, messenger `replaySentMessage`, inbox
  `validateMessage`, and explicitly enabled EventReplayer `replayEvent` destinations
  and selectors. ABI round trips must be exact: no trailing bytes, dirty padding,
  alternate offsets or malformed calls. Messages are bounded to 1 MiB, proofs to
  64 KiB and generic logs to four topics. Native ETH bridge replay remains forbidden.
- Requires empty access lists except the exact canonical inbox checksum access list
  for an import, including full-width chain IDs.
- Constructs a proof statement from the actual public records, then calls the
  proof verifier once. The statement contains the chain ID, authenticated span parent,
  claim fields and an ordered transcript commitment. The transcript binds block
  heights/timestamps/origin numbers and transaction sender, nonce, gas, destination
  and canonical calldata. Import access lists are uniquely determined by that data.
  Claim proof bytes are normalized to empty and signature bytes are excluded to
  avoid circular dependence on the proof carried by the claim transaction.

A final preflight reuses ordinary singular admission checks for the unexecuted suffix,
including origin timestamps, drift and upgrade-block restrictions. This prevents a
late deterministic schedule failure from releasing an earlier valid prefix. Future
execution hashes are not predicted; execution still checks actual ancestry and state.

Structural or proof failure drops the span and flushes its channel, without executing
any new part of the range. Submitted singular batches cannot bypass this gate.
Protocol-generated fallback blocks remain allowed. L1-derived deposits, L1-info and
upgrade transactions are added later by ordinary attributes derivation, not trusted
as operator-supplied batch records. The existing projection deposit execution rule
still applies. No range cursor is persisted: reset/reorg clears normal batch buffers,
and replay validates again against the new canonical parent and overlap. Registry
non-overlap checks and claim-follower receipt/safety/recovery checks remain in place.

This is **atomic structural admission**, not atomic EVM execution or cross-chain
invalidation. A later EVM revert, nonce/state failure, or invalid interop dependency
still follows the ordinary execution/replacement rules. The current stub proves no
correspondence with private execution, no private data availability, and no private
checkpoint continuity. A real proof system needs a versioned statement binding the
prior private checkpoint and canonical deposit history; the claim's terminal-parent
hash is only the immediate parent of its terminal block, not that checkpoint.

Go and Kona consume the same acceptance vectors, including malformed late-block
records and proof-statement hashes. Kona node and fault-proof derivation share this
admission path. This does not establish complete fault-proof execution support for
the private projection's custom execution rules.

The default renderer's conservative gas formula limits export messages to 581,329
bytes despite the larger wire allocation bound. Larger messages accepted privately
can stall publication; private application admission and improved gas budgeting
remain separate work.

## Continuation after partial invalidation: proof-system requirements

This section specifies requirements for future cryptographic continuity. It does
not choose a commitment scheme or describe an implemented proof guarantee.

Suppose a submitted range covers 1–5 and interop replaces 3–5 with 3′–5′. A later
range 6–10 must extend canonical public 5′. It must prove private execution from
the private state corresponding to 5′, not the old submitted range's endpoint 5.
The public projection state root is never a substitute for a private state root.
The previous submitted claim is not necessarily the canonical continuation anchor.

There are three distinct guarantees:

| Layer | Current guarantee |
| --- | --- |
| Structural admission | The entire submitted span has allowed public records and range framing; its parent and canonical overlap are checked before new blocks are emitted. The proof statement includes the full public parent hash. |
| Trusted private recovery | LightCL authenticates a surviving private prefix against operator claim-bound header ancestry, then locally executes canonical deposit-only replacements. It normally completes the reserved interval before publication resumes. |
| Cryptographic private continuity | Not implemented. Neither the stub verifier nor successful local recovery publicly proves the replacement private state. |

The intended interface remains pure:

```text
resolveParentContext(canonicalDerivationHistory, candidateRange)
    -> Ready(authenticatedContext) | Unavailable(reason) | Invalid(reason)
validateSpan(config, authenticatedContext, decodedSpan, proof)
    -> Valid(statement) | Invalid(reason)
```

The resolver is outside the pure function. Context must identify the canonical
projection parent immediately before the whole submitted range, including its
height/hash and chain/config domain. It must also provide either an independently
authenticated private starting commitment or an authenticated earlier private anchor
and the canonical replacement inputs from which the proof establishes that start.
A prover-supplied private root alone is not authenticated context. The attested
profile must remain explicitly distinct from a future proven profile; a missing
private commitment cannot silently fall back to the stub.

Per-block private commitments authenticated by the original range proof could make
the surviving anchor at block 2 available. They do not authenticate replacement
states 3′–5′. A recovery proof could authenticate that transition; alternatively it
could start at an older proven anchor and cover the surviving prefix as well as
replacement inputs. The recovery segment may be included in the next span proof.
These are design options, not a selected wire format or implemented circuit.

Protocol-generated replacements must remain derivable without a new publisher
proof. Private correspondence for them can be established later, for example by
that recovery segment before accepting the next proven span. A design requiring a
pre-existing proven private root at 5′, while only the next proof can establish it,
would deadlock; an earlier authenticated anchor avoids that circular requirement.

For overlapping submissions, resolve the parent before the original range, validate
the complete proof, compare the overlapping public records/origins with canonical
history, then emit only the suffix. A span matching abandoned history is invalid;
one matching surviving canonical overlap remains eligible. Never choose a parent
from the local unsafe tip or from the most recently received claim.

A known mismatch or absent required proof material is an invalid submission and
rejects the whole span. A temporary failure to obtain already committed canonical
context is different. **Current Holocene handling consumes and skips an undecided
span; it does not retain it for retry.** Before adding context-dependent proof
verification, the pipeline must retain a bounded pending candidate with its original
L1 inclusion context, retry context resolution, and invalidate/re-resolve that work
on reset or reorg. It must not flush the candidate as invalid, emit its prefix, or
substitute unsafe state. Required proof material must be publicly derivable from
committed data; an unavailable private RPC or a future publisher action cannot
become a prerequisite for advancing protocol-generated replacement history.

No pending-context mechanism or private commitment scheme is introduced by the
current structural gate. Go/Kona parity and fault-proof replay must cover any such
future change, including partial invalidation, canonical overlap, unavailable-context
retry, and restart/reorg while a candidate is pending.

## Development migration

Start a fresh deployment with matching contracts, Go services and op-reth. This rollback
restores claim wire version 1 and removes the exporter predeploy, changing projection
genesis. ClaimRegistry 2.1.0 and the explicit projection admission config also require
a fresh genesis; do not hot-swap this rule into existing deployments.
Existing experimental version-3 deployments cannot adopt this as a live upgrade.
