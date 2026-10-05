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

The October 5 refresh observes `develop` at
`b9ae98c8f1a6cef46576f48831a8e51c53f87023`, with no upstream Circle or GitHub
workflow changes since the pilot's `c8e4ba85` base. The PR denominator remains 86.

All **86 / 86 baseline PR job occurrences** are implemented, fully executed on
native RWX and supported by resolved same-SHA original-report parity. The last
three jobs were verified together at `f821983dd56cbd7e488ab903d1ac386a330de7f6`:
[OP Mainnet L2 fork](rwx-contract-l2-fork-evidence/parity.json),
[Contracts aggregate](rwx-pr-gates-evidence/contracts-parity.json) and
[selector uploader](rwx-selector-upload-evidence/f821-batch-parity.json).

Circle pipeline 135650 and the native coordinator pass all seven initial L2
cases, with zero skips or diagnostic reruns. Both retain all 407 runtime relay
requests, zero transport retries and zero HTTP 429s. Both genuine Contracts
gates pass the same 21 exact prerequisites. The selector comparison verifies the
complete compiler catalogues, all 1,120 fresh database rows and API readbacks.
The [coverage closeout](rwx-pr-gates-evidence/f821-coverage-closeout.json) retains
complete original hashes, all four successful required Circle gates, dependency
review and the remaining native check failures from this batch.

Full workload parity and final-head readiness are separate. The same batch's
native Main observer rejected a retry after a Go toolchain download timed out;
a separate Rust E2E source clone reached its limit after a GitHub HTTP 504.
Complete first failure logs remain retained. The gate now follows current
engine-bound receipts and retains independent task attempts. Twenty-seven real
Git/YQ gate and original-report fixtures pass, including a failed receipt,
successful retry and later failing retry. Final head checks are tracked in the
single draft PR; Circle remains responsible for required gates.

Earlier combined failures remain retained in the
[preparation timeout](rwx-pr-gates-evidence/contracts-first-batch.json),
[public RPC and module-proxy failures](rwx-pr-gates-evidence/contracts-second-batch.json),
[backoff-only failures](rwx-pr-gates-evidence/contracts-pacing-preflight.json) and
[shell/source correction](rwx-pr-gates-evidence/contracts-shell-preflight.json).
Circle's zero-request relay report at `3be585cb` resulted from nested Bash
reloading `BASH_ENV`, which restored the public endpoint over the loopback URL.
The earlier warm-cache attribution was incorrect. Its corrected historical
[L2 comparison](rwx-contract-l2-fork-evidence/3be5-batch-parity.json),
[Contracts comparison](rwx-pr-gates-evidence/3be5-contracts-parity.json) and
[check snapshot](rwx-pr-gates-evidence/pr-closeout.json) remain preserved.
The verified `f821983d` comparisons supersede that closeout. These task
observations establish no provider speed claim.

| PR workflow | Expanded job instances | Verified job occurrences |
| --- | ---: | ---: |
| Main | 32 | 32 |
| Contracts | 23 | 23 |
| Rust | 22 | 22 |
| Rust E2E | 9 | 9 |
| Total | 86 | 86 |

Verified implementation coverage is **86 / 86 = 100%**. Each matrix entry and
each occurrence in a different workflow counts separately; shards do not.
The CircleCI setup and schedule-trigger-check workflows are outside this
86-job denominator and have separate todos. This is occurrence coverage backed
by retained parity evidence; it does not measure runtime, cost or gate ownership.

The full aggregate Go shadow now covers all 459 selected packages and 11,613
case identities, with fresh same-SHA Circle/RWX parity verified. Its Go,
superchain, contracts, Kona and prestate producers are implemented and executed.
Main's complete Cannon Go workload now passes original-report parity for
all 16 packages and 2,881 case identities at `5ee3311d`. See the
[Cannon Go closeout](rwx-cannon-go.md). NUT pre-fork regeneration now passes full same-SHA original-report parity
for karst and lagoon, with all six actual cases passing and native reporting
correctly displaying six cases. See the [pre-fork closeout](rwx-nut-prefork.md).

Fetcher artifact validation now passes full same-SHA original-report parity at
`6245472e`, including all 729 compiler artifacts and all four untouched embedded
artifacts. See the [fetcher closeout](rwx-fetcher-artifacts.md).

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
All nine Rust E2E occurrences now pass complete same-SHA original-report parity:
76 release packages, 88 compiler units, 16 binaries, contract/prestate runtime
inputs and 410 test identities (405 passes, five skips, no retries). All fourteen
native verdict shards and the aggregate passed. See the
[E2E closeout](rwx-rust-e2e-parity.md) for original hashes, the two narrowly resolved
logger-timestamp differences, retained first failures, and compile-only warming.
Complete module preparation and all sixteen contract-fast checks now pass
same-SHA original-report parity. Both prep occurrences cover all 463 modules;
the Main occurrence was already counted. See the [PR-check closeout](rwx-pr-checks-parity.md)
for the two additional contract occurrences and retained first failures.
All seven L1 upgrade occurrences now pass complete same-SHA original-report
parity. Their full 1,359-signature selection retains four abstract declarations
with empty compiler bytecode and accounts for every one of the 1,355 executable
cases. See the [upgrade closeout](rwx-contract-upgrades.md) for original hashes,
whole-contract setup skips, pinned archive blocks and retained first failures.
All seven additional Main validators now pass complete same-SHA original-report
parity, including both fresh mock generators and their verified pinned superchain
bundle. See the [Main closeout](rwx-main-checks.md) for original hashes, complete
selection and preparation-only evidence.
The four standard and four changed-file heavy-fuzz variants now pass complete
same-SHA original-report parity. Standard discovery retains all 164 files and
2,882 signatures; heavy discovery retains both actual modified files and all
28 cases. See the [contract suite closeout](rwx-contract-suites.md) for exact
originals, generated fixture evidence, compiler-only reuse and the skip-message
follow-up. All four coverage variants now pass complete same-SHA ordinary and
upgrade comparison, including every LCOV hit and complete per-test attribution.
The unchanged-input warm rehearsal reused all four producers with zero tests.
See the [coverage closeout](rwx-contract-coverage.md) for the retained first
failure, source-bound corrections and full original hashes.
ShellCheck and both Semgrep jobs now pass complete same-SHA comparison with
exact original selections, pinned tools, baseline provenance, warnings and
exclusions. Hosted real-tool failure fixtures pass. The original missing-PATH
failure and corrected native definition rehearsal remain retained. The pushed
PATH fix passed the native PR-check shadow on `d8e7d3ec`; all four Circle gates,
dependency review and native checks also finished successfully on that head.
Every subsequent final head needs terminal verification. See the
[static-check closeout](rwx-static-checks.md).
SP1's complete native guest workload now passes same-SHA parity for both ELF
binaries, CPU verification keys, all six cases and complete dependency graphs.
Kontrol's full summary/proof build passes all four complete compiler inventories,
including all historical payloads, source graphs and cache settings. See the
[SP1 closeout](rwx-sp1-guest.md) and [Kontrol closeout](rwx-kontrol-build.md).
The full Rust aggregate now passes complete same-SHA original gate comparison at
`68f13331`, including all 21 exact Circle dependencies, 30 fresh native workload
tasks, original verifier output and both terminal GitHub checks. See the
[gate closeout](rwx-pr-gates.md). The full Main aggregate also passes complete
same-SHA original gate comparison at `4ae28fd9`, including every one of its 19
dependencies, twelve Go and sixteen acceptance shards, all ten receipts and
both terminal GitHub gate checks. The complete L2 fork and Contracts aggregate
now pass their corrected combined comparison at `f821983d`. Circle continues
to own every required gate.
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
2. Retain the completed Rust E2E and Cannon workload evidence.
3. Rehearse operational failure/routing behavior and verify the remaining report
   refinements below. All baseline contract job occurrences are complete.
4. Keep additional performance tuning deferred unless a new stage requests it.
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

## Historical gaps and remaining report refinements

At `587b4c3a`, the retained comparison found matching Go results
(1,245 pass / 2 skip) and contract results (9,503 pass / 638 skip).
Those historical comparisons were incomplete. The full original-report
comparisons linked above supersede them. The remaining unchecked items below
track report refinements and measurements beyond verified job occurrence coverage:

- [x] Retain authoritative CircleCI Go package and contract test-file manifests,
  including packages without test files, at the tested revision. Full Go and
  standard/modified contract originals now retain complete manifests.
- [x] Retain effective CircleCI profile, feature, filter, fuzz and invariant
  settings for the implemented Go and standard/modified contract shadows.
  Runtime dumps and original invocation histories are retained separately.
- [ ] Add stable reasons for the 178 contract skips whose reasons are unavailable,
  preserving the existing guards. The full `c702cfb2` originals supersede the
  older observation: only two Interop L1Block skips lack emitted reasons, and
  their exact feature guard is retained. Explicit messages are added; verify
  them in the next hosted originals before closing this item.
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

These rules apply to each inventory item. Its closeout document records the
validation and limitations. They are not additional job occurrences. Repeated
performance measurements remain conditional on making a speed claim.

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
- [x] `nut-provenance-verify` — [complete recorded-source runner](rwx-nut-provenance.md), full same-SHA hosted parity at `ad48c5ad`; both historical regenerations, source/compiler graphs, tool binaries and generated bytes agree, with zero retries
- [x] `l2-chains-sync-check`
- [x] `kona-build-sp1-elfs` — [complete native guest build and checks](rwx-sp1-guest.md); full same-SHA hosted parity at `c6b29406`, both ELFs and verification keys byte-identical, six cases and complete graphs agree
- [x] `go-lint`
- [x] `go-binaries-for-sysgo`
- [x] `generate-flaky-tests-report` — [native original reporter](rwx-flaky-report.md), complete same-SHA API/report parity at `b472a22e`, all 79 original observations and 12 acceptance rows, no retries
- [x] `contracts-bedrock-build-1`
- [x] `contracts-bedrock-upload` — [complete isolated uploader](rwx-selector-upload.md), strict same-SHA original parity at `63844aed`, all 1,120 signatures, complete initial and stable compiler catalogues, private API and fresh database readback
- [x] `diff-fetcher-forge-artifacts` — [fresh build and untouched artifact comparison](rwx-fetcher-artifacts.md)
- [x] `check-op-geth-version`
- [x] `check-nut-prefork-states` — [complete original-report parity](rwx-nut-prefork.md)
  passes all six original cases on both forks at `fd426d87`, with complete same-SHA parity.
- [x] `check-nut-locks`
- [x] `check-kontrol-build` — [full summary generation and proof build](rwx-kontrol-build.md); full same-SHA hosted parity at `c6b29406`, both summaries and all four complete compiler inventories agree
- [x] `check-generated-mocks-op-service`
- [x] `check-generated-mocks-op-node`
- [x] `cannon-prestate`
- [x] `go-tests`
- [x] `memory-all-kona-op-reth-fusaka`
- [x] `memory-all-opn-op-reth-fusaka`
- [x] `cannon-go-lint-and-test` — [complete original-report parity](rwx-cannon-go.md)
  passes for all 16 packages and 2,881 case identities at `5ee3311d`.
- [x] `ci-gate` — [exact native aggregate](rwx-pr-gates.md) passes full same-SHA original gate parity for all 19 terminal dependencies, including twelve Go and sixteen acceptance shards, with both gate checks successful

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
- [x] `contracts-bedrock-tests-l2-fork op-mainnet` — [complete L2 closeout](rwx-contract-l2-fork.md); corrected same-SHA parity at `f821983d`, all seven initial cases and all 407 runtime relay requests on both providers
- [x] `contracts-bedrock-checks-fast-feature-tests`
- [x] `required-contracts-ci` — [complete aggregate closeout](rwx-pr-gates.md); corrected same-SHA parity at `f821983d`, all 21 exact prerequisites and genuine fresh final statuses

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
- [x] `required-rust-ci` — [exact native aggregate](rwx-pr-gates.md); complete same-SHA original gate parity and terminal GitHub checks at `68f13331`, all 21 dependencies and 30 fresh native tasks

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

## Routing and required gates

Shared routing policy and its regression tests are implemented. Automatic
GitHub App push reporting is verified for all eight existing optional checks. Earlier Go and
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
