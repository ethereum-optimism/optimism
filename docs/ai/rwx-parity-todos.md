# CircleCI to RWX parity todos

Track implementation and validation in the single draft
[PR #23151](https://github.com/ethereum-optimism/optimism/pull/23151),
on `codex/rwx-ci-pilot` against `develop`. Update that PR and this checklist
as the port grows. CircleCI continues to own the four required gates.

This checklist covers the full PR workflow. Post-merge, scheduled and release
work appears separately below. Use [rwx-migration.md](rwx-migration.md) for the
migration contract and [ci-comparison.md](ci-comparison.md) for evidence collection.

## Optimization target

Optimize wall-clock time from a push to the final CI verdict, with the existing
coverage and fresh test execution preserved. During shadowing, measure completion
of all selected shadow workloads. Include queueing, setup, cache restoration,
artifact transfers, builds and the slowest test shard. Label run-start timings
explicitly when push or queue timestamps are unavailable.

Use repeated samples to track median latency and slow runs. Compiler execution
and summed task time explain bottlenecks; they are not the headline metric.
Prioritize the measured critical path: reuse valid producer outputs, restore
targets and sccache, run independent work in parallel, minimize transfers and
increase runner resources where that reduces elapsed time. Record actual resource
allocations and cache state; equal resource allocations are not a pilot constraint.
Cost analysis is deferred. Future concurrency changes need evidence of faster
execution without greater cost; performance tuning is outside this closeout.

## Baseline and progress

The October 1, 2026 inventory is
[CircleCI pipeline 135411](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135411)
at commit `587b4c3a73d3f16f15212b72e9a0ed0e8489a7c4`.

| PR workflow | Expanded job instances | Implemented validation jobs |
| --- | ---: | ---: |
| Main | 32 | 13 |
| Contracts | 23 | 4 |
| Rust | 22 | 21 |
| Rust E2E | 9 | 0 |
| Total | 86 | 38 |

Conservative implementation coverage is **38 / 86 = 44%**. Each matrix entry and
each occurrence in a different workflow counts separately; shards do not.
The CircleCI setup and schedule-trigger-check workflows are outside this
86-job denominator and have separate todos. This is job coverage, not runtime,
cost, proven equivalence or gate ownership.

The full aggregate Go shadow now covers all 459 selected packages and 11,613
case identities, with fresh same-SHA Circle/RWX parity verified. Its Go,
superchain, contracts, Kona and prestate producers are implemented and executed.
The full Fusaka acceptance variants and their remaining SP1/Cannon dependency
edges have now executed successfully on both providers. See the
[acceptance closeout](rwx-acceptance-parity.md) for exact original-report evidence.
The eight core Rust workspace jobs now have complete same-SHA parity, including
all 76 workspace packages and exactly-once coverage across ten feature partitions.
See the [Rust closeout](rwx-rust-parity.md) for original reports, cache evidence and
remaining limits. Both WASM package sets, Zepter, Typos, the Kona registry
snapshot check and the full Interop differential test now pass complete
same-SHA original-report parity. See the
[remaining Rust stage](rwx-rust-extra-parity.md) for retained evidence.
All three Cannon Rust workloads now pass complete same-SHA original-report
parity, including both MIPS client binaries and the complete final VM state.
See the [Cannon evidence](rwx-cannon-parity.md).
The Rust E2E definition is now implemented for the full workspace release build,
its separate `ci --skip test` contract artifacts, all reproducible prestates and
the five fresh Go test workloads. Hosted execution and complete same-SHA original
report comparison remain pending, so none of its nine occurrences is counted yet.
The narrower rollup mode remains CLI-only. See [the stage closeout](rwx-go-parity.md)
for original-report comparison, selected resources, caches and limitations.

A checked inventory item means its complete PR workload has been implemented
and executed as an optional RWX shadow. It does not close the evidence or
operational requirements below. **No required gate has transferred to RWX.**
Refresh the inventory and denominator when the PR workload changes.

The complete op-reth shadow passed in
[native RWX run b2a03116](https://cloud.rwx.com/optimism/runs/b2a0311680d04079ac335e1439e75a7e)
at `d2b9024161341a61fa48dcfa2f177e92651d220d`. Both release binaries were
verified; all 50 runnable integration cases passed, with one explicitly ignored
case; fresh compact vectors and regenerated superchain snapshots passed. The
original integration JUnit matched CircleCI job 5625702 for all 50 reported
identities and outcomes. Cache measurements and their remaining limitations are
in [rwx-migration.md](rwx-migration.md#op-reth-shadow-and-cache-measurements).

## Recommended order

1. Retain the completed [acceptance](rwx-acceptance-parity.md) and
   [core Rust workspace](rwx-rust-parity.md) shadows and their original-report evidence.
2. Port the Rust E2E workflow and retain the completed Cannon workload evidence.
3. Close remaining contract evidence gaps and operational failure/routing rehearsals.
4. Keep additional performance tuning deferred while porting remaining workloads.
5. Rehearse full pipeline routing and failure behavior before proposing gate
   changes. Validate post-merge and privileged work before transferring it.

## RWX speed priorities

Implementation stays in PR #23151. Measure the final shadow verdict from run
start, and retain push/queue time separately when available. Cost is deferred.

- [x] Split shared, Go, Go lint, Rust and Foundry tool layers. Bootstrap tools
  through small artifacts instead of inheriting Git history.
- [x] Separate Go and Foundry compilation from fresh tests. Add isolated native
  compiler caches and transfer runtime artifacts without compiler cache layers.
- [x] Balance the twelve aggregate Go shards using observed package durations. Preserve
  exhaustive discovery, package working directories, retries and all test flags.
- [x] Configure protected `develop` cache-rebuild targets for compilers/tool
  preparation, with no verdicts or publishing side effects.
- [x] Measure supported runner sizes and locality with retained whole-run
  observations. Release compilation improved in the 16-CPU trial; native
  observations are recorded in the migration guide. Refresh warm-run medians
  and required CircleCI gate results in the PR on each final pushed revision.
- [ ] Observe an actual native cache-rebuild event after these definitions reach
  `develop`; a CLI warm-only rehearsal does not establish this.
- [x] Document Captain's current lack of Go partitioning support; retain the
  exhaustive duration-balanced manifest and native RWX reporting.
- [x] Close full Go parity with 12 shards / parallel 8. Repeated performance
  samples and 24-shard selection are deferred at the user's request.

## Close the current evidence gaps

At `587b4c3a`, the retained comparison found matching Go results
(1,245 pass / 2 skip) and contract results (9,503 pass / 638 skip).
Both comparisons remain incomplete. These are the tasks needed to strengthen
the existing shadows:

- [ ] Retain authoritative CircleCI Go package and contract test-file manifests,
  including packages without test files, at the tested revision.
- [ ] Retain effective CircleCI profile, feature, filter, fuzz and invariant
  settings; distinguish runtime dumps from settings declared in configuration.
- [ ] Add stable reasons for the 178 contract skips whose reasons are unavailable,
  preserving the existing guards. Source inspection suggests 116 originate in
  `skipIfUnoptimized()`; verify that inference through original reports.
- [x] Repeat the complete Go baseline with fresh execution at the same SHA.
  All 11,613 identities match; seven environmental skip-message differences
  were investigated. The earlier cached rollup baseline is superseded.
- [ ] Preserve per-case retry evidence where available. Foundry reports currently
  lack attempt histories; document unknown histories rather than reporting zero.
- [ ] Collect repeated whole-workload CircleCI and RWX timings on the same commit
  and workload, including routing, queueing and setup. Snapshot probes already
  include three cold-compiler and three restored-target samples: median whole-run
  wall time was 126.3s versus 55.6s, with fresh regeneration. Common image/tool
  layers were warm; these samples establish a cache benefit within RWX.

## Completion requirements for every ported workload

- [ ] Record the CircleCI command, dependencies, profiles, features, filters,
  shards, resources, timeouts, outputs and credential names without secret values.
- [ ] Preserve behavior affected by CircleCI variables or CLI commands, including
  dependency builds, timing-based splits and Docker cleanup. Replace provider
  dependencies explicitly instead of setting compatibility variables alone.
- [ ] Retain complete discovery and prove every selected case or package is
  assigned exactly once. Fail discovery and report collection on missing data.
- [ ] Bind producer outputs and consumer reports to the tested SHA and settings.
  Transfer contracts, binaries, prestates and the superchain bundle explicitly.
- [ ] Execute verdicts freshly during comparisons; verify dependency/build cache
  inputs and invalidate them on relevant source, toolchain or profile changes.
- [ ] Compare original reports on the same SHA and verified routing context,
  retaining actual provider triggers, first verdicts, skips and observed retries.
- [ ] Retain accessible reports, logs, traces and configuration after failures.
  Successful diagnostic reruns must not replace the original failed verdict.
- [ ] Exercise build/test failure, cancellation and a dependency that never ran;
  verify the workflow and proposed gate cannot report success for those cases.
- [ ] Record repeated end-to-end wall time, critical path, actual resources and
  cache state before making speed claims. Keep phase timings separate; cost
  analysis is deferred for the pilot.

Apply these requirements to each inventory item. Record run/job links and
remaining limitations in PR #23151 when checking an item.

## PR workload inventory

Implement and execute the exact expanded CircleCI jobs below. Shared producers
can serve multiple consumers, but validate every occurrence and dependency edge.
The names come from the baseline pipeline, including CircleCI's expanded build
names and feature suffixes.

### Main workflow

CircleCI workflow: `main` (32 jobs).

- [ ] `todo-issues-check`
- [ ] `shell-check`
- [ ] `semgrep-test`
- [ ] `semgrep-scan-local`
- [x] `rust-sp1-super-range-executor`
- [x] `rust-op-reth-binary`
- [x] `rust-kona-binaries`
- [x] `rust-binaries-for-sysgo`
- [x] `prep-superchain`
- [x] `prep-go-modules`
- [ ] `op-deployer-forge-version`
- [ ] `nut-provenance-verify`
- [ ] `l2-chains-sync-check`
- [ ] `kona-build-sp1-elfs`
- [x] `go-lint`
- [x] `go-binaries-for-sysgo`
- [ ] `generate-flaky-tests-report`
- [x] `contracts-bedrock-build-1`
- [ ] `contracts-bedrock-upload`
- [ ] `diff-fetcher-forge-artifacts`
- [ ] `check-op-geth-version`
- [ ] `check-nut-prefork-states`
- [ ] `check-nut-locks`
- [ ] `check-kontrol-build`
- [ ] `check-generated-mocks-op-service`
- [ ] `check-generated-mocks-op-node`
- [x] `cannon-prestate`
- [x] `go-tests`
- [x] `memory-all-kona-op-reth-fusaka`
- [x] `memory-all-opn-op-reth-fusaka`
- [ ] `cannon-go-lint-and-test`
- [ ] `ci-gate`

### Contract workflow

CircleCI workflow: `contracts-feature-tests` (23 jobs).

- [ ] `prep-go-modules`
- [x] `contracts-bedrock-tests main`
- [x] `contracts-bedrock-tests CUSTOM_GAS_TOKEN`
- [x] `contracts-bedrock-tests OPTIMISM_PORTAL_INTEROP`
- [x] `contracts-bedrock-tests ZK_DISPUTE_GAME`
- [ ] `contracts-bedrock-tests-heavy-fuzz-modified main`
- [ ] `contracts-bedrock-tests-heavy-fuzz-modified CUSTOM_GAS_TOKEN`
- [ ] `contracts-bedrock-tests-heavy-fuzz-modified OPTIMISM_PORTAL_INTEROP`
- [ ] `contracts-bedrock-tests-heavy-fuzz-modified ZK_DISPUTE_GAME`
- [ ] `contracts-bedrock-coverage main`
- [ ] `contracts-bedrock-coverage CUSTOM_GAS_TOKEN`
- [ ] `contracts-bedrock-coverage OPTIMISM_PORTAL_INTEROP`
- [ ] `contracts-bedrock-coverage ZK_DISPUTE_GAME`
- [ ] `contracts-bedrock-tests-upgrade op-mainnet main`
- [ ] `contracts-bedrock-tests-upgrade op-mainnet CUSTOM_GAS_TOKEN`
- [ ] `contracts-bedrock-tests-upgrade op-mainnet OPTIMISM_PORTAL_INTEROP`
- [ ] `contracts-bedrock-tests-upgrade op-mainnet ZK_DISPUTE_GAME`
- [ ] `contracts-bedrock-tests-upgrade op-mainnet`
- [ ] `contracts-bedrock-tests-upgrade ink-mainnet`
- [ ] `contracts-bedrock-tests-upgrade unichain-mainnet`
- [ ] `contracts-bedrock-tests-l2-fork op-mainnet`
- [ ] `contracts-bedrock-checks-fast-feature-tests`
- [ ] `required-contracts-ci`

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
- [ ] `required-rust-ci`

### Rust E2E workflow

CircleCI workflow: `rust-e2e-ci` (9 jobs).

- [ ] `rust-workspace-release`
- [ ] `contracts-bedrock-build-2`
- [ ] `cannon-prestate`
- [ ] `kona-proof-action-single`
- [ ] `op-reth-e2e-sysgo-tests`
- [ ] `rust-e2e-restart`
- [ ] `rust-e2e-simple-kona`
- [ ] `rust-e2e-simple-kona-sequencer`
- [ ] `required-rust-e2e`

## Routing and required gates

Shared routing policy and its regression tests are implemented. Automatic
GitHub App push reporting is verified for all six optional checks. Earlier Go and
contract failure probes verified bounded failure reporting; the full native
pipeline still needs the rehearsals below.

- [ ] Replace CircleCI setup/continuation orchestration and the separate
  `circleci-schedule-trigger-check` validation while preserving their behavior.
- [ ] Exercise internal PRs, mixed/unknown changed paths, docs-only safe skips,
  merge-group SHAs and superseded runs through native RWX reporting.
- [ ] Rehearse authorized fork commits through Bailiff's `external-fork/*` path.
  Preserve human authorization of the exact SHA; a new fork commit needs new
  authorization. Verify denied paths cannot obtain privileged credentials or
  write trusted caches.
- [ ] Define RWX gate dependencies by exact job and matrix names. Include failed,
  canceled and not-run prerequisites in terminal failure propagation.
- [ ] Ensure intentional safe skips emit every proposed required context, and
  verify check association with the current PR or merge-group commit.
- [ ] Export current GitHub rulesets, provider identities, branch patterns and
  bypass rules. Cover legacy backport/proposal branches or retain CircleCI there.
- [ ] Rehearse restoring the previous required checks and provider ownership.
  Keep CircleCI runnable throughout the rollback window.
- [ ] Propose a ruleset change only after complete workload and operational
  evidence. Required CircleCI gates remain unchanged until an approved cutover.

## Identity and artifact ownership

- [ ] Audit credential names, context restrictions, GCP federation and bucket
  permissions; map equivalent RWX access without exporting secret values.
- [ ] Validate separate cache reader/writer access, or an equivalent cache design.
  Run denial checks for CLI runs, local patches and unauthorized refs before
  transferring the current `develop` cache writer.
- [ ] Preserve permissions and provenance for PR artifacts and protected writes,
  including contract output uploads. Keep one active owner per destination.
- [ ] Replace CircleCI flake-history/reporting dependencies and retain useful
  failure notifications without creating duplicate notifications or writes.

## Beyond PR parity

These are later migration tasks and are outside the PR-job percentage.

- [ ] Verify full `develop` execution, including optimized contract profiles.
- [ ] Port tag and manual-dispatch filters, schedules and maintenance automation.
  Resolve the two audited schedules without routing entries with their owners.
- [ ] Validate release-candidate artifacts, reproducibility, provenance and
  identity restrictions before transferring release publishers.
- [ ] Transfer each trigger and publishing destination to one active owner;
  rehearse rollback without duplicate maintenance or publication.
- [ ] Update CI operations, review, authorization and publishing runbooks, then
  retire obsolete configuration and identities after the rollback window and
  legacy branch obligations.
