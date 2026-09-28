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

This mapping belongs in the monorepo because implementation names change independently from protocol behavior.
Each entry names a Go package or Kona crate and one or more anchor symbols in it.
Find each anchor with `git grep`, then follow affected callers and callees.
The anchors are entry points, not a complete list of relevant files.
Go packages are paths relative to the repository root.
Kona entries use crate names from `rust/kona`.

| Behavior | Specification | Go | Kona |
| --- | --- | --- | --- |
| Message extraction and declarations | [Messaging](https://specs.optimism.io/interop/messaging.html), [Predeploys](https://specs.optimism.io/interop/predeploys.html) | `op-core/interop/messages`: `DecodeExecutingMessageLog`, `MessageFromLog`, `ExecutingMessage` | `kona-interop`: `parse_log_to_executing_message`, `extract_executing_messages` |
| Dependency, timing, and expiry rules | [Messaging](https://specs.optimism.io/interop/messaging.html), [Dependency set](https://specs.optimism.io/interop/dependency-set.html), [Interop derivation](https://specs.optimism.io/interop/derivation.html) | `op-core/interop/depset`: `DependencySet`, `StaticConfigDependencySet`; `op-node/superchain`: `LoadDependencySet`; `op-supernode/supernode/activity/interop`: `verifyExecutingMessage`; `op-interop-filter/filter`: `validateMessageTiming` | `kona-genesis`: `DependencySet`, `InteropConfig`; `kona-interop`: `MessageRules`; `kona-proof-interop`: `BootInfo` |
| Message graph and cycle detection | [Messaging](https://specs.optimism.io/interop/messaging.html) | `op-supernode/supernode/activity/interop`: `LogsDB`, `resolveFrontierVerificationView`, `verifyInteropMessages`, `buildCycleGraph`, `checkCycle` | `kona-interop`: `MessageGraph`, `check_no_cycles` |
| Safe promotion and invalidation | [Verifier](https://specs.optimism.io/interop/verifier.html), [Messaging](https://specs.optimism.io/interop/messaging.html) | `op-supernode/supernode/activity/interop`: `progressInterop`, `commitVerifiedResult`, `invalidateBlock`, `applyRewindPlan`; `op-supernode/supernode/chain_container`: `InvalidateBlock`, `DenyList`, `FullyVerifiedL2Head`, `localSafeTimestamp`; `op-supernode/supernode/chain_container/engine_controller`: `Rewind`; `op-supernode/supernode/activity/superroot`: `Superroot`; `op-node/rollup/engine`: `crossSafeCache` | No mapped Kona crate at this guide revision. Confirm during mapping validation. |
| Cross-unsafe classification | [Verifier](https://specs.optimism.io/interop/verifier.html) | `op-interop-filter/filter`: `LockstepCrossValidator`, `LogsDBChainIngester`; `op-service/eth/safety`: `Level` | No mapped Kona crate at this guide revision. Confirm during mapping validation. |
| Invalid-block replacement | [Interop derivation](https://specs.optimism.io/interop/derivation.html), [Super fault dispute game](https://specs.optimism.io/fault-proof/stage-one/super-fault-dispute-game.html), [Holocene derivation](https://specs.optimism.io/protocol/holocene/derivation.html) | `op-node/rollup/engine`: `onBuildInvalid`, `emitDepositsOnlyPayloadAttributesRequest`; `op-node/rollup/derive`: `DepositsOnlyPayloadAttributesRequestEvent`; `op-service/eth`: `IsDepositsOnly` | `kona-proof-interop`: `SuperchainConsolidator`, `replace_local_safe_head`; `kona-client`: `consolidate_dependencies`; `kona-sp1-super-range`, `kona-sp1-super-aggregation`: program entry points |
| Lagoon activation and upgrade handling | [Interop derivation](https://specs.optimism.io/interop/derivation.html), [L2 upgrades](https://specs.optimism.io/protocol/l2-upgrades-1-execution.html), [Superchain configuration](https://specs.optimism.io/protocol/superchain-config.html) | `op-node/rollup`: `IsLagoon`, `IsLagoonActivationBlock`; `op-node/rollup/derive`: `LagoonActivationUpgradeTransactions`; `op-supernode/supernode`: `LagoonTime` checks | `kona-genesis`: `is_lagoon_active`; `kona-hardforks`: `Lagoon`; `kona-derive`: `StatefulAttributesBuilder`; `kona-protocol`: `upgrade_gas_to_strip`; `kona-client`: `sub_transition` |

Callers of the Lagoon activation checks include batch validation and the sequencer.
Follow them from the anchors.

The specification uses `safe` for the verifier result.
Some implementation paths call the corresponding result `cross-safe`.
Treat these names as navigation terms, not evidence that their semantics match.

Cross-unsafe is a message-safety result in the current filter implementation.
Do not assume it is a chain head.

## Sequencer policy and consensus

A sequencer may accept less than the specification permits.
Derivation and verification may not.
They must accept every input that the specification permits.
The same rule difference is therefore a policy choice on one path and a divergence on the other.

Interop has one such difference today.
The specification permits an acyclic message whose initiating and executing timestamps are equal.
See the [timestamp invariant](https://specs.optimism.io/interop/messaging.html#timestamp-invariant) and [intra-block messaging](https://specs.optimism.io/interop/messaging.html#intra-block-messaging-cycles).
`op-interop-filter` rejects a same-timestamp message, because it implements no cycle detection.
See the simplifications comment on `LockstepCrossValidator` in `op-interop-filter/filter`.

Dismiss that rejection only for a caller that admits transactions or builds blocks.
Identify the caller and the requested safety level first.
Report the same rejection on a derivation or verification path as a specification violation.

## When to run this reviewer

Run this reviewer when a change touches a mapped package or crate, or code that an anchor depends on.
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

Compare the message set that each cross-unsafe publication branch validates with the messaging invariants.
Do not infer full validation from one successful lookup.

## Replacement checks

Trace every replacement header, receipt, transaction, and output-root lookup to its preimage source.

Distinguish the original optimistic block from its canonical replacement.

Prove data availability for each exact block identity. Do not infer availability from provider-local state.

Trace invalid-block selection and recursive dependent replacement paths.
Determine which transaction classes and log classes survive each path.
Compare the replacement loop with the specification rule for invalid dependencies.

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
