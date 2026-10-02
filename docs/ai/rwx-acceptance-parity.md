# Acceptance RWX shadow

The next port stays in draft PR #23151. `.rwx/acceptance.yml` introduces optional
`optimism-acceptance-shadow` using shared `run-main` routing. Circle retains all
required gates. Inventory coverage remains 16/86 until the new workloads execute
and their evidence is checked.

## Workload and dependencies

Both complete `./op-acceptance-tests/tests/...` variants run at the `fusaka` L1 fork:
`op-node + op-reth` and `kona-node + op-reth`. Each has eight test-name shards,
`-count=1`, `-p=8`, `-parallel=1`, a 30-minute package timeout and no test retries,
matching the current Circle acceptance command. Shared Just execution preserves
client-specific skips and all subtests. Assignment groups identical test names
across packages, so a global run regexp cannot duplicate their execution.

The Go/Cannon/superchain, ci-profile contracts, four Kona binaries, op-reth/SDM
and reproducible prestates reuse the full Go port's verified producer packages.
A new isolated SP1 producer builds `kona-sp1-super-range-executor` with Circle's
release profile and all features, staging the binary at the same dedicated path.
Cargo target and sccache caches are retained; source SHA, pins, settings and file
hashes bind every restored dependency. There are no publisher or notification
side effects.

Runtime exports all five Rust binary paths and checks each executable before tests.
`KONA_SP1_ELF_DIR` stays unset, preserving Circle's stub artifacts and mock verifier.
Real SP1 guest ELFs remain a separate Circle gate. Runtime Go builds use an
isolated native compiler cache. Go, Forge, Cast, Anvil and Docker are available;
`eatmydata` preserves Circle's fsync behavior. Archive RPC credentials are not
needed by these suites and are not added to the acceptance workflow.

Initial workers are 16 CPU / 64 GiB for op-node and 16 CPU / 32 GiB for Kona.
Circle documented peaks of 38.5 GiB and 21.3 GiB respectively; RWX's supported
sizes and reserved memory make 64 GiB the safe initial op-node allocation.
This is a reliability baseline, not a performance comparison. No benchmark
matrix or concurrency optimization is part of this stage.

## Discovery and reports

`acceptance-manifest.py` retains untagged `go list -e -json` and original
`go test -list '^Test' -json` output. Package/dependency errors, failed/incomplete
listing, duplicate identities and invalid assignments fail validation. Complete
package selection includes packages without tests. Every discovered test identity
receives exactly one initial assignment, and unknown/new names remain included.

The shared Just runner retains complete discovery, effective settings and actual
assigned identities on both providers. Circle continues using its timing-based
splitter; RWX consumes a source/settings/toolchain-bound manifest and deterministic
partitions. Valid empty shards still emit fresh package reports. Native verdict
caching is disabled, while compiler objects remain reusable.

RWX collects original events, JUnit, per-test logs, dependency provenance,
complete selection, execution coverage and a compact native reporting projection.
Collection runs after test failures and rejects missing, duplicate or unassigned
top-level verdicts without replacing original failure evidence. Protected develop
warming targets producers and discovery only, running no acceptance verdicts.

## Validation tracking

- [x] Discovery/assignment tests: missing data, errors, duplicate test names,
      unknown/new cases, empty shards, stale SHA/settings/toolchain and corruption.
- [x] Execute the Just entrypoint through Circle and RWX fixture paths, including
      failed discovery and a failing verdict with retained original reports.
- [x] Original failure retention and execution-coverage validation tests.
- [x] ShellCheck and RWX definition/package lint.
- [ ] Hosted SP1 producer and all sixteen acceptance verdicts reach terminal states.
- [ ] Compare both variants' complete original reports on the same SHA with Circle;
      investigate every missing, extra, changed or unexplained skipped identity.
- [ ] Retain hosted failures and attribute test flakes using source/history evidence.
- [ ] Final required Circle gates and optional RWX checks reach terminal states.

Native/protected warming after merge and broader gate/fork/merge-group rehearsals
remain separate operational follow-ups. Mark the five inventory jobs only when
hosted execution supports their completion.
