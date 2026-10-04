# Core Rust workspace shadow

Work stays in draft PR #23151. `optimism-rust-shadow` uses shared `run-rust-ci`
routing; Circle retains every required gate. This stage ports eight inventory
jobs, with no performance matrix, publishing or gate migration.

| Circle job | Shared workload | RWX resources |
| --- | --- | --- |
| rust-tests | Workspace/all-features nextest excluding `test_online`, beacon blob bounded-stack regression, and doctests | 16 CPU / 32 GiB |
| rust-doctest | Workspace/all-features/locked doctests, independently executed | 16 CPU / 32 GiB |
| rust-clippy | Workspace/all-targets/all-features/locked, `RUSTFLAGS=-Dwarnings` | 4 CPU / 8 GiB |
| rust-docs | Pinned nightly `just lint-docs`, with its complete rustdoc flags | 8 CPU / 16 GiB |
| rust-build | Mold, dev profile, full workspace, default features | 16 CPU / 32 GiB |
| rust-cargo-hack | Ten partitions: each library feature with no dev dependencies, then isolated test targets with default features | 4 CPU / 8 GiB per partition |
| rust-check-no-std | Every package in the shared Just selection, RISC-V target, no default features | 4 CPU / 8 GiB |
| rust-udeps | Pinned nightly, release/workspace/all-features/all-targets | 4 CPU / 8 GiB |

The unit-test producer uses nextest archives. A fresh worker verifies the archive's
revision, source/settings/toolchain hashes and bytes before execution. It retains
working directories, fixtures, source, Git metadata and the pinned superchain
submodule. Beacon and doctest verdicts remain fresh; doctests compile at execution
because rustdoc does not support precompiled test archives. The separate Circle
doctest job remains a separate workload occurrence.

Nextest's committed timeouts and narrowly scoped two-retry override are preserved.
Original JUnit, discovery JSON, stdout/stderr logs, effective commands, dependency
and tool pins, per-case retry counts, exclusions, hashes and completeness reports
are retained even on failure. Native RWX reporting consumes the original nextest
JUnit and projections of original libtest reports. Missing verdicts are errors,
not inferred skips. Circle keeps its existing flaky-retry alert ownership.

Cargo-hack's shared Just recipes have an optional command-list mode. Both
providers retain exact plans and executed commands. They use the source SHA as
the shuffle seed, replacing Circle's workflow-ID seed so partitions are stable
and directly comparable across providers. This changes assignment, preserving
the full feature and test-target workload and all ten partitions. Discovery uses
an unpartitioned plan: pinned cargo-hack 0.6.44 does not advance partition progress
while printing dry commands. Live global indices establish each command's owner.
Circle's reusable template also retains its default single-node mode.

Tool preparation and Cargo downloads are shared. Each compilation profile and
feature partition has its own native target/sccache cache. The source-freshness
helper restores stable timestamps per content revision and trusts a cache only
after successful compilation. Verdict tasks include RWX run/attempt identities in their cache keys, forcing
fresh execution while preserving tool caches. `cache: false` would also disable
tool caches ([RWX caching documentation](https://www.rwx.com/docs/caching)).
Logs/results are excluded from reusable filesystem outputs. Protected develop
warming targets only compilers and command discovery; it executes zero tests.
An actual develop warming event remains a post-merge follow-up.

## Validation

- [x] Report fixtures reject missing/duplicate verdicts, unknown exclusions,
  stale/corrupt archives, incomplete features and unsuccessful stages.
- [x] A real pinned Linux fixture executes archived tests, beacon and doctests,
  retains an intentional failure and verifies fresh success on rerun.
- [x] Helper tests, routing/Circle adapter fixtures, ShellCheck, RWX lint and
  merged/activated Circle config validation.
- [x] Native execution of all eight workloads and all ten feature partitions.
- [x] Complete same-SHA Circle/RWX original-report comparison, including ignored
  and online cases, doctests, the beacon regression and observed retry histories.
- [x] Exactly-once combined feature-plan coverage on each provider.
- [x] Retain cache reuse observations and final terminal PR checks.

## Hosted closeout

The benchmark revision is `68ad71e05f212ac925a01df72e8dda703c0385fa`:
[RWX run 49dc4302](https://cloud.rwx.com/optimism/runs/49dc43026d3e4364b2f17e0a1bba4a1e)
and [Circle pipeline 135545](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135545).
Both providers passed every workload. Complete package/features/target manifests,
input hashes, tool versions, selections, outcomes, exclusions and observed
retry histories agree. The [evidence index](rwx-rust-evidence/parity.json) records
all job/task IDs and original-file hashes.

| Evidence | CircleCI | RWX |
| --- | ---: | ---: |
| Workspace packages, including packages without tests | 76 | 76 |
| Selected unit cases passed | 3,386 | 3,386 |
| Explicit unit exclusions | 10 | 10 |
| Beacon bounded-stack regression passed | 1 | 1 |
| Doctests per occurrence: passed / ignored | 23 / 17 | 23 / 17 |
| no_std packages completed | 19 | 19 |
| Library feature commands, exactly once across 10 partitions | 352 | 352 |
| Isolated default-feature test-target commands, exactly once | 76 | 76 |
| Observed test retries | 0 | 0 |

The independently scheduled doctest occurrence and the one within `rust-tests`
are both retained. Compile-only `no_run` cases remain distinguished from executed
cases. Coverage files for unit, beacon, both doctest occurrences and no_std are
byte-identical. Rustdoc's annotation and Cargo-home path are normalized while
original discovery/output logs remain unchanged. Circle omits 17 zero-byte
workspace stderr logs from its artifact API; original final manifests declare
SHA256(empty) for those files. All nonempty original inputs are retained.

The 170-test pinned Linux helper suite passes (one opt-in live fixture skipped); the
separate real Rust fixture passes all 17 tests, including archived intentional
failure/fresh rerun, compile-only doctests, empty feature assignment and Circle's
single-node compatibility mode. All 35 routing/Circle adapter fixtures pass.
ShellCheck, RWX lint, merged Circle validation and activated Rust config processing
pass. The benchmark revision's four required Circle gates, dependency review and
six optional RWX checks are terminal and successful.

A targeted [same-SHA native repeat](https://cloud.rwx.com/optimism/runs/54b164d7362c4cc986ecc105cde6d390)
restored `rust-workspace-no-std-v1`, preserved the source fingerprint and executed
all 19 checks freshly with a new run identity. Its command execution was 14s
versus 249s on the initial compiler-cold task. These are functional cache
observations; they exclude setup/transfers and establish no pipeline speed win.

The [first-failure index](rwx-rust-evidence/first-failures.json) preserves the
initial cwd/tool-preparation failures and original report-collector failures.
Their successful corrections do not replace those failed verdicts.

## Remaining work

Coverage is **29/86 = 34%** of the baseline PR inventory, including **12/22 = 55%**
of Rust workflow jobs. Circle retains all required gates. WASM, Cannon-specific
Rust jobs, Rust E2E, remaining contracts and operational gate rehearsals remain
outside this stage.

Further performance tuning is deferred. The initial test target layer was about
39 GiB and the archive about 5.5 GiB; runtime transfers and uneven feature
partitions remain explicit bottlenecks. No runner/shard performance matrix or
RWX-versus-Circle speed claim was made. Observe actual protected-develop warming
after these definitions reach develop; the configured warm-only targets execute
zero workload tests.
