# RWX PR coverage checklist

Maintain this checklist with the [operator guide](rwx-migration.md) and
[evidence retrieval instructions](rwx-evidence-index.md). All work stays in
[draft PR #23151](https://github.com/ethereum-optimism/optimism/pull/23151).

## Scope and denominator

The baseline is [Circle pipeline 135411](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135411),
SHA `587b4c3a73d3f16f15212b72e9a0ed0e8489a7c4`, October 1, 2026.
It contains 86 expanded PR job occurrences: matrix variants and occurrences in
different workflows count separately; native shards do not. The October 5
upstream refresh at `b9ae98c8f1a6cef46576f48831a8e51c53f87023` found no upstream
workflow changes since the pilot base. Refresh the inventory when upstream CI
changes. Setup, schedule checking, protected/post-merge and release work are
outside this denominator.

| Workflow | Implemented and verified occurrences | Baseline |
| --- | ---: | ---: |
| Main | 32 | 32 |
| Contracts | 23 | 23 |
| Rust | 22 | 22 |
| Rust E2E | 9 | 9 |
| Total | 86 | 86 |

**86 / 86 = 100% workload coverage.** Every occurrence has native execution and
resolved same-SHA original-report parity in the retained evidence. This does not
establish provider speed superiority, current-head readiness or required-gate
ownership. Circle remains responsible for all four required gates.

## Upstream changes since the baseline

The October 27 merge of `develop` at `60698d94a5ebfae2df26005f3baf1d358f6c61f5`
changed these PR occurrences. The baseline counts above stay as recorded. A
full inventory refresh must still recount every workflow.

- Retired: `diff-fetcher-forge-artifacts`. #23527 replaced the forge script with
  `eth_call`s and deleted the job and its committed artifacts. The native
  fetcher workload and its shadow check were removed with it.
- Added: `go-actions-tests` (`ci-gate`). #21786 moved `op-e2e/actions` into its
  own job on `op-reth-test-engine`. The native `actions-verdict` task and its
  `main-go` gate mapping are implemented. Hosted same-SHA parity is not yet
  verified, so this occurrence has no parity credit.
- Changed inputs: `go-tests`, `check-nut-prefork-states` and
  `kona-proof-action-single` now require `op-reth-test-engine` (#23648,
  #22632). Both providers supply it from their release producers. Their earlier
  parity evidence predates this input.
- Outside the PR denominator: `contracts-bedrock-build-tests-develop` (#23619)
  runs only on `develop`.

The October 28 merge of `develop` at `68335987fb2780078740c22bed83c771161b3b7f`
changed these PR occurrences:

- Added: the `OP_PRIVATE` contract feature (#23657) in
  `contracts-bedrock-tests`, `contracts-bedrock-tests-heavy-fuzz-modified`,
  `contracts-bedrock-coverage` and `contracts-bedrock-tests-upgrade op-mainnet`
  (`required-contracts-ci`). The four native verdict tasks and their gate
  mappings are implemented. Hosted same-SHA parity is not yet verified, so these
  occurrences have no parity credit.
- Changed routing: `rust_changes_detected` now also matches `op-core/nuts/`
  (#23714).

The October 28 merge of `develop` at `018f03f82fe7e878d78db344f21306717b45f604`
changed these PR occurrences:

- Added: `shfmt-check` (`ci-gate`, #23326). The native `shfmt-check` task runs
  the same `just shfmt-check` recipe with the pinned `shfmt`, and its
  `main-checks` gate mapping is implemented. The pilot's shell scripts are
  formatted to pass it. Hosted same-SHA parity is not yet verified, so this
  occurrence has no parity credit.

The completed workload snapshot is `a1aa49aaf3713a8172f3f615f094488fd8e39c3d`;
its 92 summary hashes and complete provider originals are recoverable through the
evidence guide. Historical per-stage percentages and closeouts are archived.
The last three workload comparisons use `f821983dd56cbd7e488ab903d1ac386a330de7f6`:
seven L2 fork cases and 407 runtime relay requests on each provider, 21 exact
Contracts prerequisites, and all 1,120 private selector rows plus API readbacks.
Earlier failures remain retained and receive no additional coverage credit.

## October 19 develop refresh

Current implementation includes `rust-deny` in the Rust gate and
`opgeth-ratchet` in Main, for 88 PR occurrences (Main 33, Contracts 23, Rust 23,
Rust E2E 9). This updates selection; the 86-occurrence original-report snapshot
above remains historical. Validate the refreshed source on both providers before
assigning the two additions current parity credit.

Feature compilation uses source-bound `features-build-0` through
`features-build-9` producers. All ten `features-N` consumers still execute fresh
original Cargo Hack phases and own the gate verdicts; producers add no workload
occurrences. Count their preparation and transfer in performance comparisons.

## Current Rust execution split

`rust-tests` is implemented by fresh `tests-unit` and `tests-compiler` workers and
the stable `tests` aggregate. Nextest receives only its verified execution archive;
beacon and doctests retain their compiler baseline and cold fallback. All three
native tasks belong to the same baseline occurrence, so the denominator stays 86.
Circle commands and required gates retain their current ownership. Revalidate the
split on the final hosted SHA; historical parity evidence does not establish its
current runtime or cache-transport performance.

## Exact baseline occurrences


Implement and execute the exact expanded CircleCI jobs below. Shared producers
can serve multiple consumers, but validate every occurrence and dependency edge.
The names come from the baseline pipeline, including CircleCI's expanded build
names and feature suffixes.

### Main workflow

CircleCI workflow: `main` (32 jobs).

- [x] `todo-issues-check`
- [x] `shell-check`
- [x] `semgrep-test`
- [x] `semgrep-scan-local`
- [x] `rust-sp1-super-range-executor`
- [x] `rust-op-reth-binary`
- [x] `rust-kona-binaries`
- [x] `rust-binaries-for-sysgo`
- [x] `prep-superchain`
- [x] `prep-go-modules`
- [x] `op-deployer-forge-version`
- [x] `nut-provenance-verify`
- [x] `l2-chains-sync-check`
- [x] `kona-build-sp1-elfs`
- [x] `go-lint`
- [x] `go-binaries-for-sysgo`
- [x] `generate-flaky-tests-report`
- [x] `contracts-bedrock-build-1`
- [x] `contracts-bedrock-upload`
- [x] `diff-fetcher-forge-artifacts`
- [x] `check-op-geth-version`
- [x] `check-nut-prefork-states`
- [x] `check-nut-locks`
- [x] `check-kontrol-build`
- [x] `check-generated-mocks-op-service`
- [x] `check-generated-mocks-op-node`
- [x] `cannon-prestate`
- [x] `go-tests`
- [x] `memory-all-kona-op-reth-fusaka`
- [x] `memory-all-opn-op-reth-fusaka`
- [x] `cannon-go-lint-and-test`
- [x] `ci-gate`
### Contract workflow

CircleCI workflow: `contracts-feature-tests` (23 jobs).

- [x] `prep-go-modules`
- [x] `contracts-bedrock-tests main`
- [x] `contracts-bedrock-tests CUSTOM_GAS_TOKEN`
- [x] `contracts-bedrock-tests OPTIMISM_PORTAL_INTEROP`
- [x] `contracts-bedrock-tests ZK_DISPUTE_GAME`
- [x] `contracts-bedrock-tests-heavy-fuzz-modified main`
- [x] `contracts-bedrock-tests-heavy-fuzz-modified CUSTOM_GAS_TOKEN`
- [x] `contracts-bedrock-tests-heavy-fuzz-modified OPTIMISM_PORTAL_INTEROP`
- [x] `contracts-bedrock-tests-heavy-fuzz-modified ZK_DISPUTE_GAME`
- [x] `contracts-bedrock-coverage main`
- [x] `contracts-bedrock-coverage CUSTOM_GAS_TOKEN`
- [x] `contracts-bedrock-coverage OPTIMISM_PORTAL_INTEROP`
- [x] `contracts-bedrock-coverage ZK_DISPUTE_GAME`
- [x] `contracts-bedrock-tests-upgrade op-mainnet main`
- [x] `contracts-bedrock-tests-upgrade op-mainnet CUSTOM_GAS_TOKEN`
- [x] `contracts-bedrock-tests-upgrade op-mainnet OPTIMISM_PORTAL_INTEROP`
- [x] `contracts-bedrock-tests-upgrade op-mainnet ZK_DISPUTE_GAME`
- [x] `contracts-bedrock-tests-upgrade op-mainnet`
- [x] `contracts-bedrock-tests-upgrade ink-mainnet`
- [x] `contracts-bedrock-tests-upgrade unichain-mainnet`
- [x] `contracts-bedrock-tests-l2-fork op-mainnet`
- [x] `contracts-bedrock-checks-fast-feature-tests`
- [x] `required-contracts-ci`
### Rust workflow

CircleCI workflow: `rust-ci` (22 jobs).

- [x] `rust-fmt`
- [x] `rust-zepter`
- [x] `rust-wasm-wasi`
- [x] `rust-wasm-unknown`
- [x] `rust-udeps`
- [x] `rust-typos`
- [x] `rust-tests`
- [x] `rust-doctest`
- [x] `rust-docs`
- [x] `rust-clippy`
- [x] `rust-check-no-std`
- [x] `rust-cargo-hack`
- [x] `rust-build`
- [x] `op-reth-superchain-snapshot-check`
- [x] `op-reth-integration-tests`
- [x] `op-reth-compact-codec`
- [x] `kona-registry-snapshot-check`
- [x] `kona-lint-cannon`
- [x] `kona-host-client-offline-cannon`
- [x] `kona-build-fpvm-cannon-client`
- [x] `interop-deposits-diff`
- [x] `required-rust-ci`
### Rust E2E workflow

CircleCI workflow: `rust-e2e-ci` (9 jobs).

- [x] `rust-workspace-release`
- [x] `contracts-bedrock-build-2`
- [x] `cannon-prestate`
- [x] `kona-proof-action-single`
- [x] `op-reth-e2e-sysgo-tests`
- [x] `rust-e2e-restart`
- [x] `rust-e2e-simple-kona`
- [x] `rust-e2e-simple-kona-sequencer`
- [x] `required-rust-e2e`

## Maintenance and cutover work

- [x] Chosen 12 Go shards and reusable compile/verdict packages; preserve overlap
  with Rust/prestate builds and runtime-only credentials.
- [x] Shared report I/O/process/byte verification; suite selection and verdict
  rules remain in their owners; filtered import regression retained.
- [x] Historical documentation/probes archived outside Git; three maintained
  guides; non-CI and Circle changes audited in the operator guide.
- [ ] Complete final PR review and exact-head terminal CI before marking ready;
  preserve original failures and attribute them before rerunning.
- [ ] Rehearse native internal PR, unknown/mixed paths, docs-only safe skips,
  merge-group SHAs, superseded runs and exact check association.
- [ ] Rehearse authorized forks and denial of privileged credentials/cache writes
  on unauthorized refs, CLI runs and patched source.
- [ ] Replace Circle setup/continuation and schedule-trigger checking if migrating
  provider orchestration. Do not count them as part of the 86 completed jobs.
- [ ] Validate protected `develop` profiles and actual cache-warming events; CLI
  warm-only rehearsals are not protected-event evidence.
- [ ] Export current rulesets, provider identities, bypasses and legacy branch
  patterns; rehearse safe skips, failed/canceled/missing selected prerequisites
  and required-gate rollback before proposing cutover.
- [ ] Audit production credential/context names, GCP federation, bucket ownership
  and cache access without exporting values. Assign one writer per destination.
- [ ] Replace Circle flake-history dependency and avoid duplicate notifications.
- [ ] Port schedules, maintenance, tag/manual filters and releases separately;
  verify reproducibility, provenance and publishing restrictions before transfer.
- [ ] Retire old provider identities/configuration only after an approved cutover,
  rollback window and legacy branch obligations.

Wall-clock speed remains the optimization target. Keep the chosen configuration;
new tuning needs an identified bottleneck and complete original samples. Include
queueing, setup, transfers and the final verdict, label compiler/cache state, and
leave cost analysis deferred. No speed claim is required for workload completion.

The Rust cache trial preserves the 86-job selection, feature partitions, profiles
and runner resources. It restores source timestamps per unchanged file and
enables incremental compilation within RWX's existing host check/test profiles.
Circle settings remain unchanged. Retain the effective profile overrides and
compiler/cache-transfer timings; hosted correctness and a runtime improvement
are separate claims. The operator guide documents the CLI baseline switch.

## Helper separation and retirement prerequisites

The 86-job denominator and selected workload settings are unchanged by helper
relocation. Permanent execution/policy is in `ops/ci/runtime`; permanent scenario
tests/support are in `ops/ci/tests`. Comparers, Circle alignment/mappings, Insights,
schedule synchronization and the private selector rehearsal are migration-owned.
Separate helper task commands retain existing task keys and workload dependencies.
Native policy version 4 binds `native_policy_sha256`; original report destinations
remain unchanged.

- [ ] Approve required-gate cutover and close the rollback window.
- [ ] Archive final original comparison evidence and matching source with checksums outside Git.
- [ ] Replace or retire Circle Insights, Circle schedule synchronization and the private publication rehearsal.
- [ ] Remove migration tooling/tests/mappings, workflow invocations and migration-only filters.
- [ ] Remove Circle adapters and unused provider metadata/cache compatibility modes.
- [ ] Run permanent tests and native configuration validation after retirement.

Pilot readiness additions (2026-10-06):

- [x] Pin the modified-contract baseline across compilation and verdicts; prove
  branch advancement succeeds and mismatched pins fail before tests execute.
- [x] Complete the bounded changed-crate incremental on/off experiment; keep the
  selected configuration and record the network-transfer measurement boundary.
- [x] Rehearse patched CLI RPC denial, protected cache-write denial, and an actual
  canceled task; retain definitions, terminal states and originals outside Git.
- [ ] Obtain required approving and contract-team reviews after final-head CI.

These observations do not complete the actual fork/merge-queue event or required
gate cutover items above.

- [x] Namespace full Rust test target caches by dependency/compiler inputs after
  the observed cross-lockfile layer-cap failure; ordinary source edits still reuse
  targets. Keep the original failed producer and its successful compilation proof.
- [ ] Measure Rust target-layer transfer separately in future performance work;
  the first corrected namespace uploaded 57,315 MiB in 292 seconds. Keep the
  selected configuration for this stage.
- [x] Restrict the shared cache vault to `develop` writers; PRs, including the
  pilot branch, restore its baseline read-only. Retain the before/after grants.
- [ ] Validate full-test caches with individual target files, excluding the
  compressed transfer artifact and test reports from reusable layers. Keep the
  immutable producer artifact for empty runtime caches. Retain exact tree sizes,
  layer totals, unchanged reuse, source rebuild, failure and fresh-verdict evidence.
  Protected warming of the new namespace starts after merge on `develop`.
- [ ] Investigate the filesystem root cause of restored-rmeta `ESTALE` with RWX.
  Publish independent metadata files as the pilot workaround; retain both
  original failures and label breakpoint observations as diagnostic. Do not
  equate successful cold compilation with untouched warm-update proof.
- [x] Size full Rust test build/runtime disks for restored dependencies and RWX's
  scratch requirement. The 100 GB warm verdict failed before execution at
  `75f51b1427`; use 150 GB while retaining 16 CPU / 32 GiB.
