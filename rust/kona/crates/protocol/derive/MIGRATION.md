# kona-derive migration: async → pure deriver

This document tracks the migration corridor opened by Phase 1 of the
pure-derivation refactor. It exists for the duration of phases 1–6b; once
the async pipeline is deleted in phase 6b it can be removed.

See `plans/2026-05-06-refactor-kona-pure-derivation-plan.md` in op-claude
for the full plan.

## Phase 4 status (current PR)

kona-node migrated to `NodeDeriver` wrapping `kona_derive::Deriver`:

- `crates/node/service/src/actors/derivation/deriver.rs` — new async driver
  owning `AlloyChainProvider` + `AlloyL2ChainProvider` +
  `OnlineBlobProvider`. Translates `DeriveTrace` → `tracing::*`.
- `crates/node/service/src/actors/derivation/actor.rs` — `DerivationActor`
  no longer generic over `Pipeline + SignalReceiver`; takes a concrete
  `NodeDeriver`. The `Signal::Activation` activation site collapsed to a
  no-op (Holocene+ strict ordering subsumes it).
- `crates/node/service/src/actors/derivation/state_machine.rs` — collapsed
  the `Signal*` state-machine arms (`AwaitingSignal`,
  `AwaitingUpdateAfterSignal`, `SignalNeeded`, `SignalProcessed`). The
  remaining 4 states (`AwaitingELSyncCompletion` / `AwaitingL1Data` /
  `AwaitingSafeHeadConfirmation` / `Deriving`) are still required for
  EL-sync gating and attribute-confirmation flow.
- `crates/providers/providers-alloy/src/pipeline.rs` — **DELETED.**
  No remaining in-tree caller of `OnlinePipeline`.
- `crates/providers/providers-alloy/src/blob_decode.rs` — new module,
  lifted from `kona_derive::sources::blob_data`. Phase 5's blob/calldata
  helper requirement satisfied early since kona-node also needs it.
- `crates/providers/providers-alloy/src/l1_data.rs` — new helper
  `fetch_l1_block_data` that produces ready-to-feed `Vec<L1TxView>` for
  `kona_derive::extract_l1_input`.
- `crates/providers/providers-alloy/src/chain_provider.rs` — exposes
  `block_info_by_number` as an inherent method on `AlloyChainProvider`;
  the `ChainProvider` trait impl delegates to it. Lets `p2p.rs` drop
  the `kona_derive::ChainProvider` trait import.
- `bin/node/src/flags/p2p.rs` — uses concrete `AlloyChainProvider` now;
  trait import removed.

## Phase 1 state (initial gating)

`kona-derive` now exposes two compile modes:

- **`--features async`** (default) — the existing pipeline, traits, stages,
  sources, and async test mocks. Today's behavior, byte-identical.
- **`--no-default-features`** — only the sync surface: `PipelineError`,
  `PipelineErrorKind`, `ResetError`, `BuilderError`, `BatchDecompressionError`,
  `BlobDecodingError`, `BlobProviderError`, `PipelineEncodingError`,
  `Signal`, `ResetSignal`, `ActivationSignal`, `StepResult`, `PipelineResult`,
  `Metrics`. Nothing here pulls in `async-trait`.

The workspace dep at `rust/Cargo.toml` was flipped from
`default-features = false` to inheriting defaults, so every existing consumer
continues to compile with `async` on. No consumer Cargo.toml needs touching
in this PR.

### Deviation from plan wording

The plan lists `AttributesBuilder` (trait) and `StatefulAttributesBuilder`
(struct) as "sync items" that stay unconditional. Both are async-trait based
today (`AttributesBuilder::prepare_payload_attributes` is `async fn`,
`StatefulAttributesBuilder` only exposes that trait impl). Making them
unconditional in Phase 1 would require keeping `async-trait` as a non-optional
dependency, defeating the gate. They're therefore behind `cfg(feature = "async")`
here; **Phase 2** carves out the sync math into `core::attributes` and the
trait becomes unconditional then. The plan's wording is forward-looking to
that state.

## Consumer inventory

Every in-tree consumer of `kona_derive` async items. Phase column lists the
phase that retires that site (or migrates it onto `pure::Deriver`).

### `kona-derive::{Pipeline, SignalReceiver, Stage}` — pipeline-driver surface

| Site | Phase |
|------|-------|
| `crates/node/service/src/actors/derivation/actor.rs` (step loop, signal handling) | 4 |
| `crates/node/service/src/actors/derivation/state_machine.rs` (entire `Signal*` state machine) | 4 (collapsed) |
| `crates/node/service/src/actors/derivation/delegated/actor.rs` | 4 |
| `crates/node/service/src/actors/derivation/request.rs` | 4 |
| `crates/node/service/src/actors/engine/{client.rs,engine_request_processor.rs}` (Signal forwarding) | 4 |
| `crates/proof/driver/src/core.rs` (`Pipeline + SignalReceiver` bound on driver) | 5 |
| `crates/proof/driver/src/pipeline.rs` (`DriverPipeline::produce_payload`) | 5 |

### `kona-derive::{OnlinePipeline, OraclePipeline, EthereumDataSource}` — concrete drivers

| Site | Phase |
|------|-------|
| `crates/providers/providers-alloy/src/pipeline.rs` (`OnlinePipeline::{new_polled,new_indexed}`) | 4 (deleted) |
| `crates/proof/proof/src/l1/pipeline.rs` (`OraclePipeline::new`) | 5 (deleted) |
| `crates/proof/proof/src/l1/mod.rs` re-exports (`ProviderDerivationPipeline`, `ProviderAttributesBuilder`) | 5 (deleted) |
| `bin/client/src/single.rs` (`EthereumDataSource` use) | 5 |
| `bin/client/src/interop/{mod.rs,transition.rs}` (`EthereumDataSource` use) | 5 |
| `bin/host/src/interop/handler.rs` (`EthereumDataSource` use) | 5 |
| `bin/host/src/error.rs` (`From<PipelineError>` impl for `HostError`) | 5 (stays — error type) |

### `kona-derive::{BlobProvider, ChainProvider, L2ChainProvider, DataAvailabilityProvider}` — provider traits

| Site (`impl` block, not call site) | Phase |
|---|---|
| `crates/providers/providers-alloy/src/chain_provider.rs` (`AlloyChainProvider` `impl ChainProvider`) | 6b (trait deleted; struct stays as concrete fetcher) |
| `crates/providers/providers-alloy/src/l2_chain_provider.rs` (`AlloyL2ChainProvider` `impl L2ChainProvider`) | 6b (same) |
| `crates/providers/providers-alloy/src/blobs.rs` (`OnlineBlobProvider` `impl BlobProvider`) | 6b (same) |
| `crates/proof/proof/src/l1/chain_provider.rs` (`OracleL1ChainProvider` `impl ChainProvider`) | 6b |
| `crates/proof/proof/src/l1/blob_provider.rs` (`OracleBlobProvider` `impl BlobProvider`) | 6b |
| `crates/proof/proof/src/l2/chain_provider.rs` (`OracleL2ChainProvider` `impl L2ChainProvider`) | 6b |
| `crates/proof/proof/src/sync.rs` (uses `ChainProvider` trait for sysconfig walkback) | 5 (migrate to direct method calls) |
| `crates/providers/providers-local/src/buffered.rs` (`BufferedL2Provider` `impl L2ChainProvider`) | see below |
| `bin/node/src/flags/p2p.rs` (`ChainProvider` trait bound on a fetch helper) | 4 (drop bound, use concrete `AlloyChainProvider`) |

### `kona-derive::{AttributesBuilder, StatefulAttributesBuilder}` — attribute building

| Site | Phase |
|------|-------|
| `crates/node/service/src/service/node.rs` (`StatefulAttributesBuilder` construction) | 4 → driver-owned |
| `crates/node/service/src/service/builder.rs` (docs reference) | 4 |
| `crates/node/service/src/actors/sequencer/{actor.rs,metrics.rs,admin_api_impl.rs}` (`AttributesBuilder` trait use for sequencing) | 4 (depends on phase 2 carve-out making the trait sync) |
| `crates/proof/proof/src/l1/pipeline.rs` (`ProviderAttributesBuilder` alias) | 5 (deleted with `OraclePipeline`) |
| `crates/providers/providers-alloy/src/pipeline.rs` (`OnlineAttributesBuilder` alias) | 4 (deleted with `OnlinePipeline`) |
| `bin/client/src/single.rs` (`StatefulAttributesBuilder` doc reference) | 5 |

### `kona-derive::{Signal, ResetSignal, ActivationSignal, StepResult}` — already-sync glue

These stay live through phases 1–5 as the actor's coordination vocabulary
and are deleted in **6b** when `Pipeline::signal` and `Stage::step` go away.

### `kona-derive::test_utils::*` — async test mocks

| Site | Phase |
|------|-------|
| `crates/node/service/Cargo.toml` `dev-dependencies` (`kona-derive = { features = ["test-utils"] }`) | 6b (dep dropped) |
| `crates/node/service/src/actors/sequencer/tests/{test_util.rs,actor_test.rs}` (`TestAttributesBuilder`) | 4 (replaced with concrete sync builder or driver-level test fixtures) |

### `kona-derive::Metrics` — metrics constants

| Site | Phase |
|------|-------|
| `bin/node/src/flags/metrics.rs` (`Metrics::init`) | 6b (metrics module deleted; drivers re-emit from `DeriveTrace`) |

## providers-local fate

**Decision: retire.** `crates/providers/providers-local/src/buffered.rs`
exposes a single public type, `BufferedL2Provider`, which implements
`kona_derive::L2ChainProvider` and `kona_protocol::BatchValidationProvider`.
Its only callers are `tests/integration.rs` inside the same crate. No other
workspace crate depends on `providers-local`:

```
$ grep -rn 'kona-providers-local\|providers_local' --include='*.toml' --include='*.rs' rust/ | grep -v 'providers/providers-local/'
(no matches)
```

When the async traits go away in phase 6b there is nothing to migrate the
crate onto — its job (an in-memory mock for the async pipeline) disappears
with the async pipeline. The pure deriver doesn't take L2 lookups via a
trait; it requests them explicitly via `Derivation::NeedSpanBatchOverlap`,
so caller-side caches like `BufferedL2Provider` become a driver concern, not
a `kona-derive` concern.

The crate gets **deleted** as part of phase 6b. The cargo-machete check in
that PR confirms no remaining consumers.

## Integration suites under `rust/kona/tests/{node,supervisor}`

These are **Go** test suites (`*_test.go`) exercising the kona-node and
kona-supervisor binaries end-to-end. They do not import `kona_derive`
directly. They continue to run unchanged through phases 1–5 and act as
black-box regression coverage for the migrated driver in phase 4.

## Things that are NOT migrated

Out of scope per the plan, listed here so they don't get forgotten:

- `op-reth`, `kona-rbuilder`, `kona-supernode` — do not depend on
  `kona-derive`.
- `kona-comp` (batcher compression) — unrelated to derivation.
- Pre-Holocene derivation paths — pinned to the async pipeline through
  phase 6b, then deleted. No migration path for callers needing pre-Holocene
  rules; this is a hard scope boundary in the plan.
