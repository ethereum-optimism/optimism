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
the full feature and test-target workload and all ten partitions.

Tool preparation and Cargo downloads are shared. Each compilation profile and
feature partition has its own native target/sccache cache. The source-freshness
helper restores stable timestamps per content revision and trusts a cache only
after successful compilation. Verdict tasks disable result caching, and their
logs/results are excluded from reusable filesystem outputs. Protected develop
warming targets only compilers and command discovery; it executes zero tests.
An actual develop warming event remains a post-merge follow-up.

## Validation

- [x] Report fixtures reject missing/duplicate verdicts, unknown exclusions,
  stale/corrupt archives, incomplete features and unsuccessful stages.
- [x] A real pinned Linux fixture executes archived tests, beacon and doctests,
  retains an intentional failure and verifies fresh success on rerun.
- [x] Helper tests, routing/Circle adapter fixtures, ShellCheck, RWX lint and
  merged/activated Circle config validation.
- [ ] Native execution of all eight workloads and all ten feature partitions.
- [ ] Complete same-SHA Circle/RWX original-report comparison, including ignored
  and online cases, doctests, the beacon regression and observed retry histories.
- [ ] Exactly-once combined feature-plan coverage on each provider.
- [ ] Retain cache reuse observations and final terminal PR checks.

The parity inventory remains 21/86 until hosted execution is verified. Completing
these eight jobs would bring the baseline inventory to 29/86 (34%). WASM,
Cannon-specific Rust jobs, Rust E2E, remaining contract jobs and operational gate
rehearsals remain outside this stage.
