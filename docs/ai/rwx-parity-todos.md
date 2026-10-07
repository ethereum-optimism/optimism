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

The completed workload snapshot is `a1aa49aaf3713a8172f3f615f094488fd8e39c3d`;
its 92 summary hashes and complete provider originals are recoverable through the
evidence guide. Historical per-stage percentages and closeouts are archived.
The last three workload comparisons use `f821983dd56cbd7e488ab903d1ac386a330de7f6`:
seven L2 fork cases and 407 runtime relay requests on each provider, 21 exact
Contracts prerequisites, and all 1,120 private selector rows plus API readbacks.
Earlier failures remain retained and receive no additional coverage credit.

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
- [x] Narrow Rust compiler-cache outputs after the ordinary-rebuild layer-cap
  failure. Complete archives retain every test executable; compiler caches retain
  libraries and incremental work. The real Linux fixture rebuilds a committed
  source edit and runs fresh verdicts from the archive. Retain final cold/warm
  hosted observations separately in the evidence collection.
- [ ] Investigate the filesystem root cause of restored-rmeta `ESTALE` with RWX.
  Publish independent metadata files as the pilot workaround; retain both
  original failures and label breakpoint observations as diagnostic. Do not
  equate successful cold compilation with untouched warm-update proof.
