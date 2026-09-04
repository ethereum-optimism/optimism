# Reviewing derivation batch decoding and validation

This area guide covers batcher-controlled data between complete channel bytes and accepted L2 batch inputs.
It pairs with the [`derivation-batch-reviewer`](../../.claude/agents/derivation-batch-reviewer.md) agent.

Read [spec-driven-review.md](spec-driven-review.md) in full before this guide.
Read [derivation.md](derivation.md) for the pipeline model and general derivation rules.

This guide owns derivation scope, code navigation, domain checks, and calibration cases.
The shared guide owns the review process, evidence contract, output, and mapping validation.
Neither guide defines protocol behavior.

## Scope

Review these behaviors:

- Decompression of complete channel data and enforcement of decompressed size limits.
- Batch type dispatch and singular-batch RLP decoding.
- Span-batch prefix, payload, bitlist, transaction, and varint decoding.
- Conversion from a raw span batch into derived block inputs.
- Holocene span decomposition and singular-batch streaming.
- Singular and span semantic validation.
- Fork-gated transaction types, compression formats, limits, and validation rules.
- Batch overlap checks against the safe chain.
- The resulting accept, drop, past, future, undecided, or error outcome.

Do not review these areas unless changed behavior crosses their boundary:

- Batcher transaction retrieval, frame parsing, or channel assembly.
- Deposit derivation and payload-attributes construction.
- Execution-layer transaction processing.
- General safe-head advancement or pipeline resets.
- Batcher encoding policy and compression efficiency.

Encoding code remains useful for round-trip and canonicality checks.
Do not report encoder-only findings unless they change data that a verifier accepts.

## Specification-to-code map

This mapping belongs in the monorepo because repository paths change independently from protocol behavior.
Treat each path as a starting point, then follow affected callers and callees.

| Behavior | Specification | op-node | Kona |
| --- | --- | --- | --- |
| Channel decompression and size bounds | [Channel format](https://specs.optimism.io/protocol/derivation.html#channel-format), [Fjord derivation](https://specs.optimism.io/protocol/fjord/derivation.html) | `op-node/rollup/derive/channel_in_reader.go`, `op-node/rollup/derive/channel.go`, `op-node/rollup/chain_spec.go` | `derive/src/stages/channel/channel_reader.rs`, `protocol/src/batch/reader.rs`, `protocol/src/brotli.rs` |
| Batch envelope and singular batches | [Batch format](https://specs.optimism.io/protocol/derivation.html#batch-format) | `op-node/rollup/derive/channel_in_reader.go`, `op-node/rollup/derive/batch.go`, `op-node/rollup/derive/singular_batch.go` | `derive/src/stages/channel/channel_reader.rs`, `protocol/src/batch/reader.rs`, `protocol/src/batch/core.rs`, `protocol/src/batch/single.rs`, `protocol/src/batch/type.rs`, `protocol/src/batch/errors.rs` |
| Span-batch wire format | [Span batch format](https://specs.optimism.io/protocol/delta/span-batches.html#span-batch-format) | `op-node/rollup/derive/span_batch.go`, `op-node/rollup/derive/span_batch_tx.go`, `op-node/rollup/derive/span_batch_txs.go`, `op-node/rollup/derive/span_batch_util.go` | `protocol/src/batch/raw.rs`, `protocol/src/batch/prefix.rs`, `protocol/src/batch/payload.rs`, `protocol/src/batch/transactions.rs`, `protocol/src/batch/tx_data/`, `protocol/src/batch/varint.rs`, `protocol/src/batch/bits.rs` |
| Span conversion and overlap | [Span batch integration](https://specs.optimism.io/protocol/delta/span-batches.html#integration) | `op-node/rollup/derive/span_batch.go`, `op-node/rollup/derive/batches.go` | `protocol/src/batch/raw.rs`, `protocol/src/batch/span.rs`, `protocol/src/batch/element.rs`, `protocol/src/batch/inclusion.rs` |
| Holocene batch streaming | [Holocene derivation](https://specs.optimism.io/protocol/holocene/derivation.html) | `op-node/rollup/derive/batch_mux.go`, `op-node/rollup/derive/base_batch_stage.go`, `op-node/rollup/derive/batch_stage.go`, `op-node/rollup/derive/batch_queue.go`, `op-node/rollup/derive/attributes_queue.go` | `derive/src/pipeline/core.rs`, `derive/src/stages/attributes_queue.rs`, `derive/src/stages/batch/batch_provider.rs`, `derive/src/stages/batch/batch_stream.rs`, `derive/src/stages/batch/batch_queue.rs`, `derive/src/stages/batch/batch_validator.rs`, `rust/kona/crates/node/service/src/actors/engine/actor.rs`, `rust/kona/crates/proof/driver/src/core.rs` |
| Semantic batch validity | [Batch Queue](https://specs.optimism.io/protocol/derivation.html#batch-queue), [Span Batch Queue](https://specs.optimism.io/protocol/delta/span-batches.html#batch-queue) | `op-node/rollup/derive/batches.go`, `op-node/rollup/chain_spec.go`, `op-node/rollup/toggles.go`, `op-node/rollup/types.go` | `protocol/src/batch/validity.rs`, `protocol/src/batch/single.rs`, `protocol/src/batch/span.rs`, `derive/src/stages/batch/batch_validator.rs`, `genesis/src/rollup.rs` |

Kona paths are relative to `rust/kona/crates/protocol/` unless they start with `rust/kona/`.
All other paths are relative to the repository root.

Fork documents amend the base rules.
Search all files under each applicable `specs/protocol/<fork>/` directory for touched concepts.
Do not limit this search to `derivation.md`.
For example, Delta uses `span-batches.md`, and Lagoon uses `post-exec.md` for transaction acceptance.
At minimum, check Delta, Fjord, Holocene, and any later fork that changes transaction acceptance.

## When to run this reviewer

Run this reviewer when a change touches any mapped path or its dependencies.
Also run it for these changes:

- A Kona client release that contains mapped changes.
- A rollup configuration field used by mapped code.
- A new batch format, compression format, transaction type, or derivation fork gate.
- A protocol change that modifies a mapped specification section.
- A fix for malformed batch data, decoder panics, or cross-client disagreement.

## Derivation analysis

Trace complete channel bytes through the final batch outcome.
Trace op-node and Kona independently.
For each affected path, determine:

- Which byte strings it accepts.
- Which decoded values it produces.
- Which limit it applies before allocation or iteration.
- Which fork condition it uses.
- Which semantic validity outcome it returns.
- Whether partial state survives an error.

Compare both implementations with the specification first.
Then compare their accept sets and outputs with each other.
Follow [derivation.md](derivation.md#cross-client-wire-format-parity) before proposing either behavior as correct.

## Adversarial boundaries

Batcher-provided bytes are adversarial input.
Check zero, one, maximum, and one-past-maximum values where the format permits them.

Also check:

- Truncation before every fixed-width and variable-width field ends.
- Declared counts larger than the remaining input.
- Integer conversion, addition, subtraction, and multiplication boundaries.
- Empty batches and zero transaction counts.
- Bitlists whose byte length does not match their element count.
- Non-minimal but valid protobuf `uvarint` encodings.
- Unknown batch and transaction type bytes.
- Transaction type prefixes through decode and reassembly.
- Trailing bytes when the surrounding format permits another batch.
- Span batches that cross a fork boundary.
- Spans that overlap the safe chain partially or completely.
- Legacy and Holocene processing of the same wire format.

## Defensive decoder guidance

This guidance helps the reviewer assess an implementation.
It does not add a protocol invariant to the specification.

The specification normally defines valid results and invalid-input outcomes.
It need not name language failures such as Rust panics or Go slice-bound faults.
An implementation cannot produce its required outcome if adversarial input terminates that path first.

Treat these operations as review leads:

- Unchecked indexing, slicing, `split_at`, cursor advances, or fixed-width conversions.
- `unwrap`, `expect`, `panic`, or assertions reachable from batcher-controlled data.
- Length-derived allocation or loops before the relevant protocol limit is enforced.
- Arithmetic on decoded counts before checked conversion and bounds validation.

Do not report the operation alone.
Show concrete input, reachability, the failing operation, and the missing guard.

Classify the result from its evidence:

- If the specification defines an outcome, premature termination is a specification violation.
- If the specification is silent, report an implementation-safety finding.
- Report a specification gap only when two consensus outcomes remain plausible.

Do not demand a specification sentence that says an implementation must not panic.
Protocol authors can add such guidance later if they want an explicit totality requirement.

## Derivation dismissal checks

Try to dismiss each decoder candidate with these checks:

- An earlier length guard makes the operation unreachable.
- A decoded count is bounded before conversion or allocation.
- Callers prove the internal invariant used by the suspected panic.
- The path is encoder-only, test-only, or outside consensus derivation.
- The difference changes only an internal error name.
- A fork condition makes the compared behaviors inapplicable.
- The specification explicitly permits both outcomes.

## Calibration cases

Use historical fixes to test the reviewer method, not as protocol authority.
A useful reviewer should identify each original failure without seeing the fix:

- `ethereum-optimism/optimism#19361`: truncated fixed-width span fields reached unchecked slices.
- `ethereum-optimism/optimism#20000`: an unknown batch type reached a panic.
- `ethereum-optimism/optimism#22126`: Kona rejected valid non-minimal protobuf `uvarint` encodings.
- `ethereum-optimism/optimism#21808`: span decoding lost transaction type prefixes.

Also test recent clean changes.
A reviewer that reports every unusual decoder operation has not met the shared evidence contract.
