# Kona node refactor extraction plan

Date: 2026-10-17. Status: implementation in progress; completed package records appear below.

**Execution policy updated 2026-10-18:** the user authorized completing the remaining curated stack without stopping for approval after each draft PR. Earlier review-checkpoint instructions below are historical and superseded. Keep each PR focused and validated, publish drafts for review at the end, and do not merge them or wait for hosted CI after handoff. Optional packages still require their stated value/coverage gates.

## Goal and scope

Build one curated stack of reviewable PRs against `ethereum-optimism/optimism:develop`. Each PR must leave a useful, correct intermediate implementation and remove a specific source of complexity or a demonstrated failure mode. The stack is allowed to stop before reproducing any experimental branch in full.

This workstream makes the independent `kona-node` binary as simple, understandable, correct and robust as possible, retaining its external execution client's Engine API boundary.

The kona-in-op-reth embedding experiment is completely separate. Ignore all work, branches, code, commits, design decisions and results from that experiment when developing or implementing this plan.

Success means the engine has one owner of forkchoice, ordinary operations instead of nested tasks, explicit retry and reset behavior, and handlers that receive the dependencies they need. Sequencer publication has an explicit conductor gate. Configuration has one authority, and disabled features are handled at composition rather than through partially initialized components.

The proposed spine has eleven core review boundaries and two optional tail packages. PR 05 combines the engine consistency changes that cannot safely be staged as separate architectures. PR 11 can fold into PR 10 if moving the already-correct finality map is small. No package requires reproducing the entire final experimental branch.

### Contents

- [Code simplicity and cleanliness principles](#code-simplicity-and-cleanliness-principles)
- [Baseline and source policy](#baseline-and-source-policy)
- [Constraints learned from the experiments](#constraints-learned-from-the-experiments)
- [Intended component boundaries](#intended-component-boundaries)
- [Ordered PR stack and package details](#ordered-pr-stack)
- [Unique issue and implementation ownership](#unique-issue-and-implementation-ownership)
- [Candidate disposition and Option policy](#candidate-disposition-and-option-policy)
- [Review and verification plan](#review-and-verification-plan)
- [Implementation and PR maintenance procedure](#implementation-and-pr-maintenance-procedure)
- [Progress checklist](#progress-checklist)

Path abbreviations in package details: `engine/`, `service/`, `rpc/`, `sources/`, `gossip/`, `disc/` and `peers/` start under `rust/kona/crates/node/`; `bin/node/` starts under `rust/kona/`. Acceptance paths start under `op-acceptance-tests/`. Full `rust/` and `op-node/` paths start at the monorepo root. These identify source paths in the pinned revisions; implementation happens in the selected stack checkout.

### Four rules for every cut

1. Put a bug fix and the smallest refactor that removes its root cause in the same PR.
2. Combine changes when separating them would require scaffolding that the following PR immediately deletes.
3. Give each fix one home in the stack, including its regression tests. Later PRs inherit and preserve it.
4. Create a sibling PR only for a genuinely disjoint change. The default is one stack and one implementation owner, not parallel implementation tracks.

A PR is self-contained relative to its declared parent: it compiles, has its own review argument and validation, and can land after that parent. It need not independently apply to `develop`. Parallel investigation is useful; parallel redesign of the same interface is not.

### Selection criteria

Keep a candidate when it eliminates duplicated ownership, removes an unnecessary asynchronous hop, replaces invalid construction states with required inputs, or fixes a demonstrated failure through a simpler design. Prefer narrow interfaces.

Defer a candidate when it changes operator-facing behavior without serving a selected fix, expands the public type system without reducing states, or merely renames and relocates code. Do not count deleted lines as sufficient evidence of an improvement. Evaluate every candidate against the simplicity, understandability, correctness and robustness of the independent binary.

## Code simplicity and cleanliness principles

Simplicity means a reviewer can understand the normal flow, ownership and failure behavior with very little jumping between definitions. Apply these principles to every PR in the stack. They are part of the implementation and review criteria, rather than a separate cleanup phase.

1. **Use the fewest concepts needed for the actual problem.** Prefer ordinary functions and concrete components. Every trait, wrapper, builder, channel, task and type parameter should justify its existence. A useful abstraction expresses a meaningful boundary or invariant; forwarding calls alone usually does not.

2. **Let modules provide naming context.** Prefer `derivation::Trusted` over `derivation::TrustedDerivation`. Use established imported names consistently instead of introducing aliases or renamed wrappers. Names should describe the domain operation or value precisely. Distinguish unrelated errors rather than giving both the same vague name.

3. **Inline single-use functions by default.** Keep a one-off sequence where it executes, especially when extracting it forces the reader to jump elsewhere just to follow the operation. Keep a helper when it meaningfully isolates an invariant, a distinct algorithm or distracting detail. Single use is a strong reason to inline, rather than an absolute rule.

4. **Write control flow from top to bottom with minimal nesting.** Handle invalid inputs and exceptional cases early with guard clauses, `let ... else`, `?` and early returns. Keep the successful path visually straightforward. Avoid deeply nested matches, callback chains and helpers that obscure the order of side effects.

5. **Choose dependencies and modes at construction.** Resolve configuration once and pass each component the actual values it requires. A handler that needs an engine receives an engine. Register sequencer-specific handlers only when a sequencer exists. Components should not repeatedly rediscover their mode or check whether required dependencies were supplied.

6. **Prefer concrete types and meaningful absence.** Avoid `Box<dyn ...>`, broad generics and single-implementation traits unless a real boundary requires them. Use an enum for a genuinely closed set of alternatives. Keep `Option` where absence has domain meaning; eliminate it where `None` represents incomplete construction or an unsupported combination. Do not replace an unnecessary `Option` with an equivalent enum that preserves the same problem. Apply the specific decisions in the [candidate disposition and Option policy](#candidate-disposition-and-option-policy).

7. **Make invalid states impossible and store less state.** Establish invariants through construction, ownership and operation boundaries. Remove redundant flags, duplicated heads and independently mutable fields that must always agree. When a process has distinct phases, represent the necessary phases explicitly, with their associated data, instead of allowing arbitrary combinations of booleans and optional fields. Every retained transition should correspond to something the system genuinely needs to do.

8. **Give true failures a terminal outcome.** An essential component that cannot continue correctly must propagate its error and shut down the node. Do not log the error and leave an actor stalled, a caller waiting forever or the node apparently healthy. Expected conditions, including temporary dependency outages, stale work and rejected peer input, need explicit retry or discard behavior. Preserve enough typed information to make that decision in one authoritative place. Intentional shutdown and abandoned reply receivers retain their established normal behavior.

9. **Keep one owner for each invariant and decision.** Engine correctness belongs in the engine; sequencing decisions belong in the sequencer. Avoid making several callers interpret the same low-level errors or coordinate duplicate recovery state. Prefer operation results that tell callers what happened over callbacks and extra messages that force them to reconstruct it.

10. **Push configuration and policy decisions upward.** Inject dependencies, durations, clocks and limits instead of creating them deep inside operations. Choose branches at the appropriate caller, and let an operation own its natural batch or loop, following matklad's [Push Ifs Up And Fors Down](https://matklad.github.io/2023/11/15/push-ifs-up-and-fors-down.html). Keep component invariants encapsulated while making environmental choices explicit and deterministic.

11. **Delete the scaffolding a refactor makes obsolete.** Remove unused fields, error variants, wrappers, dependencies and obsolete tests alongside their replacement. Change internal callers when that produces a cleaner interface. Preserve supported external behavior according to this plan's compatibility decisions.

**Fix bugs by removing the conditions that permit them.** Prefer a refactor that eliminates a synchronization obligation or an invalid state over another flag, recovery transition or defensive check around it. Retain the actor and other concurrency boundaries that earn their complexity through ownership, operation lifetime or required progress guarantees; simplify their implementation under the same principles.

## Baseline and source policy

The planning baseline is public `develop` at `9c14635375e22608458da1d3ec4830af491009dd`, verified through the authenticated GitHub CLI. Refresh the baseline and the issue/PR states before implementation; reevaluate only changed areas.

| Source | Pinned revision | Role |
|---|---|---|
| `origin/develop` | `9c14635375e22608458da1d3ec4830af491009dd` | Starting behavior and existing fixes |
| `joshklop/kona` | `61d023c126ce740190f0b2f4b9c94edb66aa89b0` | Latest standalone experiment, including later corrections |
| `joshklop/kona-actor-supervision` | `ee51e1e80ab26607f1d916361d88f7531525be5e` | Public rewrite PR and issue inventory |
| `joshklop/kona-rpc-simplifications` | `2cef02680c655d03ae838b6085cae791bd7d8634` | Earlier RPC cuts |
| `joshklop/kona-watch-channels` | `3e64cb424cf7f9b3dc952ffac36fcd762baf34ca` | Published state and construction experiments |
| `joshklop/kona-engine-task-helpers` | `6ab2ccf25695c31cf61b673fa9c9924549fc12ea` | Local helper extraction; differs from current public PR head |

Use `develop`, the standalone branches listed above, their original issues/PRs, and the corresponding op-node implementation as evidence. Source branches are not acceptance criteria. Inspect the final relevant operation and its history before transplanting it. Preserve corrections already present on `develop`. Never restore an old harness simply because it appears earlier in a standalone refactor branch. A missing correction must be designed and tested against the independent binary rather than treated as an existing source patch.

Use `git show <revision>:<path>` to inspect a pinned source rather than relying on a moving working directory. All GitHub reads and any later authorized GitHub writes use the authenticated `gh` CLI.

### Current PRs and what to do with them

| PR | Observed state and actual comparison | Planned treatment |
|---|---|---|
| [23357](https://github.com/ethereum-optimism/optimism/pull/23357) | Open draft; 342 files, +21,311/-19,832 | Source umbrella; do not use it as the eventual merge unit |
| [23220](https://github.com/ethereum-optimism/optimism/pull/23220) | Merged conductor harness | Inherited from `develop`; do not transplant its earlier versions |
| [23227](https://github.com/ethereum-optimism/optimism/pull/23227) | Open on `develop`; 2 files, +13/-3 | Retain the narrow validation change and combine its dependent invalid-payload coverage |
| [23268](https://github.com/ethereum-optimism/optimism/pull/23268) | Open on the validation branch; 112 files, +3,252/-5,556 | Split by actual dependency/behavior boundaries; do not preserve the mixed diff |
| [23258](https://github.com/ethereum-optimism/optimism/pull/23258) | Open on the RPC branch; 24 files, +743/-696 | Reuse flat helper design; remove the broad RPC prerequisite where compilation permits |
| [23229](https://github.com/ethereum-optimism/optimism/pull/23229) | Open on the helper branch; 20 files, +946/-217 | Preferred source for standalone publication state and retry semantics |
| [23223](https://github.com/ethereum-optimism/optimism/pull/23223) | Open; declared parent is publication branch, but comparison spans 140 files | Reconstruct the recovery-only change on the curated parent; retain leadership assertions |
| [22307](https://github.com/ethereum-optimism/optimism/pull/22307) | Open; one test file at the end of the old stack | Move with validation; its real dependency is 23227 |

The recovery head `6ad3041d69` and its current declared parent's head `8e211de938` have 34 and 62 exclusive commits respectively. Many old RPC/helper patches are equivalent, but some harness/publication patches differ. Its current PR comparison is not evidence that recovery itself needs 140 files. Rebuild the intended delta and inspect the resulting comparison before updating any PR.

## Constraints learned from the experiments

### Preserve fixes already on develop

`develop` already bounds engine transport calls, retains resets across temporary failures, yields task failures to the actor with backoff, and avoids the original unbounded inline task retry loop. The new design must preserve these properties. Issue descriptions written against older revisions do not establish that every symptom remains unchanged.

The current signer watcher retries failed refreshes and rejects results for a stale target, including same-height L1 reorgs. The standalone signer simplification loses both properties. Preserve the behavior before deleting the machinery that implements it.

### Do not copy the standalone tip wholesale

At `61d023c126`, starting a build still lacks the required engine-boundary stale-parent check and maps the FCU error generically. A pending unsafe import also disables message receives, and reset retries run inline. These are counterexamples to treating queue deletion as a complete engine simplification.

The history moves consolidation into derivation and then back into the engine, and introduces a finality actor before replacing it with an engine-owned finalizer. Keep the final ownership lesson without reproducing the intermediate rewrites.

The final standalone RPC surface removes recovery methods and broadens admin exposure in ways not required by the selected fixes. Preserve the established admin enablement gate and recovery API.

### Establish reset boundaries from standalone behavior

The standalone derivation test `reset_discards_attributes_derived_before_it` already establishes that a reset invalidates prepared attributes even when it selects the same head. The plan must enforce that invariant at the engine's mutation boundary as well as inside derivation. Parent equality alone cannot identify which pipeline state produced attributes. PR 05 designs and tests the missing end-to-end context check against the standalone code; it is not a transplant of an existing completed implementation.

### Distinguish dead code from supported behavior

The engine state field `need_fcu_call_backup_unsafe_reorg`, `needs_consolidation`, `EngineKind`, unused `EngineNewPayloadVersion`, and `FORKCHOICE_TASK_LABEL` have no runtime monorepo consumers at the baseline. Remove the relevant leftovers alongside the engine change that owns them; update exports and docs and mark public Rust API breaks.

`EngineClientBuilder` is live on the baseline. Delete it only when the selected client-construction path replaces its actual caller and preserves its deadline tests. `SyncStartError::L1OriginMismatch` is unused, but `kona_derive::ResetError::L1OriginMismatch` is live and must remain.

The `dev`/`ws` RPC modules have test consumers (can be deleted). The safe-head database RPC is advertised even though its implementation is a stub (do not touch the safedb--that will be taken care of separately). CLI aliases, commands, flags, environment variables, API enablement and metrics features are externally observable. They require an explicit behavior decision; they are not dead merely because a later architecture removes their users.

## Intended component boundaries

### Configuration and clients

Keep the current `RollupNodeBuilder` entry point. Its internal result contains one resolved `Arc<RollupConfig>` and the shared L1/L2 providers, with deadlines applied at construction. Actor constructors receive these values directly. Do not introduce the experiment's sequence of public config, endpoints, clients, ChainSpec and builder hierarchies.

There is one production transport factory, not a separately configured client in each consumer. It accepts the real URL, JWT and duration. Preserve the full-response deadline, including a response whose headers arrive but body stalls. Deadlines bound individual requests; retrying a multi-request reset is the engine owner's responsibility.

### Engine operations and command ownership

`kona-engine` keeps ordinary execution operations, forkchoice/state types and the reset walk. `kona-node-service::actors::engine` owns the actor, concrete cloneable `EngineHandle`, private messages, channel capacities, retry state and reset control. No new actors crate or component-wide generic framework is needed.

The intended operation surface is:

```text
state()                         published engine snapshot
start_build(context, attrs)     sequencer-only build, yielding an opaque BuildToken
seal(token)                     getPayload without import, yielding a sealed envelope
canonicalize(sealed)            import that exact envelope and move unsafe head
consolidate(context, attrs)     apply derived attrs, yielding Applied/Replaced/Dropped
follow_safe(context, block)     apply an explicitly trusted delegated safe head
finalize(epoch, block, source)  validate recorded hash/provenance before finalizing
reset()                         acknowledge a completed and published engine reset
insert_unsafe(payload)          reliable bounded admission, not execution acknowledgement
try_insert_unsafe(payload)      nonblocking best-effort gossip admission
```

`BuildToken` carries payload ID, parent and reset context privately. `Sealed` preserves the exact envelope and its context. The sequencer captures its engine context before asynchronous L1/origin/attributes preparation and supplies it to `start_build`; a token created only after preparation cannot detect an earlier same-head reset. `EngineContext` includes the captured parent and reset epoch. Sequencing uses the unsafe parent; derivation uses pending-safe from its last successful pipeline reset or accepted consolidation. Do not force changes to shared proof/protocol attribute types merely to make these node-local distinctions.

The snapshot contains applied heads, a distinct pending-safe parent, reset epoch and explicit initialization/sync/reset phase. Pre-Holocene span prefixes advance only pending-safe until the span concludes; completed safe/local-safe and finality do not advance for that prefix. Fetch actual EL heads before publishing them as usable; zero/default heads are not evidence of readiness. An epoch and heads must be observed consistently. A reset increments the epoch even when it selects identical heads.

Commands requiring execution acknowledgement use `oneshot<Result<...>>`. Unsafe ingestion instead returns a bounded-admission result; it does not await execution. Only the owner sends execution replies; low-level operation helpers do not own reply channels. Publish every successful or partially applied mutation before replying, including state changes preceding a later error. A dropped reply receiver is normal abandonment, not a fatal engine error.

### RPC and the Rust crate boundary

`kona-rpc` already sits below `kona-node-service`; making it import service-owned handles would create a cycle. Keep narrow, permanent capability traits in `kona-rpc`, implemented directly by the service handles:

- `EngineAdmin`: reset derivation and admit an unsafe payload.
- A pruned existing `SequencerAdminAPIClient`: actual sequencer commands/state only, without node reset forwarding.
- A read capability shared by rollup/dev/ws handlers only if it remains useful after direct provider/snapshot reads.

These traits express real module boundaries. They do not require a second queued wrapper, raw sender export, generic actor framework, or production duplicate implementation. The concrete `Message` enums remain private in service modules. Shared read-only snapshot types can live in `kona-engine`.

Node composition registers a base admin module with an engine dependency on validators and sequencers. It registers the sequencer admin module only when an actual sequencer exists. Both remain behind the existing admin enablement flag. Every registered handler receives its required dependency concretely.

### Engine scheduling contract

Use a single bounded FIFO data queue, one active operation, a retry timer, and a separate bounded reset/control path. The component owns all of them. No second priority heap and no indefinite retry loop inside an operation await.

```text
Idle
  receive data -> attempt operation
  receive reset -> enter reset barrier

Retrying(operation, next_attempt, backoff)
  receive reset -> abandon superseded work and enter reset barrier
  retry due -> attempt the same retained work
  new data -> remain bounded in the FIFO

Resetting(epoch, stage, next_attempt, bounded acknowledgements)
  dependency failure -> retain reset, schedule retry
  newer control -> coalesce/supersede according to the bounded control policy
  FCU applied -> publish snapshot, reply, resume data processing
```

Reset is handled at the next safe operation boundary. Do not drop an arbitrary in-flight `newPayload` or FCU future and assume its server-side effect was cancelled. Finish the bounded attempt, then reconcile. Do not spawn detached mutating work whose late completion can update local state after the barrier. Shutdown may stop awaiting transport, but restart/reset reconciliation must still treat a timed-out attempt as ambiguous.

The old heap prioritizes Seal > Build > Insert > Consolidate > Finalize. FIFO changes cross-kind priority deliberately. Sequencer and derivation each limit outstanding work by awaiting their results; gossip uses nonblocking admission; reset bypasses data saturation. Verify latency/fairness under sustained ingress. If FIFO fails a required liveness test, add the smallest explicit admission policy in this owner, not another general task scheduler. -- probably using a biased tokio::select will be enough and you won't need a FIFO queue, but that's just a hypothesis

Retain observable pending-work counts for dev/ws/metrics. Define them as admitted outstanding mutation work, including active retry work, and document any difference from the old heap-only count. Do not return a fabricated zero after deleting the heap.

### Error and retry policy

| Situation | Single engine-owned decision | Caller behavior |
|---|---|---|
| Stale sequencer parent or reset epoch | Reject before mutation | Drop job and prepare on current unsafe head |
| FCU invalid-prestate error `-38002` | Enter one retained reset, including when a reply exists | Observe typed reset-required result; do not initiate a second reset |
| Transport failure or temporary syncing | Retain background work with backoff, or return typed retry for caller-owned build work | Retry according to the current publication phase |
| Invalid external unsafe payload | Apply the existing context-specific drop/sync policy | Continue processing subsequent candidates |
| `ACCEPTED` | Preserve the established mode/operation-specific Engine API policy | Do not turn it into blanket `INVALID` |
| Invalid sequenced block | Drop/rebuild as appropriate; never substitute a deposits-only block for a committed envelope | Preserve the committed-envelope invariant |
| Invalid derived block | Before Holocene: drop/continue according to protocol; after activation: deposits-only replacement and channel flush, keyed to L1 inclusion-source timestamp | Do not use L2 timestamp or reset and reproduce the same invalid batch indefinitely |
| Deposits-only replacement invalid | Critical failure | Stop the node |
| Transient reset lookup failure | Retain reset and retry with backoff | Original reset remains pending |
| Deterministic bad genesis/finality/chain structure | Critical failure | No permanent reset loop |
| Caller drops a result receiver | Continue the accepted operation's policy; ignore reply-send failure | No node failure |

Do not add the experiment's arbitrary 60-second dependency outage -> engine reset rule. Dependency unavailability alone does not establish that forkchoice is invalid. Keep error classification in one owner; reply delivery must never swallow a required reset.

### Reset and pipeline contract

1. At the safe admission boundary, publish a new reset epoch and stop applying old-context work. Data messages left in the bounded queue receive a stale/aborted outcome when examined; they do not resume after reset.
2. Reuse the corrected walk function to choose heads. A retained L1 block retrievable by hash is not necessarily canonical. Validate origins against canonical blocks by number.
3. Apply FCU, publish applied heads/epoch/phase, then acknowledge reset. Acknowledgement means engine reset completed; it does not mean the pipeline finished its own reset I/O.
4. Derivation observes the epoch, discards buffered/in-flight old attributes, resets its pipeline onto the published safe context, and only then steps again. Keep `applied_pipeline_context`; update it only after the reset signal succeeds or an accepted consolidation returns a pending-safe parent within the same epoch, including a rewind after a dropped legacy span. Never label old pipeline work with a freshly read engine epoch. Check context at application, not only at enqueue.
5. A temporary pipeline-reset failure retries the reset before any pipeline step. A newer engine reset supersedes the old target. Cancelling a pipeline step is permitted only when followed by a full pipeline reset.
6. Pipeline-local restart and forkchoice reselection are different operations. Holocene activation gets its activation signal; missing data does not automatically trigger a full EL/L1 walk. Canonicality loss and invalid prestate require engine reconciliation.
7. Engine reset never waits on derivation while derivation waits for the engine's reply. Until callbacks disappear, preserve `develop`'s waiter-release-before-callback behavior. The final path uses published context and replies without that cycle.
8. Clear or rewind applied-safe finality records at every reset epoch, including a same-head reset.

## Ordered PR stack

All selected packages below belong to one spine. Package numbers are review/implementation boundaries, not parallel authoring assignments. PRs 12 and 13 are optional tail work; they must not delay acceptance of the core stack.

```text
develop
  01  unsafe payload validation and its real acceptance coverage
  02  one resolved configuration and bounded-client authority
  03  ordinary engine operations and necessary dead-code deletion
  04  correct, reusable reset walk
  05  engine command owner, scheduler, typed outcomes and epoch barrier
  06  direct read dependencies and binary version
  07  direct admin commands and concrete sequencer dependencies
  08  conductor-aware publication and activation state machine
  09  local disaster-recovery override
  10  engine-owned bootstrap/reset causes and L1 progress
  11  finality from actually applied safe blocks
  12  optional network/discovery ownership and responsiveness
  13  optional prepared origins and attributes inputs
```

PR 02 is a prerequisite for the final provider wiring; 03 supplies reusable operations; 04 supplies the walk reused by 05 and 10; 05 establishes the engine interface that 06-11 consume unchanged. PRs 08 and 09 form one publication/recovery progression. PR 10 removes overlapping reset initiators only after the engine can observe their causes. PR 11 finishes finality ownership without introducing and then deleting a finality actor.

Actual review order may move a truly independent leaf earlier, but its patch and tests have one home. Do not branch a second implementation of config, handles, retries, epochs, sequencer state or finality.

### PR 01: Reject malformed unsafe payloads before admission

**Title:** `kona-node: reject malformed admin unsafe payloads`.

**Problem and resulting behavior.** `admin_postUnsafePayload` currently returns success after queue admission even when the claimed block hash is malformed. Validate the envelope at the accepting RPC boundary, before any actor sees it. A valid payload submitted after rejection must still advance both clients.

**Source and scope.** Retain the current narrow 23227 change (`862e9f08d8`), plus the one-file coverage from 22307. Touch `rpc/src/admin.rs`, `op-acceptance-tests/tests/base/conductor/unsafe_payload_test.go`, and `tests/sync/elsync/gap_elp2p/sync_test.go`. Keep the boundary validation intact when PR 07 changes its transport.

**Steps.**

- Call `payload.check_block_hash()` before sending the payload; preserve the selected bad-hash error contract from 23227 (`InvalidParams`, -32602). Refactoring the handler must not accidentally change that contract.
- Enable the malformed-payload test; fix the EL-only scenario to send `payload2`; assert the CL-side failure is actually bad-hash rejection.
- Submit a later valid payload and require CL and EL progression. This is execution acceptance evidence, distinct from the RPC's queue-admission acknowledgement.
- Correct any handler documentation claiming an admin submission is automatically broadcast. Do not rewrite the network here.

**Ready when.** Malformed input does not enter any queue; valid input retains asynchronous execution semantics; the registered JSON-RPC error and invalid-then-valid scenario pass for kona-node/op-reth and op-node/op-reth. Owns issue 22250. Either absorb 22307 into 23227 or make it its immediate child; it has no conductor-recovery dependency.

### PR 02: Resolve chain rules and build shared bounded clients once

**Title:** `kona-node,kona-engine,kona-node-service!: share config and clients`.

**Problem and resulting behavior.** Overrides are parsed but not applied by startup. Custom rollup files can still get global default-chain P2P identity/storage choices. The engine and ordinary L1/L2 readers construct different clients. Resolve once and inject the same authoritative inputs.

**Sources.** `8ae10f1987` (overrides), `f05d611aa0` (actual chain ID), `b9a765e39e` (duplicate rules), `3c1d99aab1` (initial signer at startup), and the final provider-factory idea around `edf80ee099`. `b420ae9423`, `c0eae687e7`, `b3db89b6b5`, `3d7e0ca0bf` explain intent; do not replay their public API rearrangements.

**Files and changes.**

- `bin/node/src/commands/node.rs::{get_l2_config,run}`: load file/registry, apply overrides, create one shared rollup config. Use its chain ID for the node, P2P bootstore and signing inputs.
- `service/src/service/{builder,node}.rs` and `actors/engine/config.rs`: pass shared providers/config; remove duplicate engine-owned URL/rules fields. Keep the builder entry point.
- `engine/src/client.rs`: add provider-based `OpEngineClient::new`; preserve the existing mockable engine seam. Delete `EngineClientBuilder` only once its real caller is replaced.
- Put the shared transport/deadline factory here or in a small node-only `kona-sources::providers` module. No client rename/crate move or observability overhaul.
- `flags/engine/providers.rs`: wire actual canonical `--l2-engine-timeout` / alias `--l2.timeout`, expressed in milliseconds. Preserve the effective current 10-second default by correcting the advertised 30,000ms default to 10,000ms; explicit operator values become effective. Document that policy once. Do not remove and later reintroduce the setting.
- Move the initial unsafe-signer network lookup out of flag/config resolution into startup using the shared client. Pass the result directly to the enabled gossip validator. Keep retry/stale-result guards for later signer refreshes.
- Resolve a configured signing-key path by reading the existing key. Baseline `SignerArgs::resolve_sequencer_key` calls the P2P `SecretKeyLoader::load`, whose create-if-missing behavior is wrong for an authorized block signer. Separate required signer-key reading from P2P identity creation; an explicit missing/unreadable/malformed signing-key file must fail without creating a random replacement. Check for an existing standalone fix before implementing this once here.

**Boundary.** Keep trust flags, aliases, JWT input formats, modes and unrelated defaults. Local files may be read during config resolution; remote reachability and capability exchange belong to startup. Do not invent defaults for missing required runtime inputs.

**Ready when.** Registry and file overrides reach the actual constructed engine; a custom chain file does not use chain 10's bootstore; injected providers are the ones every consumer calls; configured deadline covers missing headers and stalled JSON bodies; no duplicate hidden connection is built. Invalid local config fails before remote I/O. The handshake and signer lookup use the shared bounded clients. Missing/unreadable/malformed configured signing-key paths fail without creating a replacement; valid existing keys still work and P2P identity creation is unchanged. Addresses the timeout portion of 23034, not its discovery/peer requirements.

### PR 03: Flatten engine operations and delete obsolete internal surface

**Title:** `kona-engine!: flatten engine operations and remove dead code`.

**Problem and resulting behavior.** Build, seal, insertion, synchronization and consolidation construct other task objects to reuse steps. Holocene fallback recurses between task paths. Each existing task should become a flat sequence of ordinary helper calls while the current queue policy remains intact.

**Source.** Extract 23258 onto this parent, adapting only the existing provider/mock seam. Local helper history includes `a833e3d298`, `499e2f2cc6`, `a58d234aaf`, `bef3e64e02`, `6ab2ccf256`; the public PR's rebased equivalents differ. Use the authenticated CLI and compare patches, not SHA names alone.

**Changes.**

- In `engine/src/task_queue/tasks/{synchronize,insert,build,seal,consolidate,finalize}`, extract ordinary `synchronize`, `insert_payload`, `start_build`, `get_payload`, and derived replacement helpers.
- Remove `SynchronizeTask` execution from tasks/reset and eliminate `BuildSealCoupling` once separate derived and sequenced entry points make the distinction explicit.
- Do not change the retry scheduler, error recovery owner, public RPC surface or CLI in this PR.
- Remove unused engine state/types/labels listed above in a separate mechanical commit within this PR, with full-tree reference checks. Keep live version selection and actual canonicality checks. Do not remove a `Display` impl merely because the source deletion commit also did so.
- Keep lower-level operation tests next to the surviving helpers. Helpers do not send actor replies.

**Ready when.** No engine task constructs and executes another task. Existing build/seal/insert/finalize behavior is retained. Tests cover derived fallback, Holocene deposits-only success/rejection and pre-Holocene behavior. Required protocol corrections have explicit assertions and one owner; do not mix an unreviewed fallback behavior change into a claimed mechanical extraction.

### PR 04: Correct the reset walk without changing its owner

**Title:** `kona-engine!: choose reset heads from the canonical L1 chain`.

**Problem and resulting behavior.** Fetching an L1 origin by hash does not prove it is canonical; providers retain old-fork blocks. Correct canonicality, ancestry and coherent L1 reads in the existing walk, then reuse that implementation in later reset ownership.

**Safe-selection dependency.** Retain Kona's existing recent-window resume policy here: return the current block once its origin is more than the lookback behind the highest confirmed canonical origin, or finalized is reached. This may select above the old safe label. A strict prior-safe ceiling would force first-sync/imported-EL startup back to genesis because Kona has not yet established a reusable bootstrap checkpoint. PR 10 owns the explicit initialization checkpoint contract and the stricter normal-reset safe selector together. PR 05 preserves this compatibility policy while changing command ownership; it must not implement a second selector.

**Evidence and implementation basis.** At the pinned baseline, `engine/src/sync/mod.rs::find_starting_forkchoice` accepts an unsafe origin whenever a by-hash lookup returns a block, then starts its safe search from unsafe without a prior-safe ceiling. Standalone `ce54ccff44` identifies the canonicality problem. Compare the existing `op-node/rollup/sync/start.go::FindL2Heads` and its test table. Implement the correction through kona-node's existing external RPC boundary; this package requires new standalone implementation and tests rather than a ready-made Rust patch.

**Changes.**

- Keep `kona-engine::find_starting_forkchoice`; refactor only the read boundary needed for scripted cases in `engine/src/sync/{mod,forkchoice,error}.rs`.
- Validate origin hashes against canonical L1 blocks by number; preserve plausible unsafe origins beyond current L1 visibility instead of treating lack of visibility as proof of invalidity.
- Preserve configured genesis, finalized anchor, sequence continuity and reorg-distance rules. Measure the safe-resume lookback from the highest confirmed canonical origin, rather than an unseen ahead-of-L1 origin. Lookback is the sequencing window or the longer AltDA challenge-plus-resolution window, in L1 blocks. Cache repeated canonical origin reads within one walk, not in a new actor.
- Distinguish retryable dependency reads from deterministic invalid chain structure. The later owner retries; this function does not sleep forever internally.
- Capture L1 visibility at the beginning of an attempt, anchor the relevant canonical chain by number, and follow its parent hashes. Recheck the numeric anchor before returning; a changed or missing anchor returns a retryable error to the existing owner, which restarts the whole walk. Ordinary L1 extension leaves that captured visibility valid and does not force a retry. This does not promise an atomic snapshot through the later FCU.
- Preserve an early safe-resume return; do not add a mandatory traversal from that boundary to finalized or genesis. Check either anchor's identity whenever its height is reached. Missing provider data remains retryable; malformed ancestry and wrong genesis/finality remain fatal through the existing reset error classifier.

**Ready when.** Cover the applicable op-node canonicality/ancestry/error behavior with focused regressions; do not transplant overlapping table rows or claim full safe-output parity. Cover retained-old-hash, unsafe-ahead-of-L1, missing parent, finalized reorg and genesis cases. A large-height startup with genesis safety labels must resume after bounded recent-window reads. The function returns an ordinary result and PR 05 calls it unchanged. PR 10 reuses the canonical reader/cursor/anchor and changes the shared safe-selection policy once with its bootstrap prerequisite; no duplicate walk or temporary public API.

### PR 05: Give engine commands one owner and honest outcomes

**Title:** `kona-node,kona-engine!: execute engine commands through one owner`.

**Problem and resulting behavior.** Raw request messages are translated into a second task queue, reply channels hide failures, publication happens after replies, and reset can be trapped behind retained data work. Replace these together with the intended engine contract and scheduler. This is the largest core review unit because splitting it would create a known-bad temporary scheduler or a second command API.

**Sources.** `e840d5ee35`, `4025a47e0d`, `3f8d4ad018`, `1b042549b5`, `d08e39eb5c`, `dd0a91ce8e`, `649adc9252`, `d6b1a78679`, later correction `c8ab5cb4dd`; take concepts, not chronological patches. Closed/unmerged [22686](https://github.com/ethereum-optimism/optimism/pull/22686), commit `f60e50dc03d6b5`, supplies the stale-build correction; it is not on `develop`. Extend the standalone derivation reset tests, including `reset_discards_attributes_derived_before_it`, to establish the same-head context contract across the real engine handoff.

**Implementation commits within this PR.**

1. Add service-owned handle/private messages and result types. Introduce the final split start/seal/canonicalize API immediately. Existing sequencing can call seal then canonicalize consecutively until PR 08 inserts conductor acceptance between them; do not add a temporary combined public method.
2. Replace task structs/dispatch with ordinary engine operations from PR 03. Keep the engine crate and module boundaries; no actor-crate migration.
3. Replace heap scheduling with active FIFO work, bounded reset control and a retry timer. Remove optional reply senders and duplicate unsafe-head publishers. Preserve queue/dev/ws observability with actual pending-work counts.
4. Centralize recovery classification. Check stale sequencer parent before FCU, while derived replacement retains its legitimate different parent. Route `-38002` from start and canonicalize into one engine-owned reset even if returning an error to a caller.
5. Publish snapshot before replies. Introduce the epoch barrier and context capture/application check from birth, including same-head resets and already queued old messages.
6. Adapt engine-operation callers directly to the final handle, removing only their obsolete queued wrappers. PR 07 owns the RPC-to-sequencer wrapper removal. Establish the complete result/context derivation handoff here: `Consolidated` is the sole completion/flush response, and the atomic engine snapshot is the reset/sync source. Remove all pushed safe-head, Flush, reset and sync callbacks, their queue, and `EngineDerivationClient` together; no result-to-old-queue adapter or temporary versioned callback mechanism.
7. Introduce `applied_pipeline_context` in that handoff. Observing a newer Resetting epoch invalidates/cancels old work immediately, but derivation waits for completed usable heads before signalling its pipeline reset. Only a successful reset signal installs the new epoch. Initializing/Syncing snapshots are not readiness. Adapt the existing state-machine transitions directly or reduce them to the final result-driven loop here once; PR 10 does not replace that protocol. Retain current bootstrap initiators and sync policy until PR 10 takes ownership of their causes.
8. Correct existing finality bookkeeping at the handoff: record only the actual Applied/Replaced block hash and its real L1 source after acknowledgement; record nothing for Dropped/stale/failed work. Clear/rewind on reset epoch, and validate safe/canonical hash before finalization. Do not keep baseline pre-application/by-number recording after introducing those outcomes. PR 11 will move the already-correct map, not implement this fix a second time.

**Existing rejection policy to correct here.** PR 03 deliberately preserves these policies so its operation extraction remains reviewable. PR 05 owns their correction together with command outcomes and derived context:

- Select Holocene activation using the actual L1 inclusion-source timestamp carried by `attributes.derived_from`, rather than the L2 payload-attributes timestamp. Test both directions of the activation boundary when those timestamps differ.
- Apply deposits-only replacement to derived `INVALID` payloads. The existing fallback trigger groups `INVALID` and `ACCEPTED` and also runs for sequenced inputs; give each outcome its context-specific handling. This describes which status triggers fallback, not whether a replacement succeeds.
- Route invalid build-start FCU results through derived rejection recovery. The existing temporary `EngineBuildError::InvalidPayload`, wrapped by `BuildTaskError::EngineBuildError`, bypasses replacement and can retain the batch indefinitely.
- Drop and continue after invalid derived batches before Holocene, rather than retaining the existing temporary error indefinitely.

These corrections belong to this PR only; later reset, sequencing and finality packages consume the resulting outcomes without implementing a second recovery policy.

**Files.** `service/src/actors/engine/{actor,request,client,tests}.rs`, a new private `handle.rs`, `engine/src/{task_queue,state}`, sequencer `engine_client.rs`/callsites, derivation engine client/callsites and service composition. Shared snapshot/outcome types belong in `kona-engine` only when RPC also needs them. Remove engine-command forwarding wrappers whose callers migrate here. The read-only RPC wrapper stays unchanged until PR 06 and the sequencer-admin wrapper until PR 07; do not add a new temporary API to either. Retain narrow permanent capabilities where they express a real crate boundary.

**Ready when.**

- A stale sequencer start causes no FCU; a derived unsafe-chain replacement still succeeds.
- Start/seal/canonicalize return real typed errors; deterministic rejections do not retry indefinitely; `ACCEPTED` retains its context-specific behavior.
- State is current immediately after a successful reply. Partial mutation followed by failure is visible too.
- A failing unsafe import with a full data queue cannot prevent reset admission/handling during backoff. Retried unsafe blocks remain ordered; finalized/derived work is not permanently starved.
- A same-head reset rejects attributes/build tokens captured before it, including when its new notification is queued behind old pipeline work. A reset during a bounded FCU/newPayload attempt reconciles rather than assuming the server cancelled the request.
- Derivation can await consolidation while its notification queue is full without cyclic waits. It advances context only after successful pipeline reset. Dropped/stale/failed work creates no finality entry; Applied/Replaced records the actual applied hash and source.
- Losing a reply receiver leaves the engine alive. Reset retries recover without a fresh inbound message and without unbounded retained waiters.
- Add a real-node short sequencing-window/batcher-outage force-inclusion regression through a Kona-capable minimal preset. Resume batching and require sequencing plus safe/finalized convergence. Do not substitute the supernode-only expiry test or an in-process Go action sequencer.

Owns issue 22681's engine failure mechanism. Its closure requires the actual force-inclusion scenario, not just new unit tests. PR 10 extends startup/reset lifecycle tests without reimplementing this policy.

### PR 06: Serve RPC reads from their actual dependencies

**Title:** `kona-rpc: serve engine reads directly and report the node version`.

**Problem and resulting behavior.** Ordinary reads cross forwarding clients and an engine RPC actor; version/health use the RPC library package version. Give handlers a read-only provider, published state, resolved config and actual binary version.

**Sources and correction.** `d2515230fe`, `f01ec9b0da`, `f13931c1b5` (rebased equivalents `8749012953`, `8f465327c2`, `23c8df5484`) supply the standalone read/version refactors. Preserve existing dev/ws methods from `develop` while removing their forwarding path. The baseline `EngineQueries::handle` fetches a block and then requests storage by the original number/tag; implement and test hash-pinning directly when extracting that helper rather than copying this race from the source refactor.

**Changes.**

- `rpc/src/rollup.rs`: read config/state directly and calculate output in one ordinary provider helper. Resolve a block number/tag once. Before Isthmus, request the message-passer proof/storage at that exact fetched block hash; after Isthmus, use that header's withdrawals root. Never re-resolve a moving tag between block and proof.
- Inject the cached binary `op-version` release string into both `RollupRpc` and `HealthzRpc`. Release and development values match CLI/metrics; neither handler uses `CARGO_PKG_VERSION`.
- Delete `service/actors/engine/rpc_actor.rs`, engine query enums/channel plumbing and queued RPC forwarding together.
- Preserve `dev`/`ws` methods using the real state/pending-work watches, HTTP/WebSocket behavior, limits, JSON-RPC health and GET `/healthz` proxy. Retain existing L1 query wiring until PR 10 supplies its final publication shape.

**Ready when.** A stalled output read does not serialize config/version/status; pre-Isthmus proof stays hash-pinned across a simulated reorg/tag movement; Isthmus/missing-block/error cases are covered; injected release/dev version reaches both registered endpoints; publisher closure is handled; dev/ws still function. Owns [protocol-team 418](https://github.com/ethereum-optimism/protocol-team/issues/418).

### PR 07: Route admin operations to their owner

**Title:** `kona-rpc,kona-node-service: give admin handlers their required dependencies`.

**Problem and resulting behavior.** Validator reset depends on an absent sequencer; unsafe input crosses the network actor; every sequencer endpoint repeats an optional-handle check. Resolve these at composition using the final engine handle and a sequencer-owned private protocol.

**Sources.** `685b671f52`, `d8e35f6476`, `a4a16a8360`, `3a7adc6c58`, selected final `admin.rs`. Preserve recovery methods omitted by the tip. The final constructor accepts real dependencies; do not export a raw engine request sender as an intermediate API.

**Changes.**

- Define the narrow permanent `EngineAdmin` capability in `kona-rpc`; implement directly on service's engine handle. Split base admin and sequencer admin module registration under `--rpc.enable-admin`.
- Add service-owned `SequencerHandle`, private messages, oneshot results and one published sequencer state. Implement the existing pruned RPC capability directly; delete queued RPC wrappers/duplicated admin dispatch.
- Route `admin_resetDerivationPipeline` straight to engine reset, preserving completed-reset acknowledgement and retry semantics. Remove only its sequencer forwarding message/helper.
- Route validated `admin_postUnsafePayload` to reliable engine admission. Remove `NetworkAdminQuery` and its forwarding channel; keep the validation and tests from PR 01 unchanged.
- Preserve active/recovery/conductor queries, `admin_setRecoverMode`, existing recovery-read wire spelling and sequencer-only MethodNotFound on validators. Read immutable configuration directly; publish mutations before acknowledging.

**Ready when.** Registered reset works on both node modes with admin enabled; a full sequencer queue cannot block it; acknowledgement precedes waiting on full derivation notification delivery; temporary L1/L2 failures recover through the RPC path. Closed channels retain existing internal error mapping. Active/recovery reads work while command mailbox is full. Dropped result receivers do not undo commands. Admin-disabled exposure remains disabled. Owns [protocol-team 419](https://github.com/ethereum-optimism/protocol-team/issues/419); PR 10 must preserve these exact tests.

### PR 08: Make conductor publication and activation one state machine

**Title:** `kona-node: gate sequencer publication on conductor acceptance`.

**Problem and resulting behavior.** Sealing currently imports/canonicalizes before conductor acceptance; activation ignores required preconditions; job and retry state are split across fields. Use one publication state machine on PR 05's permanent operations and PR 07's sequencer handle.

**Source.** Dedicated 23229 and its standalone state machine (`644390a25b`, `92ccb35001`, `74562c5f28`, `90b54dc8d1`), plus activation `a9862b640f` and injected transport `0850ad121c`. Resolve retry and stop behavior against the original standalone issues and current op-node implementation; the latest standalone patch is not sufficient evidence of the intended contract.

**Files.** `service/src/actors/sequencer/{actor,conductor,config,error,admin_api_impl}.rs`, private handle/state, `bin/node/src/flags/sequencer.rs`, node composition, registered `admin_startSequencer` signature/error mapping, and focused Rust/Go tests.

**State model.**

| State/event | Required action |
|---|---|
| Idle, active tick | Prepare/start build on current engine context; retain opaque job |
| Building, seal due | Obtain payload without importing; ask conductor to commit |
| Commit error/rejection/timeout | No canonicalization or gossip; retain job and retry `getPayload` after one second |
| Positive commit acknowledgement | Retain the exact envelope as Committed; canonicalize it |
| Committed, temporary import failure | Retry the exact envelope; no new getPayload, recommit or replacement block |
| Stale/invalid committed envelope | Resolve/drop the unusable job under the engine contract; never substitute another envelope |
| Expired payload job | Discard/rebuild; do not retry an unknown ID forever |
| Canonicalization success | Engine state already published; then sign/enqueue gossip and schedule next build |
| Stop with only uncommitted work | Clear job/timer; return settled current unsafe hash |
| Stop with committed work | Apply the explicit settle contract below; no stale job survives a later restart |
| Engine reset/epoch change | Invalidate old job and do not resurrect it on resume |

**Activation.** Change the registered start method to accept expected unsafe hash. A conductor-configured node starts stopped in production construction even without fixture flags. Check leadership with the configured deadline and require known/current matching head before activation. Pin op-node's already-running, non-leader, leadership-error, unknown-head and mismatch behavior rather than inheriting the tip's early-success shortcut. Use compatible domain errors; channel/transport failures remain distinct.

**Stop and responsiveness.** Accepted Stop prevents new build admission. Finish and record the current bounded seal/commit/import action; never cancel it and assume it had no effect. Discard uncommitted jobs. If a commit succeeds, including a late reply after Stop was requested, retain the exact envelope and drive its import retries while checking leadership. A successful import publishes the head and enqueues gossip before Stop acknowledgement; reconcile a late import success too. A not-leader result or failed leadership query ends the settle wait and clears the job as in op-node; stale/invalid resolution also ends it. Then publish inactive state, clear job/timer, and reply with the settled head. Restart starts a fresh job. A dropped RPC reply does not cancel an accepted Stop or strand settlement. Shutdown follows the node's reconciliation path; no old job initiates engine mutations after Stop acknowledgement, though already queued gossip may finish delivery. Reads remain available during L1 preparation and engine round trips. Clear-and-rely-on-repost is not the selected design.

**Timeouts.** This PR uniquely wires the existing whole-seconds conductor timeout flag into the injected transport for both leader checks and commits. Keep existing syntax. Do not add a method-local timeout wrapper only to replace it with a transport later. Stop waiting uses bounded individual dependency calls while preserving reset/shutdown progress; a process-wide arbitrary outage deadline must not discard acknowledged committed work silently.

**Ambiguous commits.** A timeout can occur after Raft acceptance. Local publication still requires positive acknowledgement; retry the job under the standalone protocol. Once acknowledgement is observed, the exact envelope is pinned. Do not claim exactly-once delivery from an HTTP timeout.

**Ready when.** Test commit failure/no publication/no next build; acceptance with lost reply; reseal-before-ack retry; exact post-ack import retry; expired job; stale/invalid outcomes; unmanaged sequencing; stop during each phase, including commit/import success arriving after Stop and abandoned stop reply; leadership failure while settling; start after stop/reset; state-before-reply ordering; actual configured timeout; production stopped startup; all start rejections. Keep shared conductor assertions and test wrong-hash start on both CLs. Convert the paused-controller publication diagnostic into durable acceptance coverage. Owns 23264, 23323 and conductor-timeout part of 22667. Do not close 22667 while max-safe-lag remains unsupported.

### PR 09: Add a local conductor recovery override

**Title:** `kona-node: support conductor disaster recovery locally`.

**Problem and resulting behavior.** Recovering local sequencing after quorum loss is different from overriding conductor's leader-gated proxy. Add one local policy fact to PR 08's state machine.

**Source.** `eeb931514a` and the intended narrow 23223 delta, reconstructed on the curated parent. Do not import its obsolete lower-stack history.

**Changes.** `admin_overrideLeader` bypasses local leadership/commit policy; it does not call `conductor_overrideLeader`. Conductor configuration reporting remains meaningful. Preserve process-lifetime-only policy unless persistence is explicitly required. Match op-node's harmless no-conductor override behavior. Delete the obsolete `Conductor::override_leader` forwarder when it has no caller.

**Ready when.** Keep `TestDisasterRecoveryLeaderOverride`'s pre-override `sequencer is not the leader` rejection. Verify production resumes after local override, proxy remains refused until a separate conductor override(true), and proxy refusal returns after external override clears. Same assertions on kona-node and op-node; no client-specific weakening. Rust tests show no conductor leader/commit requests after local override. Owns 23266; keep 21903/21906 open until all umbrella criteria are met.

### PR 10: Make bootstrap and reset causes engine-owned

**Title:** `kona-node: own bootstrap and L1 reorg resets in the engine`.

**Problem and resulting behavior.** PR 05 already removed callback cycles and established the final pipeline handoff. Startup and sequencer code still initiate some engine resets. Give those causes to the engine only once it can detect them itself, retaining the same public operations, error policy, applied pipeline context and reset observation.

**Sources.** Selected intent from `e582433780`, `286307f410`, `97d8169524`, `b9c7697989`, `2bab1546eb`, `ca10cfc35f`, `b89878078f`, and repeated-reset backoff `25849b7ad6`. Use standalone `2cef02680c` for L1 observation/progress publication. Verify the existing delegated-derivation polling path against the original status contract and add standalone coverage for progress changes while the safe head stays unchanged. Reuse PR 04's walk and PR 05's epoch/error/scheduler/applied-pipeline-context implementation. Do not introduce the final standalone Safety/Finality traits or global L1 chain actor.

**Changes.**

- Fetch real initial EL heads through the shared bounded client; engine owns initial reset and readiness/EL-sync phase. Do not use finalized>genesis alone as an unproved universal readiness heuristic.
- Establish the initialization checkpoint contract before enforcing a prior-safe ceiling. Cover validator first-VALID and sequencer initial-reset paths, including an imported/synced EL far from genesis without safety labels. State the trusted resume assumption explicitly: canonical origins and elapsed windows alone do not prove local L1 derivation. Preserve finalized; do not promote unsafe to safe/finalized as an expedient shortcut. Any initialization exception must be consumed after successful initialization and must not apply to later ordinary resets or override a known reusable checkpoint.
- In the same change, activate strict normal-reset safe selection: never exceed prior safe, find the first qualifying epoch-start block using the strict sequencing/AltDA lookback, then select its parent. Reuse PR 04's canonical ancestry and coherent-view machinery; this policy and its bootstrap prerequisite have one implementation owner here.
- Engine observes the L1 head and canonicality of the origins its applied heads rest on, using bounded canonical-by-height lookup. Remove sequencer-initiated startup/reorg resets only after the engine observes those causes itself.
- Remove only obsolete bootstrap initiator calls/gates. The result-driven pipeline handoff, applied context, consolidation outcomes, flush handling and post-application finality records from PR 05 stay unchanged. Do not rewrite the pipeline loop or introduce another reset interface.
- Callback queues, `EngineDerivationClient` and pushed safe/reset/sync/Flush delivery were already deleted in PR 05. This PR neither reintroduces nor migrates them.
- Keep pipeline-local reset separate from full forkchoice walk. Retry a failed reset signal before stepping; back off repeated reset-without-progress; select newer engine epoch before further work. Take prepared attributes without an intervening await or explicitly clear their buffer on reset.
- Publish actual pipeline `current_l1` separately from latest/safe/finalized L1 observations, with one agreed stable publication shape. Remove `L1WatcherQueries` and per-RPC L1 I/O when this source is ready. Poll observations independently so one blocked source/consumer does not stall all three.
- Trusted derivation publishes validated source progress on every relevant poll even when safe head does not change; preserve legitimate unknown current-L1 state.

**Files.** `service/src/actors/{engine,derivation,l1_watcher}`, derivation state-machine/client/request modules, service startup/composition, `rpc/src/{l1_watcher,rollup}.rs`. No new canonical-history storage, proof-provider actor dependency, trust-default change or CLI renaming.

**Ready when.** Table-test pristine sequencer, first-sync validator, synced/restarted validator before/after finality, and trusted following. Large-height initialization without safety labels uses bounded recent history; subsequent ordinary resets obey the prior-safe ceiling and epoch-parent boundary. A temporary initialization failure retains eligibility, success consumes it, known safe checkpoints are respected, and finalized never advances merely to accelerate startup. Temporary L1/EL loss recovers without new events. Same-head reset discards old work; pipeline reset failure forbids stepping; newer reset supersedes retry; startup reset publishes before consumers use heads. Actual current-L1 can lag the observed head and change while safe head is stable. Signer refresh still retries without a new head and rejects stale/same-height-reorg results. Preserve PR 07's registered reset/acknowledgement tests and PR 05's force-inclusion regression.

### PR 11: Finalize only blocks the engine actually made safe

**Title:** `kona-node: record applied safe blocks for finality in the engine`.

**Problem and resulting behavior.** PR 05 has already corrected pre-application finality recording and stale/by-number outcomes. Move that correct map into the engine so the component that applies safe blocks also owns their finalization, eliminating the remaining finalization command hop. This PR improves ownership; it does not claim to fix PR 05's bug again.

**Sources and remaining work.** Use standalone `6e79d5a578`, `b41ce04222` and the final map logic around `61d023c126`. Verify the independent node's existing finalization behavior and op-node's corresponding contract for idle/canonicality cases; implement any missing correction directly with standalone tests. Do not create `cc12cdc769`'s separate finality actor first or copy finalizer logic that only infers reset from backwards heads.

**Changes.** Move the existing post-application record operation, map and epoch rewind together, preserving actual replacement hash and source. Observe finalized L1 independently of pipeline work; select the highest eligible record and verify it is still safe/canonical on the recorded L1 chain before FCU. Retain records across temporary FCU failure. Handle trusted-source finality as explicit mode policy, not invented L1 provenance. Remove the now-redundant derivation finalization dispatch and remaining by-number API only when no legitimate caller needs it. If that move is trivial enough to fit PR 10 cleanly, absorb this package there rather than maintaining a ceremonial extra PR.

**Ready when.** Failed/stale consolidation creates no finality record; L1 already finalized when new safe work arrives progresses; derivation-idle finality progresses; wrong-fork same-height hash cannot finalize; same-head reset invalidates old-source record; failed FCU retries; catch-up does not lose progress. Verify multiple newly safe blocks under the same finalized L1 observation: the optimization in `61d023c126` is not authority to suppress required later progress. Reuse epoch/snapshot code unchanged. This package changes in-memory finalization ownership, not persistent safe-head history.

### PR 12 selected leaf: Wait for actual P2P connection effects

**Title:** `kona-rpc: wait for peer connection effects within one deadline`.

**Value and dependency.** The tail audit found two directly reproducible RPC bugs that can be corrected without a network actor rewrite. The existing retry treats `Ok(false)` as completion, so a normal asynchronous connection transition fails too early. Its delay budget also leaves channel admission and replies unbounded. This leaf consumes the existing request interface unchanged.

**Changes.** Send the requested mutation once, then observe connected peer membership until the promised connect/disconnect effect occurs. Poll unsatisfied observations at a bounded pace under one ten-second deadline covering mutation admission, observation admission and replies. Channel closure returns promptly. Register canonical `opp2p_blockAddr` while retaining `opp2p_blocAddr` as an explicit legacy alias. Delete the RPC crate's resulting unused retry dependency.

**Ready when.** Registered regressions prove delayed effects, terminal closure, a false predicate through deadline, stalled mutation admission/read admission/reply, and a combined admission-plus-reply wait bounded by the same deadline. Both endpoint spellings must validate arguments and mutate/read the same actual connection gater. Reuse the existing real P2P common suite as a compatibility smoke check.

**Deferred broader candidate.** Do not transplant private network/discovery actors, snapshot APIs, a general network handle, or an origin preparation system to fix these RPC bugs. Honoring `--p2p.no-discovery` remains valuable separate work: it needs no UDP bind, ENR/discovery startup, bootstore I/O or discovery-only DNS, plus occupied-port and real explicit-peer propagation coverage. Discovery-ban queue responsiveness, the gossip self-channel and peer-manager settings also remain separate. Issue 23034 stays open. PR 05/07 already own engine admission/admin routing and PR 08 owns committed sequencer publication; do not recreate those paths.

### PR 13 optional: Prepare origins and the actual attributes inputs before sequencing

**Execution decision.** Defer the performance implementation: no measurement establishes a material avoidable preparation delay, and the attributes reader's actual receipt/header inputs would need to share any warmed cache. Issue 23281 stays open. Select a separate small correctness leaf in the existing delayed provider: a positive confirmation depth must not silently disappear while the observed L1 head is unknown. Preserve explicit zero-depth/genesis behavior and direct by-hash current-origin reads; use overflow-safe eligibility and prove actual RPC counts before/after a watch observation. This leaf introduces no preparation actor, cache or performance claim. Its implementation/validation is recorded separately below when complete.

**Title:** `kona-node: prepare L1 origins and attributes inputs before sequencing`.

**Value and dependency.** Own issue 23281 only if selected. Build on the settled sequencer/reset context from the core stack. The standalone origin actor alone does not fix this issue: its attributes builder still has a separate receipt cache.

**Sources.** `89b1359c13`, `63d6dd0e00`; preserve the issue's single-writer and confirmation rules. Do not port the first waiting-handle implementation or depend on a full shared L1 history actor.

**Changes.**

- One preparation task publishes a bounded immutable current/next/lookahead origin window keyed by parent origin and reset context. One writer owns promotion; selection is a synchronous decision.
- Warm headers and receipts in the exact cache the sequencer's `StatefulAttributesBuilder` reads. Cloning `AlloyChainProvider` clones cache contents; it does not share future inserts. Use a node-local shared hash-keyed cache or explicitly prepared per-origin inputs, with no lock held over await.
- Recheck returned hashes/heights, confirmation depth and canonical parent links. Unknown L1 observation does not disable confirmation policy. An orphaned current origin enters the established engine reset path once.
- Trigger lookahead from L1 observations and adoption/engine-head changes. Add an adoption signal only if a deterministic race test proves the existing trigger insufficient; never add a second window writer.
- Keep waiting in preparation. Unavailable data skips a sequencing tick under drift rules; no unbounded max-drift fallback in the sequencer. Reset invalidates old preparation even with unchanged heads.

**Ready when.** RPC counters demonstrate no header/receipt/by-number wait between seal and next build in prepared steady state; 10-second L2/12-second L1 orderings keep up; zero/nonzero/unknown confirmation head is correct; same-height reorg and in-flight reset cannot publish stale preparation; shared cache inserts are actually observed; L1 outage leaves admin/reset responsive. Run a delayed-L1 real sequencer/verifier origin transition. Do not close 23281 for origin-window code alone.

## Unique issue and implementation ownership

This ledger is the authority for where fixes live. When a source patch lands separately, inherit it from the refreshed baseline and remove that implementation task from the stack. Keep its regression tests through later structural changes.

| Issue or existing work | One home | Completion rule |
|---|---|---|
| [22250](https://github.com/ethereum-optimism/optimism/issues/22250), 23227, 22307 | PR 01 | Synchronous rejection plus invalid-then-valid real client coverage |
| Parsed hardfork overrides ignored | PR 02 | Actual engine and node use resolved registry/file rules |
| Custom-chain P2P identity/bootstore uses global default | PR 02 | Loaded rollup chain ID drives construction |
| [23034](https://github.com/ethereum-optimism/optimism/issues/23034), engine timeout portion | PR 02 | Non-default duration changes full-response deadline on the shared client |
| [22681](https://github.com/ethereum-optimism/optimism/issues/22681) | PR 05 | Stale-build/error/control regression plus full force-inclusion recovery; later PRs preserve it |
| Same-head reset invalidates prepared work | PR 05 | Engine checks producer's applied context before mutation; both sequencer and derivation interleavings |
| Safe record before application / wrong-fork finality | PR 05 fixes handoff; PR 11 only moves correct ownership | Actual applied hash/source only; no Dropped/stale record |
| [protocol-team 418](https://github.com/ethereum-optimism/protocol-team/issues/418) | PR 06 | Registered version and health return injected release/dev string |
| [protocol-team 419](https://github.com/ethereum-optimism/protocol-team/issues/419) | PR 07 | Registered validator/sequencer reset, completed acknowledgement, outage recovery |
| [23264](https://github.com/ethereum-optimism/optimism/issues/23264), 23229 | PR 08 | Conductor acknowledgement gates publication; exact post-ack envelope retries |
| [23323](https://github.com/ethereum-optimism/optimism/issues/23323) | PR 08 | Production stopped startup, registered expected-hash/leadership contract |
| [22667](https://github.com/ethereum-optimism/optimism/issues/22667), conductor timeout portion | PR 08 | Configured deadline reaches leader and commit transport |
| 22667 max-safe-lag / [22266](https://github.com/ethereum-optimism/optimism/issues/22266) | Deferred explicit feature/removal decision | Do not close for timeout alone or silently delete/reintroduce the flag |
| [23266](https://github.com/ethereum-optimism/optimism/issues/23266), 23223 | PR 09 | Local escape hatch and separate proxy override, with original shared assertions |
| [22262](https://github.com/ethereum-optimism/optimism/issues/22262), merged 23220 | Baseline harness | Verify actual Kona route/coverage; do not count old harness transplantation as new work |
| [21903](https://github.com/ethereum-optimism/optimism/issues/21903), [21906](https://github.com/ethereum-optimism/optimism/issues/21906) | Final gate across 01, 08, 09 | Shared real Kona conductor suite, CI route, handoff/failure/convergence and publication safety all proven |
| Actual `current_l1` progress | PR 10 | Use standalone `2cef02680c`; do not create a second implementation outside its owning status source |
| Reset canonicality / coherent L1 ancestry | PR 04 | Retained old-fork hashes cannot validate an origin; moving anchor retries through the existing owner |
| Bootstrap checkpoint and strict safe-reset selection | PR 10, together | Bounded initialization far from genesis; subsequent reset respects prior safe and epoch-parent boundary; PR 05 preserves PR 04's compatibility selector |
| Configured signing-key path creates a random missing key | PR 02 | Required key reading is distinct from P2P identity creation; verify baseline/existing standalone PR before authoring |
| `opp2p_blockAddr` declaration typo | PR 12 or a disjoint early leaf, once | Baseline registered-method test; no duplicate correction in both places |
| 23034 discovery-disable portion | Optional PR 12 | Disabled discovery has no discovery side effects; occupied UDP port is harmless |
| 23034 peers low/high/grace portion | Deferred explicit removal/feature decision | No existing manager to configure; do not add a manager merely to make the refactor comprehensive |
| [23281](https://github.com/ethereum-optimism/optimism/issues/23281) | Optional PR 13 | Origin selection and actual receipt/header preparation both removed from the steady-state build gap |

Refresh the baseline and search existing standalone PRs before authoring any listed correction. If one already has the selected fix, reuse its review identity or inherit its landed change. Do not create a second signer-loader, status source or endpoint correction elsewhere in the spine.

## Candidate disposition and Option policy

| Candidate | Decision | Reason or condition |
|---|---|---|
| Nested engine tasks and helper recursion | Keep PR 03 | Ordinary reusable operations establish a clear base |
| Task heap and duplicate dispatch | Keep PR 05 | Replace together with a correct owner, not a blocking direct loop |
| Engine-owned consolidation | Keep | Unsafe matching, replacement and forkchoice need one serialized owner |
| Move consolidation into derivation | Drop | Reversed in source; duplicates execution ownership |
| Applied-safe finality map in engine | Keep PR 11 or absorb into 10 | Removes finalization hop; reuse PR 05's corrected map |
| New separate finality actor | Drop | Unnecessary intermediate owner |
| New Safety/Finality trait hierarchy | Defer | Selected result/context protocol does not need it |
| Full canonical-L1 actor/history/window | Defer | Correct reset lookup and published observations do not require new storage/query machinery |
| Required engine config/providers initialized as None | Eliminate at PR 02 construction | Pass the actual values from the top |
| Optional unsafe-head sender in engine | Remove PR 05 | State publication is unconditional; subscribers vary |
| Optional sequencer in every admin endpoint | Replace PR 07 registration | Base/admin capabilities have concrete dependencies |
| Optional signer inside an enabled signer component | Resolve when constructing it | Preserve actual disabled mode at composition, fail missing configured key |
| Optional conductor client | Real unmanaged/managed policy | Resolve cleanly in PR 08; do not invent no-op clients merely to erase Option |
| Optional discovery/P2P configuration | Real disabled mode | Preserve as composition choice; enabled component gets complete inputs |
| No first L1 observation or unknown trusted progress | Keep legitimate unknownness | Do not manufacture genesis/latest values |
| Optional partial head updates | Keep where meaningful | Not equivalent to missing required dependency |
| Dependency-set defaulting / `f4cad5c24e` | Defer | Changes protocol defaults/activation behavior; a wrapper does not make it harmless type cleanup |
| Remove node aliases, subcommands or env bindings | Defer | Operator surface churn unrelated to selected root causes |
| Delete ignored flags wholesale / `f26a5d1ffc` | Drop as a transplant | It removes conductor/discovery controls that later code reintroduces |
| Remove unsupported max-safe-lag and peer-manager flags | Possible small later policy PR | If removal chosen, remove docs/metric labels/dashboard references together; leave umbrella issues accurately partial |
| Always expose admin APIs | Drop | Option removal does not justify broader exposure |
| Remove recovery-mode RPCs | Drop | Standalone callers/tests still require them |
| Remove dev/ws/health methods or safe-head stub | Defer from core | Public behavior; forwarding can be removed while preserving methods |
| Shared actor builders or new actors crate | Defer | Component handles do not require node-wide packaging/framework changes |
| Delete `NodeActor` or change server restart policy | Defer | Existing supervisor/step model can host selected components; migrate only if a measured/observed lifecycle problem requires it |
| Named lifetime supervision | Conditional minimum support | If a selected component needs a lifetime future, change its owning supervision once; do not port the full historical adapter/token migration |
| RPC middleware request accounting | Optional later observability PR | Preserve feature builds, wire names, bounded unknown-method labels, batch/subscription semantics |
| Unconditional metrics, metric renaming, dashboard rewrite | Defer | Not a prerequisite of operation ownership |
| Utilities `kona-cli::GlobalArgs`/`OverrideArgs` leftovers | Optional small deletion | Distinguish node's live same-named types; not part of engine prerequisites |
| Persistent safe-head history implementation | Separate proposal | Requires its own provenance, rewind, durability and shutdown review; preserve the existing RPC contract here |
| Execution-client internals and shared proof/protocol behavior | Separate work | Keep this stack focused on the independent binary; do not claim node refactors fix unrelated client or proof-program defects |

## Review and verification plan

### Code quality review for every PR

Review the final parent-to-child diff against the [code simplicity and cleanliness principles](#code-simplicity-and-cleanliness-principles) before marking a package ready:

- Names use their module context and established terminology. The normal flow reads from top to bottom with minimal nesting; single-use helpers that add indirection are inlined.
- Every retained trait, wrapper, generic parameter, trait object and optional dependency has a concrete purpose. Constructors provide complete inputs, and mode selection happens at composition.
- State has one authority. Redundant fields and flags are removed, invalid combinations are excluded by construction, and each remaining transition has a necessary role.
- Expected failures have explicit recovery or discard behavior. Unrecoverable failures and unexpected termination of essential components propagate to node shutdown; they cannot leave a stalled actor or an unresolved caller presented as normal operation.
- Configuration, timing and limits are supplied at the appropriate boundary. Obsolete scaffolding and its unused dependencies, tests and documentation are removed with the owning change.

Apply this review within the existing package boundaries. Do not create another cleanup track or revisit a shared interface in several PRs merely to satisfy these criteria.

### One regression home per behavior

Tests follow the surviving owner rather than an experimental crate layout:

- Ordinary operation/walk tests: existing `rust/kona/crates/node/engine/src/task_queue/tasks/*` and `sync/*`, moved next to their surviving functions.
- Engine owner/scheduler/context: `service/src/actors/engine/tests.rs` with the actual handle/owner and scripted dependency responses. Test failure interleavings, not merely that a method constructs the expected enum.
- Pipeline applied context, failed reset, cancellation and repeated-reset backoff: derivation's owning test module. Keep the same cases from PR 05 through 10.
- Registered JSON-RPC contracts: `kona-rpc` modules plus real local jsonrpsee registration, so namespace/method/error/parameter mistakes are caught.
- Sequencer phases: real sequencer state machine with controlled engine/conductor responses and paused Tokio time. Each fake is a dependency seam, not a second implementation of engine policy.
- System behavior: existing `op-acceptance-tests/tests/base/conductor`, `tests/sync/elsync/gap_elp2p`, selected sync suites, and Kona common/restart/reorg tests.

### Non-negotiable interleavings

1. Pause sequencer attributes preparation, perform a same-head reset, resume; old context must be rejected before FCU.
2. Let engine reset finish while the old pipeline has not successfully reset; old work must keep the old epoch and be rejected, not relabelled with a fresh snapshot epoch.
3. Hold an unsafe import on retry and fill data capacity; reset can still enter and reconcile. No old job resumes afterward.
4. Consolidation is awaited while the former derivation callback queue would be full; no callback/result cycle remains.
5. Lose a result receiver after admission, including Stop; the accepted transition still completes correctly.
6. Commit succeeds remotely but the reply is lost; no local publication until positive acknowledgement. After acknowledgement, all import retries use that exact envelope.
7. Stop arrives during commit/import; a late success is reflected in settled head and required gossip enqueue before acknowledgement. No old job initiates later mutations.
8. Applied replacement hash differs from requested/old unsafe hash; only the actual applied block becomes a finality candidate.
9. L1 reorg retains old block by hash; walk rejects it as noncanonical by height. Failed or same-head reset cannot leave stale finality provenance.
10. Output block/tag changes between reads; the proof remains pinned to the fetched block hash.
11. Signer refresh fails without another L1 head, then recovers; same-height reorg during lookup cannot publish the old signer's result.
12. Optional origin preparation warms a different provider clone in a deliberately failing test, then passes only when the actual builder cache is shared.

### Commands and phase gates

These commands are implementation checks, not results of this planning pass. Run from the actual stack checkout, using its pinned mise tools and package membership. There is no `kona-node-actors` prerequisite package in this plan.

From `rust/`, adjust the package list to the PR's changed crates:

```sh
mise exec -- cargo check -p kona-engine -p kona-node-service -p kona-rpc -p kona-node --all-targets
mise exec -- cargo nextest run -p kona-engine -p kona-node-service -p kona-rpc -p kona-node -E '!test(test_online)'
mise exec -- cargo clippy -p kona-engine -p kona-node-service -p kona-rpc -p kona-node --all-targets --all-features -- -D warnings
mise exec -- just fmt-check
```

PR 02 adds `kona-sources`; optional network work adds `kona-gossip`, `kona-disc`, `kona-peers` and affected examples; optional origin work adds the actual provider package. Check relevant default/metrics feature configurations and rustdoc when public APIs/exports change. Preserve the workspace's no-warning policy.

From `op-acceptance-tests/`, use the existing wrapper so dependencies are built before tests:

```sh
DEVSTACK_L2CL_KIND=kona-node DEVSTACK_L2EL_KIND=op-reth RUST_JIT_BUILD=1 mise exec -- just test -parallel=2 ./op-acceptance-tests/tests/base/conductor/...
DEVSTACK_L2CL_KIND=op-node DEVSTACK_L2EL_KIND=op-reth RUST_JIT_BUILD=1 mise exec -- just test -parallel=2 ./op-acceptance-tests/tests/base/conductor/...
DEVSTACK_L2CL_KIND=kona-node DEVSTACK_L2EL_KIND=op-reth RUST_JIT_BUILD=1 mise exec -- just test -run TestInvalidPayloadThroughCLP2P ./op-acceptance-tests/tests/sync/elsync/gap_elp2p
```

The wrapper uses incremental dependency builds and runs Go from the monorepo root. A raw `go test` is appropriate only after its contracts/prestates/binaries are known ready. Do not rely on the stale `rust/kona/tests/justfile` acceptance recipe's nonexistent `optimism/op-acceptance-tests` path. Run the existing Kona node common/restart/reorg wrappers from `rust/kona/tests` when their behavior is touched; keep the selected binary commit explicit.

| Boundary | Required evidence before marking ready |
|---|---|
| Each mechanical/helper/construction PR | Focused check/tests/Clippy/format and relevant feature/docs checks; actual tests for changed observable behavior |
| PR 01 | Invalid-then-valid test against both CLs, real production RPC path |
| PR 04 | Canonical ancestry/coherent-view cases, bounded large-height startup, valid L1 reorg fixtures in both node topologies and existing restart cases; safe-selector parity remains PR 10's responsibility |
| PR 05 | Engine/pipeline interleavings; real short-window force-inclusion recovery; relevant sync/restart cases |
| PRs 06-07 | Registered HTTP/WS/API-gating/version/reset tests; validator and sequencer modes |
| PRs 08-09 | Actual Kona conductor cluster, op-node baseline, no weakened assertions, production routing and durable commit-failure test |
| PRs 10-11 | Startup-mode matrix, canonicality/walk/reset/finality tests and live idle/catch-up/reorg convergence |
| Optional 12-13 | Two-node gossip/discovery checks and delayed-L1 origin transition, with the new interfaces |
| Final selected tip | Workspace check/lint/test/docs, existing required CI jobs, broader Kona acceptance and common/restart/reorg suites; inspect skips and exact binary revisions |

Use the repository's `just lint`, `just test-unit`, `just test-docs` and relevant CI equivalents at the final selected tip. Run the full acceptance variant with bounded concurrency rather than launching many heavyweight local devstacks in parallel. A Kona skip is not a passed regression. Existing source reports of 323 tests or twelve conductor cases do not validate a newly extracted patch.

Node-only changes should not silently alter fault-proof/no_std code. If the selected diff actually changes shared protocol/genesis/proof code, add its cross-target/prestate checks and obtain that review explicitly; do not claim the normal node test set establishes proof equivalence.

## Implementation and PR maintenance procedure

**Complete the remaining draft stack before user review.** Work through the packages in order. Once a package's implementation, agent review and required local validation are complete, publish its draft PR, record the result here, and continue with the next selected package. The user authorized this policy on 2026-10-18. Keep every parent-relative diff independently reviewable. Do not merge the drafts or wait for hosted CI after handoff; report hosted results separately from local validation.

1. Refresh public `develop`, the existing PR comparison metadata and relevant issues with authenticated `gh`. Record the selected base and source tips. Keep the original experimental branches and dirty worktrees intact.
2. Before editing, write the final shared type/method sketch from this plan into the working brief and assign each issue to its ledger home. One implementation owner maintains the spine. Specialists review relevant packages; they do not maintain competing versions of common APIs.
3. Assemble each child from its already curated parent. Use original commits as references or extract selected hunks. Never merge an experimental branch just to resolve dependencies. Do not transplant its old harness or source branch's other fixes implicitly.
4. Review commit intent and parent-to-child diff before testing, including the [code quality review for every PR](#code-quality-review-for-every-pr). The PR body states the concrete failure/ownership rule, resulting behavior, any intended compatibility change, and evidence. A reviewer must understand it without reading the entire experiment.
5. Keep mechanical deletions/moves separately readable inside the owning PR, but do not create a prerequisite cleanup branch that changes the same interfaces again. Avoid unnecessary rename commits and lockfile churn.
6. Run focused checks and required behavior gates. Record exact commands, commit IDs, pass/fail/skip results, and unresolved limitations. Do not reuse old validation claims for a new parent or patch.
7. For existing PRs, preserve review context where practical and rewrite title/body around the final retained scope. Reconstruct 23223's intended small patch; remove 22307's artificial dependency; split or supersede 23268 deliberately. Perform actual GitHub updates only when implementation/PR work is authorized.
8. On user-authorized continuation, address review feedback and restack descendants onto the same accepted parent versions. Use `git range-diff` and inspect `git diff <parent>...<child>` and unique commits to detect accidental old-stack inclusion or lost fixes. Re-run checks when changed parents affect the validated behavior. Continue the authorized stack after each finished draft PR.
9. At each gate, reconsider the remaining packages. Stop when the selected core is simple and the identified bugs are fixed. Optional packages do not become mandatory because a later source branch contains them.

### Review budget and when to combine

PR 05 is intentionally larger than a leaf fix: scheduler, outcomes, reset epochs and pipeline handoff form one consistency boundary. PR 03 removes most low-level mechanical noise first. Keep PR 05's review focused on state transitions and tests, with no packaging/CLI/RPC-method deletion.

PRs 06 and 07 are consecutive but distinct: read dependencies versus mutation dependencies. Combine only if the resulting delta remains smaller and clearer than the two coherent reviews. PR 11 may fold into 10 if moving the already-correct finality map is trivial. Conversely, do not split PR 08 by creating a publication phase model once for commit ordering and a different one for stop/activation; its explicit phase contract is reviewed together, followed by the small override policy in 09.

If any proposed split requires a new adapter, intermediate public enum, callback coalescer, or duplicate retry owner that the next PR deletes, revise the split before implementation. Existing unchanged compatibility interfaces may survive until their owning PR; inventing new temporary ones is the problem.

## Progress checklist

Planning is complete. A check means the package has its actual implementation, review and required local validation, with its implementation commit recorded below. Remaining packages follow the execution policy above.

- [x] Pin baseline/source revisions and inspect existing PR comparisons through authenticated CLI.
- [x] Investigate engine/reset, RPC/conductor, configuration/L1/network and deletion candidates.
- [x] Reconcile the investigations into one stack and review intermediate-state hazards.
- [ ] Reconcile existing standalone fixes with the refreshed baseline, without duplicating them.
- [x] PR 01: synchronous unsafe-payload validation and its acceptance tests.
- [x] PR 02: shared resolved configuration and bounded clients.
- [x] PR 03: ordinary engine operations and necessary dead-code deletion.
- [x] PR 04: canonical reset ancestry and coherent reads; strict safe selection stays with PR 10 bootstrap.
- [x] PR 05: engine ownership, scheduler, honest outcomes, complete pipeline handoff and epoch barrier.
- [x] PR 06: direct read dependencies, version and retained API coverage.
- [x] PR 07: direct admin routing and concrete sequencer dependencies.
- [x] PR 08: conductor publication/activation/stop phase contract and configured timeouts.
- [x] PR 09: local recovery override with shared assertions preserved.
- [x] PR 10: autonomous bootstrap/reorg resets and actual L1 progress publication.
- [x] PR 11: engine-owned accepted-safe finality, independent retries and canonical signal checks.
- [x] Select only the optional PR 12 connect/disconnect deadline and endpoint-alias leaf; defer the broader network/discovery redesign.
- [x] Validate and publish the selected narrow PR 12.
- [x] Defer unmeasured PR 13 origin/receipt prefetch; select only the independent confirmation-policy correction.
- [ ] Validate and publish the narrow confirmation-policy leaf.
- [ ] Final selected-tip validation, issue closure audit and removal of obsolete PR dependencies.

### Decisions to retain during execution

The sole target is an independent kona-node binary that is simple, understandable, correct and robust. Keep its external Engine API and supported RPC/admin behavior. Prefer one source of config/client truth, one engine command owner, one sequencer publication state machine, and one applied pipeline context. Retain narrow permanent crate-boundary traits where they avoid cycles. Deletions caused by those decisions travel with them. Broad CLI, crate, metrics, discovery-dependency and proof-program redesigns remain separate decisions.


## Execution record: PR 01

Completed on 2026-10-17 in branch `codex/kona-refactor-01-unsafe-validation`, based directly on public `develop` at `9c14635375e22608458da1d3ec4830af491009dd`.

- First commit: `d9666c8734`, the approved plan with no implementation changes.
- Implementation: `ccbcc03c5a` (`kona-node: reject malformed admin unsafe payloads`), four source/test files, +116/-17, no dependency changes.
- Source references: 23227 at `862e9f08d83e6ffe88b2405ee56be4df83300af7` and 22307 at `08b5891e02da14d171978b8e7892e6b22fd362bb`. Their branches and PRs are preserved. The new branch includes only the selected fix and coverage, with no old stack dependencies.
- Contract: malformed hashes return JSON-RPC `InvalidParams` (-32602) before queue admission, with op-node's claimed/actual hash message. Valid submissions retain asynchronous admission semantics. The API documentation distinguishes admission from execution and states that submissions are not automatically gossiped.
- Review: separate Rust and Go review agents found no actionable issues. One implementation owner and one validation runner performed the work; reviewers launched no duplicate builds.

### Local validation

| Check | Result |
| --- | --- |
| `cargo nextest run -p kona-rpc --all-features` from `rust/` | 3 passed, 0 skipped, including the registered RPC regression |
| Remove only the new hash guard and rerun the regression, then restore it | Fails as expected because malformed input returns success; restored source passes |
| `just lint` from `rust/` | Passed: formatting, workspace Clippy across features/targets, rustdoc and mirror checks |
| `just lint-go` from the repository root | Passed, including module tidiness |
| `gofmt` on changed Go files; `git diff --check` | Passed |
| Current Kona/op-reth binary builds | Passed |
| Pinned registry bundle, current contract/deploy artifacts, and Cannon prerequisites | Built successfully |

The following acceptance tests each passed against **both** kona-node/op-reth and op-node/op-reth, for six passes and zero skips:

- `TestPostedUnsafePayloadAdvancesVerifier`
- `TestPostedUnsafePayloadRejectsBadBlockHash`
- `TestInvalidPayloadThroughCLP2P`

Run each client selection sequentially from the repository root using `DEVSTACK_L2CL_KIND=kona-node` or `op-node`, `DEVSTACK_L2EL_KIND=op-reth`, `RUST_JIT_BUILD=1` and `RUST_LOG=info`:

```sh
go test -count=1 -timeout 30m -p 1 -parallel 1 -v \
  -run '^(TestPostedUnsafePayloadRejectsBadBlockHash|TestPostedUnsafePayloadAdvancesVerifier)$' \
  ./op-acceptance-tests/tests/base/conductor
go test -count=1 -timeout 30m -p 1 -parallel 1 -v \
  -run '^TestInvalidPayloadThroughCLP2P$' \
  ./op-acceptance-tests/tests/sync/elsync/gap_elp2p
```

The initial acceptance attempt inherited `RUST_LOG=warn`, which hid the info messages used by the harness to discover op-reth's ephemeral RPC ports. It was interrupted before payload assertions; the complete matrix above passed after explicitly enabling info logs. No product or harness code was changed for this environment correction.

The gap test now actually submits the invalid-ancestor payload, then submits a valid payload and independently requires both CL and EL advancement. Together with the registered RPC regression, this covers issue 22250's synchronous rejection and continued valid-payload processing requirements.

**Review checkpoint:** open PR 01 as a draft and stop implementation. Local checks are complete; GitHub CI status must be reported separately. Do not begin PR 02 or run background work on later packages until the user resumes.


## Execution record: PR 02

Completed on 2026-10-17 in `codex/kona-refactor-02-resolved-config`, stacked directly on [PR 01](https://github.com/ethereum-optimism/optimism/pull/23484) at `30fcd89998c72fc6ca3a828bbf88f38fab774872`. Public `develop` remains `9c14635375e22608458da1d3ec4830af491009dd`.

Implementation commit: `72499cf49a` (`kona-node,kona-engine,kona-node-service!: share config and clients`), 22 files, +778/-555, including tests, API examples and removal of obsolete direct dependencies. Existing standalone fixes were used as references; no experimental branch was merged.

- CLI resolution applies overrides to registry and file configurations before constructing the shared `Arc<RollupConfig>`. The resolved chain controls bootstore selection and network rules.
- `RollupNodeBuilder::build` creates the bounded provider pair once and assembles through `build_with_providers`. `OpEngineClient::new` receives those providers and the shared rules. Runtime nodes retain the client and mode rather than connection settings.
- Capability exchange and the initial unsafe-signer lookup run at service startup through those same providers, after local configuration validation. The signer read is pinned to the returned L1 head hash. Existing refresh retries and stale/same-height-reorg guards remain intact.
- `--l2-engine-timeout`, `--l2.timeout` and their existing environment variable now govern complete L2 responses, including body reads. The advertised default is corrected to 10,000ms, preserving the previous effective engine deadline. L1 requests have a separate 10-second deadline.
- Configured signing-key files are read without generating a replacement; missing, unreadable, malformed and invalid-scalar inputs fail. Existing valid key formats and P2P identity loading/creation are retained. JWT file/encoded/default fallback is preserved and tested.
- Network construction receives the resolved rules and a real signer watch receiver. Local network settings no longer carry independent runtime copies of those values.

**Rust API migration.** Replace `EngineClientBuilder` and `OpEngineClient::rpc_client` with `l1_provider`/`l2_provider` and `OpEngineClient::new(l1, l2, config)`. Pass an `Arc<RollupConfig>` to `RollupNodeBuilder`; `EngineConfig` drops duplicate rules/L1 URL and receives a `Duration`. `NetworkConfig::new` no longer takes runtime rules/signer; use `NetworkBuilder::from_config(config, &rules, signer_watch)`. Updated operator examples and the commit/PR breaking-change notes document these changes.

### Local validation

| Check | Result |
| --- | --- |
| Affected-package nextest, all features | 327 passed, 0 skipped |
| Full workspace `just lint` from `rust/` | Passed: Clippy across features/targets, formatting, rustdoc and mirror checks |
| Fresh `cargo build -p kona-node --bin kona-node` | Passed with default features |
| Common wrapper, `simple-kona` | All 3 selected live checks passed |
| Common wrapper, `simple-kona-sequencer` | All 3 selected live checks passed |
| `git diff --check` | Passed |

The package tests cover configured 2-second and 20-second deadlines with missing headers and stalled bodies; JWT authentication; actual provider injection with unusable fallback URLs; shared rule identity; permanent authentication rejection versus transient startup retries; local errors before any startup RPC; registry/file overrides; actual-chain bootstore selection; and required signer-key file behavior. Existing signer refresh/reorg regressions also pass. Review found and corrected a JWT test-server request-body race; final independent Rust and deletion/config reviews have no unresolved findings.

The six live passes are `TestRollupConfig`, `TestP2PChainID` and `TestL2TransactionInclusion` in each configuration. They use the freshly built Kona binary and the unchanged op-reth binary explicitly; existing contract/deployment prerequisites match the unchanged public base.

With the pinned tool environment, run package checks from `rust/`:

```sh
cargo nextest run -p kona-engine -p kona-node-service -p kona-rpc -p kona-node \
  -p kona-sources --all-features --test-threads 6 -E '!test(test_online)'
just lint
cargo build -p kona-node --bin kona-node
```

Run the live wrappers from `rust/kona/tests`, with `RUST_LOG=info` and `RUST_BINARY_PATH_KONA_NODE`/`RUST_BINARY_PATH_OP_RETH` set to the selected binaries:

```sh
just test-e2e-sysgo-run node node/common simple-kona \
  '^(TestRollupConfig|TestP2PChainID|TestL2TransactionInclusion)$ -parallel=1'
just test-e2e-sysgo-run node node/common simple-kona-sequencer \
  '^(TestRollupConfig|TestP2PChainID|TestL2TransactionInclusion)$ -parallel=1'
```

This completes the timeout portion of issue 23034; its discovery/peer requirements remain open. **Review checkpoint:** publish PR 02 as a draft and stop. Report GitHub CI separately; do not begin PR 03 until the user resumes.

## Execution record: PR 03

Implemented on 2026-10-17 in `codex/kona-refactor-03-engine-operations`, stacked directly on [PR 02](https://github.com/ethereum-optimism/optimism/pull/23486) at `1e1516becc9ba0f2e26df76e0f31b43f125fb8a1`. Public `develop` remains `9c14635375e22608458da1d3ec4830af491009dd`. Selected operation changes from [23258](https://github.com/ethereum-optimism/optimism/pull/23258), head `138de49da2dea5630bf7ae7fc8d0d7160eb35260`, were adapted to the current generic provider seam; its old parent stack was not merged.

The implementation has two review units:

- `0da66dba19`: shared engine operations, flat derived build/import and one deposits-only replacement attempt. Existing tasks retain reply handling; the queue retains scheduling and recovery ownership. Reset calls the shared forkchoice operation.
- `bbf81d50c1`: mechanical removal of `need_fcu_call_backup_unsafe_reorg`, `needs_consolidation`, `EngineKind`, `EngineNewPayloadVersion` and `FORKCHOICE_TASK_LABEL`, after whole-tree reference and state-write checks. Live version selection, payload dispatch, canonicality checks, metrics and useful `Display` implementations remain.

**Rust API migration.** `SynchronizeTask` and `BuildSealCoupling` are removed. Use `Engine::reset` for reset synchronization and existing queued operations for ordinary head updates. `SealTask` is sequencer-only; drop its `is_attributes_derived` and `coupling` constructor arguments. Derived attributes go through `ConsolidateTask`. Remove unused kind/backup-state plumbing; `newPayload` dispatch follows the payload envelope. RPC methods, CLI flags and node operating policy are unchanged.

### Local validation

| Check | Result |
| --- | --- |
| Extraction before mechanical cleanup | 336 affected-package tests passed; full workspace lint passed |
| Final affected-package nextest, all features | 335 passed, 0 skipped; see diagnostic below. The deleted selector's own test accounts for the count difference |
| Full workspace `just lint` from `rust/` | Passed: formatting, Clippy across features/targets, rustdoc and mirror checks |
| Fresh default-feature `kona-node` build | Passed |
| Common and restart wrappers, both configurations | All 8 selected live cases passed, without retries or skips |
| Independent Rust/protocol and deletion reviews | No unresolved findings |

Nextest marked the unchanged synchronous `attributes::tests::test_attributes_match` test as LEAK in the full run, after its assertions passed. One isolated diagnostic run passed without that flag. The original full-run diagnostic is retained in this record.

Tests exercise actual mismatching `ConsolidateTask::execute` fallback to build/import with a parent different from the current unsafe head; successful safe and unsafe deposits-only replacement commits heads and sink before Flush; rejected replacement publishes neither. The synchronization matrix preserves status/error classification, state publication and first/no-op FCU behavior. Existing build-version, sequenced stale-parent/reply and hash-pinned finalization coverage remains. These assertions preserve existing rejection policy; the protocol corrections listed under PR 05 still belong there.

Run package checks from `rust/` in the pinned tool environment:

```sh
cargo nextest run -p kona-engine -p kona-node-service -p kona-rpc -p kona-node \
  -p kona-sources --all-features --test-threads 6 -E '!test(test_online)'
just lint
cargo build -p kona-node --bin kona-node
```

Run each live command from `rust/kona/tests` with `RUST_LOG=info` and explicit `RUST_BINARY_PATH_KONA_NODE`/`RUST_BINARY_PATH_OP_RETH` paths. Repeat for `simple-kona-sequencer`:

```sh
just test-e2e-sysgo-run node node/common simple-kona \
  '^(TestEngine|TestL2SafeSync|TestL2TransactionInclusion)$ -parallel=1'
just test-e2e-sysgo-run node node/restart simple-kona \
  '^TestRestartSync$ -parallel=1'
```

**Review checkpoint:** publish PR 03 as a draft and stop all implementation for review. Report GitHub CI separately; do not begin PR 04 or later packages until the user resumes.

## PR 04 execution record

**Implementation:** `a9659f14c3db1cc41590fad7b4c2ae021494ffe8` on `codex/kona-refactor-04-canonical-reset`, based on PR 03's `8d8edf7364885fcdf75a8d8ec9501b98793d4e5f`. Public `develop` remains `9c14635375e22608458da1d3ec4830af491009dd`. Parent PR 03 has no review feedback, 79 successful checks and one skipped Mintlify deployment at the final pre-publication check.

**Result.** A retained old-fork L1 block served by hash no longer validates an unsafe origin. The walk anchors canonical ancestry by number, caches a descending parent cursor and rechecks its numeric anchor before returning. Origins beyond captured L1 visibility remain plausible. Traversed ancestry, sequence, configured genesis, finality and actual unsafe-rewind distance are checked. Missing reads and a changed L1 view retry through the existing reset owner; deterministic invalid structure remains fatal. There is no internal retry loop or new actor. Public function/client signatures are unchanged; exhaustive downstream matches must handle the added `SyncStartError` variants, disclosed by the breaking-change title and commit body.

**Safe-selection cut.** The user approved retaining the existing current-block/recent-window resume policy while correcting its canonicality basis. Safe can still exceed its prior label and need not be an epoch boundary. Applying the stricter selector before establishing an initialization checkpoint would make a synced/imported EL with genesis safety labels walk back to genesis. PR 10 owns that checkpoint contract and the normal-reset prior-safe/epoch-parent selector together; PR 05 consumes this walk unchanged. The stricter proposal is not included in this implementation.

**Regression coverage.** Focused regressions cover retained orphan hashes, plausible ahead origins, historical holes, moving/missing anchors, malformed responses/parents, sequence transitions, genesis/finality, unsafe-rewind limits, AltDA windows and read caching. A real RPC adapter test verifies canonical numeric reads, full L2 blocks, caller-supplied genesis config and label-error compatibility. A compact negative matrix preserves errors outside the unset Safe/Finalized compatibility case. A sparse chain near L2 height 10 million with genesis safe/finalized labels resumes after exactly three parent reads plus three label reads; older history is inaccessible, so a hidden genesis walk fails the test.

**Live fixture correction.** Fixed depth alone did not establish the intended L2 safety relationship. The shallow fixture now waits for nonzero EL safe, drains batching while L1 runs, then produces its unsafe suffix. It asserts that all earlier inclusion history and L1 safety anchors survive, safe origins precede divergence and unsafe origins are replaced. The deeper fixture replaces L2 safe history while preserving L1 safe/finalized anchors. Existing L2 safe/unsafe canonicality, origin and resumed-progress assertions remain intact. No shared FakePoS redesign or assertion relaxation is included.

**Validation.** Root ran all checks serially using pinned tools through the trusted original checkout's `mise exec`, then executed in the isolated worktree. Rust/Go concurrency was bounded to six. Commands, per-step overrides, exit codes and logs are recorded in `work/pr04-validation-results.json` in the task workspace; its runner scripts capture the shared environment.

| Check | Result |
|---|---|
| Go `TestFindSyncStart` reference table | 16 cases passed |
| Rust `just fmt-fix` and Go fixture formatting | Passed |
| Five node packages, all features, sync/reset selection | 43 passed |
| `kona-engine`, `kona-node-service`, `kona-rpc`, `kona-node`, `kona-sources`, all features, offline selection | 367 passed, zero skipped |
| Full Rust workspace `just lint` | Passed, including Clippy, rustdoc and format checks |
| Root `just lint-go` | Passed, zero issues; module tidy diff empty |
| Rust `just check-no-std` | All 19 package builds passed for `riscv32imac-unknown-none-elf` |
| Fresh default `cargo build -p kona-node --bin kona-node` | Passed |
| `TestL2ReorgAfterL1Reorg`, `simple-kona` | Shallow and deep subcases passed |
| `TestL2ReorgAfterL1Reorg`, `simple-kona-sequencer` | Shallow and deep subcases passed |
| `TestRestartSync` and `TestSequencerRestart` | Both passed with the actual Kona sequencer/validator pair |

The reorg presets include reference op-node components; the Kona-sequencer preset adds the op-node sequencer required by the test-sequencer harness. They are mixed-client integration tests. Online unit tests are excluded. The unconditionally skipped `TestL2Reorg` was not selected or counted. The genesis crate changes are two documentation corrections from seconds to L1 blocks; protocol runtime code is unchanged.

**Review.** Independent Rust/protocol, client/actor/API/deletion and Go fixture/flake reviews have no remaining findings within the revised scope. The startup regression was addressed by the approved scope cut and bounded-startup regression; invalid live fixture assumptions were corrected and both topologies rerun successfully. No finding was dismissed as a flaky pass.

**Review checkpoint:** publish PR 04 as a draft on PR 03 and stop all implementation for user review. Hosted CI is separate from the completed local validation; do not claim it is green before its results exist. Do not begin PR 05 or later packages until the user resumes.


### PR 04 test simplification

Following user review, reduce `sync/tests.rs` from 966 to 585 formatted lines. Remove the overlapping 16-row Rust transplant of the Go table, its Case/Expected types, the second fixture layer, duplicate decoded label/genesis/boundary cases and trivial classifier-only assertions. Keep focused regressions for each distinct invariant, including orphan ancestry beneath a plausible ahead suffix, view-change retry, bounded startup, sequence/parent checks, genesis/finality, and AltDA/depth boundaries. The real RPC test covers actual request methods/parameters and nonzero L1/L2 genesis with the caller's config; existing Alloy mocks cover label compatibility concisely.

The original local results above record the initial implementation. The test-only revision reruns formatting, the focused sync/reset selection (37 passed) and all five affected node packages (361 passed, zero skipped), plus full workspace Rust lint (passed). Results are recorded in `work/pr04-trim-validation-results.json` in the task workspace. Production code, Go fixtures and previous live/default-build/no-std validation are unchanged. The user explicitly authorized continuing to PR 05 after this update.

### PR 05 implementation and validation

Branch `codex/kona-refactor-05-engine-owner` starts from updated PR 04 head `6b4323811f0789cd8df80afd405c957a46c91cf5`. The engine task heap and wrappers, operation forwarding clients, callback queue and derivation state machine are replaced together. The engine actor owns bounded scheduling, epochs, reset retries and publication; derivation consumes acknowledged outcomes and records actual completed-safe hashes with inclusion sources. The existing reset walk and bootstrap compatibility remain unchanged.

Sequencer retries retain the original retrieved payload and use paced ticks. Gossip admission is nonblocking; reliable admin ingress retains at most one pending payload without blocking other network events. PR 12 inherits these ingress rules and does not implement them again. Read-only RPC forwarding and the RPC-to-sequencer admin wrapper remain for PRs 06 and 07; conductor acceptance/publication policy remains PR 08.

**Protocol correction.** The review found that the baseline collapsed pending and completed span safety. PR 05 adds a distinct pending-safe head: pre-Holocene prefixes cannot advance completed safe/local-safe or finality. An invalid later element rewinds only pending-safe; the existing batch queue drops its cached span on the changed parent. Holocene uses inclusion-source time and concludes every valid singular element. This correction belongs with the honest `Dropped` outcome and is implemented once here. A rejected deposits-only block is fatal; temporary statuses never trigger a replacement.

**Local validation.**

| Check | Result |
|---|---|
| Engine operations, scheduler, derivation and sequencer regressions | 115 passed |
| Five affected node packages, all features, offline tests | 321 passed, zero skipped |
| Workspace Rust formatting, Clippy, docs and repository lint | Passed |
| Repository custom Go lint and module tidiness | Passed, zero issues and empty tidy diff |
| Fresh default `kona-node` binary build | Passed |
| `TestSequencerRecoversAfterForcedInclusion` | Passed with a Kona sequencer, isolated Kona verifier and external op-reth |
| Existing L1/L2 reorg regression in both presets | Four actual subcases passed |
| `TestRestartSync` and `TestSequencerRestart` | Both passed with the candidate binary |

The force-inclusion test is in the existing `node/reorgs` CI package. It checks an orphaned ordinary unsafe marker, a real L1 deposit becoming canonical on both nodes, resumed batching/sequencing, and safe/finalized agreement on a recovered transaction. The verifier receives one real startup block, then has no L2 peers or further unsafe injection. Idle-start bootstrap remains PR 10. Existing reorg fixtures contain reference op-node components; the separate force-inclusion and restart cases exercise the Kona sequencer directly. Online unit tests are excluded. Shared proof/protocol/alloy runtime code is unchanged, so no new proof-equivalence claim is made.

**Review.** Independent Rust/protocol, Go, scheduler and deletion/API reviews completed. All actionable findings were corrected, including span safety, metric-write preservation, paced exact-job retries and responsive reliable admin ingress. Public documentation and examples use the surviving APIs. Public Rust API changes are marked breaking and have a migration note in the PR.

**Review checkpoint.** Publish PR 05 as a draft on PR 04 and stop implementation for user review. Do not begin PR 06, merge, or wait for hosted CI after handoff. Hosted CI status is separate from the completed local checks above.


### PR 06 implementation and validation

Branch `codex/kona-refactor-06-rpc-reads` starts from PR 05 head `db30a19fd66e1a6bb831e6d6284886b1bfa8eb48`. Rollup reads use the shared L2 provider, resolved configuration and engine state watch directly. Pre-Isthmus proofs use the fetched block's canonical hash; Isthmus outputs use that header's withdrawals root. The binary's cached release version reaches both version and health endpoints. L1 query and admin wiring remain for their owning packages.

The query actor, enums, request channel, queued client and both forwarding/read traits are removed together. The resulting unused engine proof/label methods, their error type and mock helpers are also deleted; the transport deadline regression now uses the surviving full-block read. Dev queue and unsafe/safe/finalized subscriptions retain their wire interfaces and consume the actual owner watches. Publisher closure is handled, and idle client disconnect releases subscription receivers. Public examples and architecture docs use the surviving APIs.

**Local validation.** All 329 offline tests in the five affected node packages passed with all features and zero skips. Full workspace Rust `just lint` passed, including formatting, Clippy, docs and repository checks; the fresh default `kona-node` binary build passed. Registered endpoint tests cover supplied release/development versions, JSON-RPC health and GET `/healthz`, proof parameters, Isthmus/missing/invalid/error paths, stalled-proof concurrency, distinct head subscriptions, publisher closure and idle disconnect. The live `simple-kona-sequencer` common suite passed all four selected cases: `TestRollupConfig`, `TestSyncUnsafe`, `TestSyncSafe` and `TestSyncFinalized`. It exercised the candidate binary against external op-reth, including canonical output comparisons. Commands and results are recorded in the task workspace's `work/pr06-validation-results.json`.

**Review and comparison.** Independent Rust and deletion reviews have no remaining findings. The original RPC experiment remains open as PR 23268; this curated package selects its direct-read/version intent while retaining dev/ws methods and flags. Public Rust API removals and constructor changes have an explicit breaking-change migration note. No Go or shared proof/protocol/alloy runtime source changed. Hosted CI is separate from these completed local results. Continue with PR 07 under the updated execution policy.


### PR 07 implementation and validation

Branch `codex/kona-refactor-07-admin-routing` starts from PR 06 head `3c4d75b44eea7974125067b4235fd589ddda2e5f`. Base admin handlers require the permanent engine capability: reset now works on validators and sequencers, acknowledges completed engine recovery after publication, and bypasses sequencer backpressure. Validated unsafe submission waits for reliable bounded engine admission before returning; it does not wait for execution or gossip. The network admin queue and pending forwarding payload are removed.

The sequencer owns a concrete handle, four private mutation messages, a bounded mailbox and one published active/recovery state. Reads bypass pending commands and dependency I/O; mutations publish state and metrics before replying. Dropped response receivers do not undo accepted commands. Registration handles the real optional sequencer once, preserving the admin enable gate, sequencer-only MethodNotFound behavior and literal `admin_adminRecoverMode` wire spelling. No-argument start, conductor publication and existing stop policy remain unchanged until PR 08. The queued sequencer wrapper, public query enum and reset forwarding are deleted together. Public docs describe the actual owner, admission and acknowledgement boundaries.

**Local validation.** All 59 focused regressions passed. All 305 offline tests in the five affected node packages passed with all features and zero skips. Duplicate direct/channel Cartesian tests were replaced by focused owner-handle cases; distinct failure invariants remain covered. Full workspace Rust lint passed; custom Go lint reported zero issues with an empty module tidy diff. The fresh default binary build passed. Registered tests cover all four admin exposure modes, full sequencer mailbox reads/reset independence, closed channels, state-before-reply, abandoned replies, actual reset retries through L1/L2 outages and a slow read with an unpolled real derivation pipeline, and reliable FIFO unsafe admission before execution.

The new `TestAdminResetPreservesSafeProgress` passed in both `simple-kona-sequencer` and `simple-kona`. It resets the actual Kona roles, requires later safe progress, and compares canonical output blocks across all CL peers, including reference nodes. The existing real `TestPostedUnsafePayloadAdvancesVerifier` and `TestPostedUnsafePayloadRejectsBadBlockHash` both passed with Kona and external op-reth. Exact commands/results are recorded in the task workspace's `work/pr07-validation-results.json`.

**Review.** Independent Rust, Go and deletion reviews have no remaining findings. Fixed findings include the explicit state fixture and non-vacuous cross-client convergence comparison. Public Rust capability/constructor removals have a breaking-change migration note. Hosted CI remains separate from local validation. Continue with PR 08 under the updated execution policy.


### PR 08 implementation and validation

[Draft PR 23492](https://github.com/ethereum-optimism/optimism/pull/23492), branch `codex/kona-refactor-08-conductor-publication`, starts from PR 07 head `62c2d95cf7c615aed6005c859025b6c1c8a1f273`. The sequencer retains a build token until conductor acknowledgement, then pins the exact envelope for import retries. Rejection, timeout or a lost reply cannot canonicalize or gossip a new block. Unmanaged sequencing has explicit local permission. Expired/stale/invalid work is discarded, and temporary retries are paced without admitting another build.

Managed nodes start inactive in production construction. Registered `admin_startSequencer` requires the expected unsafe hash, checks leadership before active state, and reads the head after the leadership await. Domain rejections use -32000; malformed/missing parameters remain -32602 and closed owners remain -32603. The existing whole-seconds conductor timeout reaches the shared complete-response transport for both leader and commit calls.

Stop persists after its RPC caller disappears. Late accepted commits/imports reconcile before acknowledgement; committed retries keep the exact envelope while leadership and context remain valid. Successful engine publication and gossip admission precede the inactive state and returned settled hash. Queued controls run before the following build, and Stop/Start during asynchronous preparation discards those old attributes even if the final state and head match their originals.

**Local validation.** All 82 focused regressions passed. All 328 offline tests across the five affected node packages passed with all features and zero skips. Full workspace Rust lint passed after correcting two Clippy findings. Custom Go lint reported zero issues and an empty module tidy diff; a fresh default binary build passed. The shared `TestRejectedConductorCommitDoesNotPublish` and `TestSequencerStopsAndRestartsAtPreviousHead` both passed with external op-reth on Kona (66.247 seconds total) and op-node (60.895 seconds total). The publication test pauses controller automation while leaving Raft live, proves follower commit rejection, checks both attached EL and CL remain at the settled head across repeated retries, then verifies recovery. Wrong-hash rejection and exact-head restart assertions remain shared across clients. Commands/results are recorded in `work/pr08-validation-results.json` in the task workspace.

**Review and scope.** Independent Rust, Go and deletion/API reviews have no remaining findings. Public admin documentation describes required hash input and Stop settlement. Public Rust constructor/trait changes and the required RPC parameter have an explicit breaking-change migration note. Local override and the full shared conductor suite follow in PR 09; issue 22667 remains partial because max-safe-lag is separate. No shared proof/protocol/alloy runtime source changed. Hosted CI is separate from these local results. Continue without a user approval pause.


### PR 09 implementation and validation

Branch `codex/kona-refactor-09-local-override` starts from PR 08 head `b84e6c9badba6c47587238218f41f5a91311f720`. `admin_overrideLeader` installs process-local permission to bypass conductor leader checks and commits, including committed Stop settlement. It is idempotent, survives an abandoned reply, does not activate sequencing, and clears when constructing a fresh node process. Unmanaged override is harmless. Configured conductor reporting, required expected head, engine epochs and exact-envelope retries remain intact. The external override forwarder and unused error variant are removed.

The shared disaster-recovery scenario retains the pre-override non-leader rejection required by issue 23323. It resumes real production with only the node override, requires the conductor's external override to remain false and its proxy to refuse traffic, then separately enables proxy access with the conductor override. Clearing that external override refuses all five API families again. The Kona skip is removed without changing the shared contract.

**Local validation.** All 330 offline tests in the five affected node packages passed with all features and zero skips. Full workspace Rust lint, custom Go lint/module tidiness (zero issues, empty tidy diff) and a fresh default node build passed. The complete shared conductor suite now has thirteen cases including PR 08's new publication regression; all thirteen passed with zero skips against both Kona/op-reth (333.060 seconds) and op-node/op-reth (298.283 seconds). These include startup, membership, proxy gating, unsafe submission, exact-head stop/restart, publication rejection, handoff, failover, convergence and disaster recovery. Results/commands are in `work/pr09-validation-results.json` in the task workspace.

**Review and scope.** Rust, Go and deletion reviews have no remaining findings. The public override documentation distinguishes process-local permission from the conductor proxy and ordinary Raft leadership recovery. The removed public Rust trait method/error variant have a breaking-change migration note. This completes the selected conductor gate across PRs 01, 08 and 09; the broader umbrella issues remain open. No hosted-CI completion is claimed. Continue directly with PR 10.


### PR 10 implementation and validation

[Draft PR 23494](https://github.com/ethereum-optimism/optimism/pull/23494), implementation commit `283c1d710d96d0b56a7dd440cfd4f5495b652ffc`, branch `codex/kona-refactor-10-engine-bootstrap`, starts from PR 09 head `3924e3fa7c44fe5c1e8975bf8805c9ecd273491c`. The engine initializes from real EL labels and owns canonical-origin checks, reset selection and dependency retries. Readiness requires a current `VALID` forkchoice. A fresh EL that reports syncing until its first forkchoice can confirm the exact configured genesis, including a `SYNCING` then `VALID` response, without redirecting any later advertised/requested sync target. Trusted following retains its exact target until its body and origins are usable.

Initialization preserves the documented once-only bounded trusted-resume assumption when no reusable safe checkpoint exists; canonical history is not local derivation proof. It preserves finalized and creates no local finality record. Later resets retain the completed-safe ceiling and immutable finalized anchor, select a validated epoch parent and tolerate an EL current unsafe below that retained ceiling. Accepted pending-safe remains the pipeline-only reset parent. Local pipeline retries back off without pretending progress or clearing accepted finality; engine epoch changes invalidate context and sequencing origin caches.

Independent L1 head/safe/finalized observations replace RPC query messages. RPC reports actual pipeline or validated delegate progress even with unchanged L2 safe. Unknown values remain optional internally and become the existing zero wire values only at the RPC boundary. Signer retries and stale-result guards remain. Sequencing consumes usable owner context and no longer independently requests startup/origin resets.

**Local validation.** All 357 offline tests across the five affected node packages passed with all features and zero skips. Full workspace Rust lint, custom Go lint/module tidiness (zero issues, empty tidy diff) and a fresh default node build passed. The isolated forced-inclusion/recovery test passed (313.11 seconds), both existing L1/L2 reorg subcases passed in both presets, both restart cases passed, and registered admin reset passed for sequencer and validator. The three selected shared conductor startup, disaster-recovery and exact-head stop/restart cases passed with zero skips on Kona/op-reth (76.794 seconds).

The existing forced-inclusion regression now has no startup unsafe-payload primer: its isolated verifier must bootstrap and derive solely from L1, then independently agree on the forced deposit and recovered safe/finalized chain. No recovery, finality or cross-node assertion was removed. Commands/results are recorded in `work/pr10-validation-results.json` in the task workspace.

**Review and scope.** Independent Rust/protocol, Go and whole-tree deletion/API reviews have no unresolved findings. Corrected findings include pending-safe pipeline reset, local-finality retention, cache invalidation, no-observation origin checks, first-forkchoice startup and retained safe-ceiling recovery. Public Rust changes have an explicit migration note; no shared proof/protocol/alloy runtime source changed. Engine-owned finality follows separately in PR 11. Optional networking is narrowed to the RPC effect/deadline/alias leaf; the separate confirmation-policy correction adds no prefetch. Hosted CI remains separate from these local results. Continue directly without a user approval pause.


### PR 11 implementation and validation

[Draft PR 23495](https://github.com/ethereum-optimism/optimism/pull/23495), implementation commit `d720ed2eecd095d269e3697bae6cfed853eba8f3`, branch `codex/kona-refactor-11-engine-finality`, starts from PR 10 implementation `283c1d710d96d0b56a7dd440cfd4f5495b652ffc` ([draft 23494](https://github.com/ethereum-optimism/optimism/pull/23494)). The existing source-keyed finality map moves unchanged into the engine. Actual completed-safe Applied/Replaced results install the accepted hash and inclusion source before reply, including an abandoned caller; pending prefixes, dropped/stale/failed work and bootstrap trust create no record. Engine epoch invalidation clears records, delegated claims and retries; pipeline-only reset preserves them naturally.

New records and independent finalized-L1 observations both drive finality, including while derivation is idle or a finalized observation remains unchanged. The owner validates the signal itself, inclusion source and canonical L2 hash before FCU. Temporary read failures retain records and retry after 100 milliseconds without new traffic. An uncertain or `SYNCING` forkchoice requires ordinary recovery; proposed finalized state is committed only after `VALID`.

The finalization forwarding command and finalized-only derivation notifications are deleted. Delegated following carries its actual validated safe/finalized pair together; safe acceptance is acknowledged separately from finalized completion in the snapshot. A current claim survives Following's initial epoch invalidation, waits for usable readiness, and never invents local provenance. The remaining head notification carries its real block inline without allocation.

Due canonicality/finality work rotates, with an admitted command/unsafe opportunity between periodic attempts. New head-origin identities receive their mandatory first canonical check before another mutation; due finality then runs before another producer can continually replace those identities. Reset admission retains priority. Automatic finality never decrements admitted queue counts.

**Local validation.** All 373 offline tests across the five affected packages passed with all features and zero skips. After a readability-only Clippy correction, all 44 engine-owner/map regressions passed again. Full workspace Rust lint and a fresh default binary build passed. The isolated sequencer/verifier forced-inclusion test passed through canonical deposit recovery and recovered safe/finalized agreement (258.91 seconds). Registered admin reset passed for sequencer (24.29 seconds) and validator (35.97 seconds).

**Review and scope.** Independent Rust/protocol, deletion/API and follow-up scheduler/unboxing reviews have no unresolved findings. Tests preserve actual replacement hash/span safety, canceled reply ownership, idle/new-record finality, timer retries, canonical signal/source/L2 rejection, epoch clearing, delegated claim lifetime and queue accounting. Paused slow-dependency/slow-command/new-origin regressions pin the maintenance priorities. Public Rust API migrations and operator docs are updated. The parent reorg/restart gates remain recorded at PR 10; the final selected tip receives the full shared Kona conductor preservation run. No Go or shared proof/protocol/alloy runtime source changed. Continue directly to the narrow RPC leaf without waiting for hosted CI or user approval.


### PR 12 selected RPC leaf: implementation and validation

Branch `codex/kona-refactor-12-p2p-effects` starts from PR 11 implementation `d720ed2eecd095d269e3697bae6cfed853eba8f3` ([draft 23495](https://github.com/ethereum-optimism/optimism/pull/23495)). Connect/disconnect now send their mutation once and wait for the promised observed connection effect. A single ten-second deadline encloses mutation admission, peer-query admission/reply and paced false-predicate retries; channel closure is terminal. The canonical `opp2p_blockAddr` endpoint is registered alongside the existing `opp2p_blocAddr` alias. Both mutate the same connection gater.

The RPC-only retry dependency is deleted. Tokio's paused-time support is enabled only for tests; no package version or runtime dependency is added. Registered regressions cover delayed effects, false-until-deadline, each stalled admission/reply phase, combined admission/reply time under one budget, closure, input rejection and actual shared-gater alias behavior.

**Local validation.** All 379 offline tests in the five affected packages passed with all features and zero skips. Full workspace Rust lint and a fresh default binary build passed. The live `TestP2PPeers` and `TestNetworkConnectivity` checks passed with the candidate Kona binary and external op-reth (17.17 and 15.18 seconds). Commands/results are in `work/pr12-validation-results.json` in the task workspace.

**Review and scope.** Two independent Rust/deletion/API reviews have no unresolved findings. Operator documentation describes the observed effect, total deadline and retained alias. No new network actor/handle/snapshot framework, discovery policy or engine-ingress path is introduced. Discovery disablement, peer-manager settings and swarm/discovery responsiveness remain separate; issue 23034 stays open. Continue directly to the confirmation-policy leaf without waiting for CI or user approval.
