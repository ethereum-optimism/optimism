# Contract coverage shadow

The optional `optimism-contract-coverage-shadow` implements all four configured
coverage feature variants through the existing shared contracts routing. These
four occurrences remain **uncounted** until complete hosted original-report
comparison passes. Implementation coverage remains 67/86 (78%), with 19 remaining.
Circle retains every required gate and production publisher.

The shared adapter expands the original `just coverage-lcov-all` into its exact
ordered, short-circuiting recipes: `just coverage-lcov`, followed by
`just coverage-lcov-upgrade --match-contract "OPContractsManager.*_Upgrade_Test"`.
The upgrade recipe still runs the production `prepare-upgrade-env` and its
`test/{L1,dispute,cannon}/**` filter, ten fork retries and 1,000 ms backoff. The
original default-profile `just build-source` prerequisite stays separate from
the `cicoverage` verdict profile: optimizer disabled, no compilation restrictions,
16 threads, one fuzz run, one invariant run and depth one. Go FFI, recursive
submodules, source paths, fixtures, commit/branch metadata and pinned tools remain
available. Inherited filters and feature switches are cleared and the selected
feature is applied explicitly.

Each pass adds Foundry's `--report attribution` alongside LCOV. The pinned Forge
1.8.3 emits original per-execution JSON containing identities, outcomes and every
covered source item/hit. Compatible invariant predicates share one campaign and
one attribution row. For multiple file reports, it ignores `--report-file`
and writes the default `lcov.info`; the adapter retains each pass independently
before restoring the original public `lcov.info` and `lcov-upgrade.info` paths.
Reports are retained even after an initial failure. A diagnostic `just test-rerun`
cannot replace the original outcome or overwrite its coverage evidence. Empty
failure caches cannot accidentally run the whole suite as a diagnostic.

Coverage's human-readable original verdict log is validated against its complete
machine attribution. Native test reporting uses a clearly named **derived**
JUnit view of those originals. No result is fabricated, and comparison rechecks
every derived case, status and skip reason against both original sources.
Ordinary/upgrade prefixes keep reporting identities distinct even when both
passes execute the same Solidity case. The full original identities stay intact.
LCOV entries and complete attribution items are compared without discarding hit counts
or branch details. The comparison index fingerprints every complete attribution
row, including all items and fields, while full originals remain retained.
Compiler signatures, abstract empty-bytecode declarations and
whole-contract setup skips account for every selected case.

The first hosted run at `463983e15a8817d6894f3c17a67e5de7ab6683fa` failed in
the shared report parser on both providers. All four native producers passed;
the ordinary coverage command exited zero. Main's original engine summary and
attribution contain 2,477 execution records. Three of those records are merged
invariant campaigns with twelve separately reported predicates. The parser now
retains all twelve canonical predicate verdicts, binds each complete campaign to
its actual attribution anchor, and independently validates the original engine
totals. Main therefore has 2,486 predicate/unit verdict identities at this point;
whole-contract setup skips receive their usual complete selection accounting.
Missing members, duplicate names, absent anchors, wrong kinds, unclosed campaigns
and contradictory engine totals fail validation. Campaign coverage stays shared;
the report does not invent individual covered-item lists for merged predicates.
This behavior follows the pinned
[campaign selection](https://github.com/foundry-rs/foundry/blob/cae51ad458f6abb64852b7709eb784352429825d/crates/forge/src/runner.rs#L399)
and [attribution serializer](https://github.com/foundry-rs/foundry/blob/cae51ad458f6abb64852b7709eb784352429825d/crates/forge/src/coverage.rs#L329).
Real pinned Forge fixtures now cover multiple predicates and a failed campaign.
The upgrade pass had not started at the first failure;
this observation does not establish full coverage parity. Original first-failure
reports remain under `.ci/rwx-contract-coverage-evidence/4639/` and immutable
provider artifacts, including both complete ordinary attribution reports.
The [first-failure index](rwx-contract-coverage-evidence/first-failure.json) retains
both complete original seals, canonical predicate events, full changed attribution
items and the source-bound diagnosis. All 150 original files verify against their
seals. Complete selection accounting agrees: 2,308 passes and 551 skips, including
whole-contract setup skips. This remains ordinary-pass evidence, with upgrade absent.

The opt-in Circle parameter `c-contract_coverage_replay` defaults to false and
preserves its usual unseeded coverage execution. RWX uses a seed derived from the
full tested SHA and feature. A same-SHA comparison must opt Circle into this same
replay seed; comparison rejects unseeded or different-input benchmark reports.
The first complete comparison also found two independent sources of hit-count
differences: `ForgeArtifacts.ensurePath` loops over an absolute output path, and
eight trie tests use Go FFI OS randomness outside Foundry's seeded generator.
Native coverage now clones into `project/`, matching Circle's checkout depth using
[git/clone's supported path parameter](https://www.rwx.com/docs/rwx/packages/git/clone).
All artifact, compiler-cache and report paths follow that checkout; working
directories and fixtures remain relative to the real repository root.
Opt-in replay also binds `OP_CI_FFI_REPLAY_SEED` to the same benchmark seed. Only the
FFI trie helper accepts it, requiring `CI=true` and the `cicoverage` profile. A
ChaCha8 stream is derived from the seed and complete command arguments; the original
trie variants and unbiased range sampling stay intact. Ordinary calls retain
`crypto/rand`. All nine real CLI variants reproduce their complete outputs with
the same input, change with a different seed, and reject invalid replay inputs.
The comparison still rejects every changed LCOV hit or attribution field; these
corrections await a new complete hosted ordinary/upgrade comparison.
All fuzz/invariant counts, filters and test assertions remain unchanged. Changed
revisions and features receive different samples. This controls the input of the
benchmark; it does not prove arbitrary unseeded runs have identical hit counts.

The test-only mainnet L1 archive RPC is exposed solely to preflight and runtime
tasks through the existing restricted pilot vault. Preflight executes the
production daily 00:00 UTC block discovery, verifies chain 1 and its complete
block identity, and seals its originals. Circle's existing fork-cache key uses
that same block number. Both consumers revalidate the block and retain the sealed
preflight before testing; comparison rejects different blocks or hashes. RPC
authentication is redacted before streaming or retaining logs, attribution,
generated counterexamples and fixture reports.

Four isolated producers build source and Go FFI, then compile complete test
signatures with `forge build` under the coverage profile. `forge test --list`
alone creates minimal artifacts without the needed compiler metadata; the real
build is required for authoritative signature bindings. Both ordinary and
upgrade discovery are retained. Consumers reject stale source/settings/tools,
missing or corrupt compiled files, uninitialized submodules and unexpected
tracked fixture mutations before running tests. The selected NUT bundle writer
retains its exact before/after snapshot payloads as in the standard shadow.

Producers and verdicts start at 16 CPU / 64 GiB, matching the existing coverage
profile's documented memory needs and 16-thread cap. These are supported
[RWX runner specifications](https://www.rwx.com/docs/runner).
Go module/build/runtime caches remain isolated. Runtime Foundry RPC data is
reusable; verdicts, logs, test-failure caches and generated fixtures are excluded
from reusable outputs. Foundry's coverage instrumented compilation executes
inside the fresh verdict; this stage does not claim a compile-only coverage API.
Protected `develop` warming targets only the four producers and executes zero
verdicts or RPC tasks. A CLI warm-only rehearsal must be labeled separately from
an actual protected event.

Local pinned Linux fixtures execute the production Go FFI build recipe and both
coverage Just recipes, verify upgrade environment/selection, setup skips and an
abstract declaration, prove fresh execution, compare full original LCOV and
attribution, and preserve an intentional initial failure plus diagnostics.
Missing RPC inputs, stale preparation and a corrupt FFI binary fail before
testing. Comparison fixtures reject missing passes/new files, duplicate
assignments, reduced settings, altered commands/archives, changed LCOV hits,
invented derived skip reasons, source mutation, corruption and unexplained retries.
Hosted parity, full skip-reason investigation and cache observations remain
required before adding these four occurrences to the inventory.
