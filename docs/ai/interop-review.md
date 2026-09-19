# Reviewing interop verification and safety

This area guide covers protocol-visible interop verification from message extraction through published safety results.
It pairs with the [`interop-reviewer`](../../.claude/agents/interop-reviewer.md) agent.

Read [spec-driven-review.md](spec-driven-review.md) in full before this guide.
Read [derivation.md](derivation.md) and [fault-proofs.md](fault-proofs.md) for surrounding system context.

This guide owns interop scope, code navigation, and domain checks.
The shared guide owns the review process, evidence contract, output, and mapping validation.
Neither guide defines protocol behavior.

## Scope

Review these behaviors:

- Initiating-message and executing-message extraction.
- Message identifier and payload matching.
- Dependency-set membership, ordering, timing, and expiry checks.
- Message graph construction and same-timestamp cycle detection.
- Safe and cross-unsafe safety evaluation.
- Invalid-head selection, invalidation, and rewind.
- Deposit-only replacement and recursive dependent replacement.
- Lagoon activation and dependency-set upgrade handling.

Do not review these areas unless changed behavior crosses their boundary:

- Interop predeploy contract implementation.
- Bridge or messenger business logic.
- General L2 derivation and execution.
- Transaction-pool and sequencer policy.
- RPC transport and API presentation.
- Metrics, alerts, and monitor-only code.
- Database performance and storage layout.
- General fault-game mechanics.

Persistence code remains relevant when it changes graph inputs or published safety results.

## Specification-to-code map

This mapping belongs in the monorepo because repository paths change independently from protocol behavior.
Treat each path as a starting point, then follow affected callers and callees.
All paths are relative to the repository root.

| Behavior | Specification | Go | Kona |
| --- | --- | --- | --- |
| Message extraction and declarations | [Messaging](https://specs.optimism.io/interop/messaging.html), [Predeploys](https://specs.optimism.io/interop/predeploys.html) | `op-core/interop/messages/logs.go`, `op-core/interop/messages/messages.go` | `rust/kona/crates/protocol/interop/src/message.rs`, `rust/kona/crates/protocol/interop/src/graph.rs` |
| Dependency, timing, and expiry rules | [Messaging](https://specs.optimism.io/interop/messaging.html), [Dependency set](https://specs.optimism.io/interop/dependency-set.html), [Interop derivation](https://specs.optimism.io/interop/derivation.html) | `op-core/interop/depset/depset.go`, `op-core/interop/depset/static_depset.go`, `op-core/interop/depset/json.go`, `op-core/superchain/types.go`, `op-core/superchain/chain.go`, `op-node/superchain/depset.go`, `op-node/service.go`, `op-node/config/config.go`, `op-supernode/cmd/depset.go`, `op-supernode/supernode/supernode.go`, `op-supernode/supernode/activity/interop/algo.go`, `op-interop-filter/filter/config.go`, `op-interop-filter/filter/service.go`, `op-interop-filter/filter/lockstep_cross_validator.go` | `rust/kona/crates/protocol/genesis/src/rollup.rs`, `rust/kona/crates/protocol/genesis/src/chain/config.rs`, `rust/kona/crates/protocol/genesis/src/interop/constants.rs`, `rust/kona/crates/protocol/genesis/src/interop/config.rs`, `rust/kona/crates/protocol/genesis/src/interop/depset.rs`, `rust/kona/crates/protocol/genesis/src/interop/mod.rs`, `rust/kona/crates/protocol/registry/build.rs`, `rust/kona/crates/protocol/registry/src/lib.rs`, `rust/kona/crates/proof/proof-interop/src/boot.rs`, `rust/kona/crates/protocol/interop/src/graph.rs`, `rust/kona/crates/protocol/interop/src/rules.rs` |
| Message graph and cycle detection | [Messaging](https://specs.optimism.io/interop/messaging.html) | `op-supernode/supernode/activity/interop/logdb.go`, `op-supernode/supernode/activity/interop/log_backfill.go`, `op-supernode/supernode/activity/interop/raftwallogdb/db.go`, `op-supernode/supernode/activity/interop/verification_view.go`, `op-supernode/supernode/activity/interop/algo.go`, `op-supernode/supernode/activity/interop/cycle.go` | `rust/kona/crates/protocol/interop/src/graph.rs`, `rust/kona/crates/protocol/interop/src/rules.rs`, `rust/kona/crates/protocol/interop/src/traits.rs`, `rust/kona/crates/protocol/interop/src/errors.rs` |
| Safe promotion and invalidation | [Verifier](https://specs.optimism.io/interop/verifier.html), [Messaging](https://specs.optimism.io/interop/messaging.html) | `op-supernode/supernode/activity/interop/interop.go`, `op-supernode/supernode/activity/interop/checker.go`, `op-supernode/supernode/activity/interop/types.go`, `op-supernode/supernode/activity/interop/verified_db.go`, `op-supernode/supernode/activity/interop/reader.go`, `op-supernode/supernode/activity/superroot/superroot.go`, `op-supernode/supernode/chain_container/super_authority.go`, `op-supernode/supernode/chain_container/invalidation.go`, `op-supernode/supernode/chain_container/engine_controller/rewind.go`, `op-node/rollup/engine/engine_controller.go`, `op-node/rollup/engine/cross_safe_cache.go` | No current Kona production path publishes live safe results or invalidates live heads. |
| Cross-unsafe classification | [Verifier](https://specs.optimism.io/interop/verifier.html) | `op-interop-filter/filter/logsdb_chain_ingester.go`, `op-interop-filter/filter/lockstep_cross_validator.go`, `op-interop-filter/filter/backend.go`, `op-service/eth/safety/safety.go` | No current Kona production path publishes cross-unsafe safety. |
| Invalid-block replacement | [Interop derivation](https://specs.optimism.io/interop/derivation.html), [Super fault dispute game](https://specs.optimism.io/fault-proof/stage-one/super-fault-dispute-game.html), [Holocene derivation](https://specs.optimism.io/protocol/holocene/derivation.html) | `op-node/rollup/engine/payload_process.go`, `op-node/rollup/engine/build_invalid.go`, `op-node/rollup/engine/payload_success.go`, `op-node/rollup/derive/deriver.go`, `op-node/rollup/derive/pipeline.go`, `op-node/rollup/derive/attributes_queue.go`, `op-service/eth/types.go` | `rust/kona/crates/proof/proof-interop/src/provider.rs`, `rust/kona/crates/proof/proof-interop/src/consolidation.rs`, `rust/kona/bin/client/src/interop/mod.rs`, `rust/kona/bin/client/src/interop/consolidate.rs`, `rust/kona/sp1/programs/super-range/src/main.rs`, `rust/kona/sp1/programs/super-aggregation/src/main.rs` |
| Lagoon activation and upgrade handling | [Interop derivation](https://specs.optimism.io/interop/derivation.html), [L2 upgrades](https://specs.optimism.io/protocol/l2-upgrades-1-execution.html), [Superchain configuration](https://specs.optimism.io/protocol/superchain-config.html) | `op-node/rollup/toggles.go`, `op-node/rollup/types.go`, `op-node/rollup/derive/attributes.go`, `op-node/rollup/derive/lagoon_activation_transactions.go`, `op-node/rollup/derive/upgrade_transaction.go`, `op-node/rollup/derive/payload_util.go`, `op-node/rollup/derive/batches.go`, `op-node/rollup/sequencing/sequencer.go`, `op-supernode/supernode/supernode.go` | `rust/kona/crates/protocol/genesis/src/rollup.rs`, `rust/kona/crates/protocol/derive/src/attributes/stateful.rs`, `rust/kona/crates/protocol/hardforks/src/lagoon.rs`, `rust/kona/crates/protocol/protocol/src/utils.rs`, `rust/kona/crates/protocol/protocol/src/batch/single.rs`, `rust/kona/crates/node/service/src/actors/sequencer/actor.rs`, `rust/kona/bin/client/src/interop/transition.rs`, `rust/kona/bin/client/src/interop/consolidate.rs` |

The specification uses `safe` for the verifier result.
Some implementation paths call the corresponding result `cross-safe`.
Treat these names as navigation terms, not evidence that their semantics match.

Cross-unsafe is a message-safety result in the current filter implementation.
Do not assume it is a chain head.

The protocol permits acyclic messages with equal initiating and executing timestamps.
`op-interop-filter` currently rejects them as a stricter sequencer policy.
Do not report this intentional policy difference as protocol divergence.

## When to run this reviewer

Run this reviewer when a change touches any mapped path or its dependencies.
Also run it for these changes:

- A Kona release that contains mapped changes.
- An interop specification change after publication.
- A dependency-set or message-format change.
- A graph, cycle, expiry, or timestamp rule change.
- A safety label, invalidation, rewind, or replacement change.
- A Lagoon activation or upgrade transaction change.

Do not trigger for tests, metrics, logs, CLI wiring, or monitor-only changes without production changes.

## Interop analysis

Trace the implementation from executing-message declaration parsing through initiating-log lookup.
Track each required field across type conversions, storage reads, and validation calls.

Identify the first failing stage for each rejection branch.
Distinguish malformed declarations, unavailable source data, invalid source data, and graph invalidity.
Confirm which surrounding work each branch preserves.

Trace graph construction from block inputs to each safety or replacement decision.
Enumerate every edge rule and its input source.
Check both edge directions used by cycle detection.

## Sibling paths in interop

The shared guide requires a rule-list diff against sibling paths.
In this area the sibling groups are:

- Go dependency links, supernode verification, interop filtering, and Kona message rules.
- Current-frontier message lookup and accepted-history lookup.
- Initiating-message extraction and executing-message declaration parsing.
- Initial invalid-block replacement and recursive dependent replacement.
- Live Go safety evaluation and Kona proof consolidation.

The last pair shares protocol rules but produces different system outputs.
Compare their rule coverage, not their internal state machines.

## Message and graph checks

Check these boundaries:

- Global log-index accounting across empty and non-empty receipts.
- Message identifier and payload fields through every conversion.
- Dependency membership before source lookup.
- Ordering and expiry arithmetic at exact boundaries.
- Sources at the current frontier and in accepted history.
- Graph nodes with unavailable headers or receipts.
- Both edge types used by same-timestamp cycle checks.
- Self-dependencies and multi-chain cycles.
- Duplicate declarations and repeated initiating messages.

For each missing-data branch, determine whether the code retries, holds, rejects, prunes, or replaces.
Compare each observable outcome with the current specification.

## Safety checks

Identify every frontier source used by each safety decision.
Check hold, progress, invalidation, rewind, reorg, and empty-input paths.

For each promotion branch, identify:

- The local safety input.
- All required executing-message dependencies.
- The accepted-history or current-frontier source.
- The persisted result.
- The value published to consumers.

Prove that each cross-unsafe publication branch validates all required messages.
Do not infer full validation from one successful lookup.

## Replacement checks

Trace every replacement header, receipt, transaction, and output-root lookup to its preimage source.

Distinguish the original optimistic block from its canonical replacement.

Prove data availability for each exact block identity. Do not infer availability from provider-local state.

Trace invalid-block selection and recursive dependent replacement paths.
Determine which transaction classes and log classes survive each path.
Confirm replacement repeats until no invalid dependency remains.

Compare live-node and proof replacement rules independently.
Do not infer one path from the other.

Check whether a later unsafe-ingestion path can restore denied ancestry.
Follow deny records, rewind targets, replacement completion, and head updates across code paths.

## Activation checks

Compare the activation timestamp and activation-block calculation in every implementation.
Compare the dependency configuration selected before, during, and after activation.

Check these cases:

- The last block before activation.
- The activation block.
- The first ordinary block after activation.
- Chains that activate at different local heights.
- Missing or inconsistent dependency configuration.
- Reorgs across activation.

Follow linked upgrade specifications for activation-block transactions and gas allocation.
Do not derive those rules from implementation constants.

## Interop dismissal checks

Apply the shared dismissal checks first.
These interop cases extend them:

- The path implements advisory sequencer policy only.
- The code publishes monitoring data without changing protocol safety.
- Missing data causes a retry before any safety result changes.
- A persistence difference preserves identical graph inputs and published results.
- The compared live and proof paths intentionally expose different internal states.

Do not dismiss a rule-list difference because both paths eventually reject.
Record their rejection stages and surviving work first.
