# CircleCI to RWX parity todos

Track implementation and validation in the single draft
[PR #23151](https://github.com/ethereum-optimism/optimism/pull/23151),
on `codex/rwx-ci-pilot` against `develop`. Update that PR and this checklist
as the port grows. CircleCI continues to own the four required gates.

This checklist covers the full PR workflow. Post-merge, scheduled and release
work appears separately below. Use [rwx-migration.md](rwx-migration.md) for the
migration contract and [ci-comparison.md](ci-comparison.md) for evidence collection.

## Baseline and progress

The October 1, 2026 inventory is
[CircleCI pipeline 135411](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135411)
at commit `587b4c3a73d3f16f15212b72e9a0ed0e8489a7c4`.

| PR workflow | Expanded job instances | Implemented validation jobs |
| --- | ---: | ---: |
| Main | 32 | 1 |
| Contracts | 23 | 4 |
| Rust | 22 | 1 |
| Rust E2E | 9 | 0 |
| Total | 86 | 6 |

Conservative implementation coverage is **6 / 86 = 7%**. Each matrix entry and
each occurrence in a different workflow counts separately; shards do not.
The CircleCI setup and schedule-trigger-check workflows are outside this
86-job denominator and have separate todos. This is job coverage, not runtime,
cost, proven equivalence or gate ownership.

The complete `op-node/rollup/...` shadow covers 16 packages and 1,247 cases,
about 11% of the 11,613 case identities observed in the retained aggregate Go
streams. It is partial coverage of `go-tests`, so that job remains unchecked.
RWX already has checkout, tool and module preparation, routing, and contract
bootstrap tasks; these do not establish complete producer/consumer parity for
the remaining jobs.

A checked inventory item means its complete PR workload has been implemented
and executed as an optional RWX shadow. It does not close the evidence or
operational requirements below. **No required gate has transferred to RWX.**
Refresh the inventory and denominator when the PR workload changes.

## Recommended order

1. Close the existing comparison evidence gaps.
2. Shadow `rust-op-reth-binary` and its integration checks with real outputs.
   The observed CircleCI producer took 9m25s and was the final prerequisite
   holding aggregate Go tests; this identifies a dependency, not a savings estimate.
3. Complete the producers needed by aggregate Go and acceptance tests, then
   port their consumers with equivalent discovery and sharding.
4. Complete the remaining Rust, contract and independent validation jobs.
5. Rehearse full pipeline routing and failure behavior before proposing gate
   changes. Validate post-merge and privileged work before transferring it.

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
- [ ] Repeat the Go baseline with fresh test execution. The retained CircleCI
  streams reused cached results for 17 rollup cases.
- [ ] Preserve per-case retry evidence where available. Foundry reports currently
  lack attempt histories; document unknown histories rather than reporting zero.
- [ ] Collect repeated cold and warm samples on equivalent commits and workloads,
  with actual billed data. Current RWX timings establish only warm observations.

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
- [ ] Record repeated elapsed time, critical path, resources, cache state and
  billing separately before making performance or cost claims.

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
- [ ] `rust-sp1-super-range-executor`
- [ ] `rust-op-reth-binary`
- [ ] `rust-kona-binaries`
- [ ] `rust-binaries-for-sysgo`
- [ ] `prep-superchain`
- [ ] `prep-go-modules`
- [ ] `op-deployer-forge-version`
- [ ] `nut-provenance-verify`
- [ ] `l2-chains-sync-check`
- [ ] `kona-build-sp1-elfs`
- [x] `go-lint`
- [ ] `go-binaries-for-sysgo`
- [ ] `generate-flaky-tests-report`
- [ ] `contracts-bedrock-build-1`
- [ ] `contracts-bedrock-upload`
- [ ] `diff-fetcher-forge-artifacts`
- [ ] `check-op-geth-version`
- [ ] `check-nut-prefork-states`
- [ ] `check-nut-locks`
- [ ] `check-kontrol-build`
- [ ] `check-generated-mocks-op-service`
- [ ] `check-generated-mocks-op-node`
- [ ] `cannon-prestate`
- [ ] `go-tests`
- [ ] `memory-all-kona-op-reth-fusaka`
- [ ] `memory-all-opn-op-reth-fusaka`
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
- [ ] `rust-zepter`
- [ ] `rust-wasm-wasi`
- [ ] `rust-wasm-unknown`
- [ ] `rust-udeps`
- [ ] `rust-typos`
- [ ] `rust-tests`
- [ ] `rust-doctest`
- [ ] `rust-docs`
- [ ] `rust-clippy`
- [ ] `rust-check-no-std`
- [ ] `rust-cargo-hack`
- [ ] `rust-build`
- [ ] `op-reth-superchain-snapshot-check`
- [ ] `op-reth-integration-tests`
- [ ] `op-reth-compact-codec`
- [ ] `kona-registry-snapshot-check`
- [ ] `kona-lint-cannon`
- [ ] `kona-host-client-offline-cannon`
- [ ] `kona-build-fpvm-cannon-client`
- [ ] `interop-deposits-diff`
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
GitHub App push reporting is verified for all three shadows. Earlier Go and
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
