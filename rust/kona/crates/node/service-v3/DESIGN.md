# Kona Node Service V3 Design

This document records the current design direction for the service-v3 refactor. It is an
architectural plan, not a description of an implemented API.

## Goals

Service V3 should:

- separate safe-chain derivation, unsafe-chain acquisition/production, and RPC control;
- keep one authoritative view of execution-layer forkchoice state;
- serialize complete semantic Engine operations rather than individual RPC calls;
- prioritize local sequencing without running derivation and sequencing concurrently over stale
  forkchoice state;
- let a stopped sequencer behave exactly like a P2P follower;
- preserve an exact locally built payload after any potentially ambiguous external side effect;
- give every long-running task explicit ownership and shutdown behavior.

## Top-Level Task Graph

The node has three top-level Tokio tasks:

```text
                         shared stateful Engine
                    Arc<tokio::sync::Mutex<Engine<C>>>
                             ▲               ▲
                             │               │
                  semantic safe ops   semantic unsafe ops
                             │               │
                    ┌────────┴──────┐ ┌──────┴─────────┐
                    │ SafeChain     │ │ UnsafeChain    │
                    │ Builder       │ │ Builder        │
                    └───────────────┘ └────────────────┘
                             ▲               ▲
                             │ handles       │ handles
                             └───────┬───────┘
                                     │
                                  ┌──┴──┐
                                  │ RPC │
                                  └─────┘
```

A node-level composition root constructs these tasks, owns their `JoinHandle`s, supervises
unexpected exits, and performs ordered shutdown.

### `SafeChainBuilder`

`SafeChainBuilder` owns:

- the derivation pipeline;
- its direct L1 reader and canonical L1 cursor;
- an L2 read client used by derivation;
- safe/finalized L2 bookkeeping;
- a clone of the shared stateful Engine.

Its Engine-facing conceptual operations are:

- `L1Reorg`;
- `AdvanceSafeAndFinal`;
- `BuildSafe`.

### `UnsafeChainBuilder`

`UnsafeChainBuilder` owns:

- P2P unsafe payload intake;
- local sequencing state;
- origin selection and payload-attribute construction;
- the conductor client;
- outbound gossip publication;
- sequencer administration;
- a clone of the shared stateful Engine.

Its Engine-facing conceptual operations are:

- `StartBuildUnsafe`;
- `FinishBuildUnsafe`;
- `AcceptUnsafeFromNetwork`.

The unsafe task always constructs its sequencing workflow and starts in follower mode. Local
production begins only after an explicit RPC control request. No configuration boolean separately
models sequencing capability or startup activity.

### RPC

RPC owns transport and routes requests through narrow handles:

- sequencer start/stop/recovery/conductor administration goes to `UnsafeChainBuilderHandle`;
- derivation reset administration goes to `SafeChainBuilderHandle`;
- read-only engine state should come from a watch snapshot or dedicated read adapter rather than
  taking the mutation mutex.

RPC does not implement chain-domain behavior.

## Shared Engine

The engine is not merely a raw HTTP Engine API client. It is a stateful semantic engine:

```rust,ignore
pub struct Engine<Client> {
    client: Arc<Client>,
    config: Arc<RollupConfig>,
    state: EngineState,
    active_unsafe_build: Option<ActiveUnsafeBuild>,
}
```

`Engine` has no internal synchronization and does not know that it is shared. The composition root
accepts an unshared `Engine`, creates an `Arc<tokio::sync::Mutex<_>>`, and passes clones down to the
safe and unsafe tasks. This keeps Tokio synchronization out of the engine implementation and the
public node-construction boundary.

The mutex is the linearization point for complete semantic operations. All reads and mutations of
`EngineState`, and every mutating Engine API sequence derived from it, occur while holding this
same mutex.

The Engine must never maintain separate authoritative state in the safe and unsafe builders.
Builder-local head values are cached expectations only; the state under the Engine mutex is
authoritative.

### Semantic Operation Boundaries

The mutex covers each conceptual operation in its entirety, including all Engine API calls and
state transitions belonging to that operation. It does not merely serialize individual HTTP
requests.

Examples include:

```text
AcceptUnsafeFromNetwork:
    validate candidate -> newPayload -> forkchoiceUpdated -> update EngineState

AdvanceSafeAndFinal:
    consolidate/update labels -> forkchoiceUpdated -> update EngineState

BuildSafe:
    build/get/insert derived payload -> forkchoiceUpdated -> update EngineState

L1Reorg:
    discover recovery forkchoice -> synchronize EL -> update EngineState
```

`EngineState` is updated only after the corresponding FCU result has been successfully accepted.
If `newPayload` succeeds and FCU fails transiently, the implementation retains the operation and
retries the same FCU rather than exposing a partially updated local state or constructing a new
payload.

## Unsafe Build Protocol

`StartBuildUnsafe` and `FinishBuildUnsafe` are separate semantic operations because the EL builds
the payload between them. The start operation returns a build handle that captures its exact
precondition:

```rust,ignore
pub struct UnsafeBuild {
    id: UnsafeBuildId,
    payload_id: PayloadId,
    parent: L2BlockInfo,
    attributes: OpAttributesWithParent,
}
```

The expected parent comes from this handle, not from an independently mutable unsafe-builder
cache.

### `StartBuildUnsafe`

While holding the Engine mutex, start must:

1. read the authoritative unsafe head;
2. validate that the attributes build on that head;
3. execute the appropriate build FCU;
4. create an `UnsafeBuild` containing the parent and payload ID;
5. optionally install an active-build reservation when strict sequencing priority is required.

The mutex is then released while the EL performs the build.

### `FinishBuildUnsafe`

Finish is an optimistic compare-and-commit operation. It acquires the Engine mutex and validates
before retrieving, committing, or publishing anything:

```rust,ignore
let mut engine = self.engine.lock().await;

if engine.state().sync_state.unsafe_head() != build.parent {
    return Ok(FinishOutcome::Stale);
}
if !engine.is_current_build(build.id) {
    return Ok(FinishOutcome::Stale);
}

let payload = engine.get_payload(build.payload_id, &build.attributes).await?;
self.conductor.commit_unsafe_payload(&payload).await?;
self.network.publish_unsafe(payload.clone()).await?;
engine.accept_locally_built_payload(payload).await?; // newPayload + FCU + state update
```

The lock remains held from validation through payload retrieval, conductor commitment, gossip,
payload insertion, FCU, and the local state update. No safe-chain or competing unsafe operation can
change forkchoice after validation and before canonicalization.

### Compatible and Conflicting Interleavings

Safe operations may execute between start and finish. Finish detects whether they invalidated the
build.

| Operation while an unsafe build exists | Required behavior |
| --- | --- |
| Advance safe/final without changing unsafe head | May proceed |
| Build safe and change unsafe head | Defer, or invalidate the unsafe build |
| L1 reorg recovery | Must be allowed to invalidate the unsafe build |
| Network unsafe import while locally sequencing | Suppress or defer |
| Second local build | Reject |
| Finish current local build | Validate and consume its build ID |

A stale-head check is sufficient for correctness. An active-build reservation additionally
provides strict sequencing priority by preventing `BuildSafe` from repeatedly invalidating local
builds. Correctness must not depend on Tokio mutex waiter ordering.

## Distribution and Cancellation Safety

Holding a Tokio mutex across `await` is deliberate here, but it has consequences:

- conductor or gossip stalls also stall derivation and L1 reorg application;
- conductor and network calls must not re-enter or wait on code that needs the same Engine mutex;
- read-only RPC should use snapshots rather than waiting on this mutex;
- shutdown must wait for an accepted distribution action to reach a defined boundary.

The local candidate has conceptual phases:

```text
Built -> Retrieved -> ConductorCommitted -> Gossiped -> Inserted -> Canonicalized
```

Before an external side effect, the build can safely be abandoned as stale. Once conductor commit
or gossip may have succeeded, the exact payload must be retained through retries. The service must
not silently drop it, rebuild a different payload, or allow cancellation to turn an ambiguous
external side effect into a new block action.

Implementation requirements include:

- bounded timeouts around conductor and publication attempts;
- classification of definite versus ambiguous failures;
- retrying ambiguous operations with the exact same payload;
- retrying `newPayload`/FCU with the same candidate after publication;
- treating rejection of a locally built and already published payload as critical;
- making shutdown quiesce and await the in-flight action instead of aborting its future.

Process-crash durability is a separate concern unless persistent candidate storage is added.

## Unsafe Runtime Modes

The unsafe task has only two runtime states:

```rust,ignore
enum UnsafeMode {
    Following,
    Sequencing,
}
```

In `Following` mode it receives P2P payloads and executes `AcceptUnsafeFromNetwork`. Every node
starts on this path.

In `Sequencing` mode it drives local build deadlines and suppresses competing P2P unsafe payloads.
Administration transitions modes only at a safe operation boundary:

- start installs fresh sequencing workflow state and transitions to `Sequencing`;
- stop finishes or explicitly aborts the current unexposed build, clears any reservation, records
  the resulting unsafe hash, and transitions to `Following`;
- shutdown quiesces new work and completes any candidate that has crossed an ambiguous external
  side-effect boundary.

`start_sequencer` and `stop_sequencer` are methods on the cloneable handle. The running unsafe task
owns and applies the mutable transition.

## Error Model

At minimum, distinguish:

- temporary Engine transport failures that retry the exact operation;
- stale unsafe builds that are safely abandoned before distribution;
- invalid P2P unsafe payloads that are dropped;
- invalid locally built/published payloads, which are critical;
- reset-required and flush-required derivation outcomes;
- ambiguous conductor/publication/Engine responses that retain the exact candidate;
- unavailable services and dropped responses;
- terminal invariant failures.

Do not collapse these into a single string error: caller behavior differs materially by category.

## Lifecycle

The composition root should:

1. construct the shared Engine and perform startup forkchoice recovery once;
2. wrap the Engine in a mutex and construct safe and unsafe builders with clones of it;
3. construct narrow safe/unsafe handles and RPC routing;
4. spawn and retain all three top-level task handles;
5. treat unexpected success, error, or panic from any critical task as node failure;
6. stop RPC intake;
7. quiesce unsafe work and finish any externally exposed candidate;
8. stop safe derivation;
9. join every task and report the first failure without detaching children.

## Current Scaffold Status

The first scaffold is implemented in this crate:

- `Engine<Client>` currently owns the raw client, rollup config, and default `EngineState` without
  internal synchronization;
- `RollupNode` creates the shared mutex at the composition root and passes it to the safe and unsafe
  tasks;
- `SafeChainBuilder`, `UnsafeChainBuilder`, and `Rpc` are separate long-running Tokio tasks;
- each task has a bounded control channel and cloneable handle;
- the unsafe task always starts in `Following` and changes mode only through start/stop control;
- `RollupNode::run` starts all three tasks, handles Ctrl-C/SIGTERM, detects unexpected exits, and
  shuts down RPC, unsafe-chain, then safe-chain in order;
- `RollupNode::run_until` provides deterministic embedded/test shutdown;
- lifecycle and unsafe-mode transition tests are present;
- metrics and concrete chain workflows are not implemented.

The task loops currently process lifecycle/control messages only. No derivation, P2P, conductor,
RPC-server, startup-reconciliation, or semantic Engine operation is implemented yet.

## Implementation Plan

### 1. Crate and core types

The crate root, task types, handles, shared Engine scaffold, and basic supervisor exist. Remaining
core-type work is to:

- keep composition-owned synchronization out of the Engine implementation as operations are added;
- add authoritative state watches and typed semantic errors;
- wrap existing `kona-engine` tasks where they already implement the required protocol mechanics;
- avoid duplicating raw Engine API version-selection logic unnecessarily.

### 2. Semantic Engine operations

Implement and unit test:

- startup forkchoice recovery;
- `AcceptUnsafeFromNetwork`;
- `AdvanceSafeAndFinal`;
- `BuildSafe`;
- `L1Reorg`;
- `StartBuildUnsafe`;
- `FinishBuildUnsafe`;
- unsafe-build reservation, invalidation, and stale detection.

Each operation should expose a semantic result rather than leaking raw task errors.

### 3. `UnsafeChainBuilder`

- Port origin selection and attributes construction.
- Port conductor integration and exact-payload publication behavior.
- Integrate P2P intake and publication without giving Network ownership of Engine state.
- Implement following/sequencing transitions and administration.
- Reuse the existing network builder/handler and signer tracking where practical.

### 4. `SafeChainBuilder`

- Port local and delegated derivation as required.
- Own pipeline reset, L1 cursor validation, finality mapping, and reorg detection.
- Translate derivation outcomes into the three safe semantic Engine operations.
- Preserve flush/reset behavior for invalid derived payload recovery.

### 5. RPC and supervision

- Route admin methods through safe and unsafe handles.
- Publish coherent read-only Engine snapshots for rollup/dev/websocket APIs.
- Add explicit startup handshakes, task supervision, and ordered shutdown.

### 6. Core compatibility

- Match the existing consensus-critical Engine, derivation, sequencing, and follower behavior.
- Keep service V1/V2 available as behavioral references until core parity is demonstrated.

Metrics and other observability parity are explicitly deferred until after the core implementation
is complete.

## Required Tests

In addition to ordinary unit tests, add deterministic concurrency tests covering:

- safe/final advancement between unsafe start and finish without changing unsafe, followed by a
  successful finish;
- `BuildSafe` changing unsafe between start and finish, producing `Stale` with no conductor or
  gossip side effect;
- L1 reorg invalidating an active unsafe build;
- an active-build reservation deferring conflicting safe construction;
- the finish lock preventing safe operations from interleaving between validation, conductor,
  gossip, `newPayload`, and FCU;
- a stopped sequencer importing P2P payloads exactly like a validator;
- starting sequencing suppressing competing P2P unsafe imports;
- stopping sequencing at each candidate phase;
- conductor and gossip ambiguity retrying the same payload hash;
- transient FCU failure after successful `newPayload` retrying the same forkchoice transition;
- invalid network payload drop versus invalid local payload terminal failure;
- shutdown during planning, building, conductor commitment, gossip, insertion, and FCU;
- no task detachment or channel-closed failure during ordered shutdown.

A recording mock Engine client should assert exact Engine API call order and payload identity across
retries. Integration tests should then verify validator following, sequencer production, start/stop,
derivation consolidation, reorg recovery, and restart behavior against an EL.
