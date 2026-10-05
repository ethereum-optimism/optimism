# Acceptance RWX shadow

The next port stays in draft PR #23151. `.rwx/acceptance.yml` introduces optional
`optimism-acceptance-shadow` using shared `run-main` routing. Circle retains all
required gates. The five new job/dependency edges advance
implementation coverage to 21/86 (24%); main-workflow coverage is 13/32 (41%).

## Workload and dependencies

Both complete `./op-acceptance-tests/tests/...` variants run at the `fusaka` L1 fork:
`op-node + op-reth` and `kona-node + op-reth`. Each has eight test-name shards,
`-count=1`, `-p=8`, `-parallel=1`, a 30-minute package timeout and no test retries,
matching the current Circle acceptance command. Shared Just execution preserves
client-specific skips and all subtests. Assignment groups identical test names
across packages, so a global run regexp cannot duplicate their execution.

The Go/Cannon/superchain, ci-profile contracts, Kona host/client/node plus op-zk-proposer, op-reth
and reproducible prestates reuse the full Go port's verified producer packages.
A new isolated SP1 producer builds `kona-sp1-super-range-executor` with Circle's
release profile and all features, staging the binary at the same dedicated path.
Cargo target and sccache caches are retained; source SHA, pins, settings and file
hashes bind every restored dependency. There are no publisher or notification
side effects.

Runtime exports all six Rust binary paths and checks each executable before tests.
`KONA_SP1_ELF_DIR` stays unset, preserving Circle's stub artifacts and mock verifier.
Real SP1 guest ELFs remain a separate Circle gate. Runtime Go builds use an
isolated native compiler cache. Go, Forge, Cast, Anvil and Docker are available;
The pinned Glamsterdam geth tool is installed, checked before execution and retained
with its file hash and version. `eatmydata` preserves Circle's fsync behavior. Archive RPC credentials are not
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
- [x] Hosted SP1 producer and all sixteen acceptance verdicts reach terminal states.
- [x] Compare both variants' complete original reports on the same SHA with Circle;
      investigate every missing, extra, changed or unexplained skipped identity.
- [x] Retain hosted failures and attribute the observed setup failures using originals and source evidence.
- [x] Comparison revision's required Circle gates and optional RWX checks reach terminal states.

Native/protected warming after merge and broader gate/fork/merge-group rehearsals
remain separate operational follow-ups. Protected warming includes the pinned
geth tool layer and executes no acceptance verdicts.

The first clean hosted attempt at `fb8f75b685` retained complete failure reports
and exposed two port-specific runtime gaps: Kona host/client lookup fell back to
unavailable Cargo, and the selective tool layer omitted pinned Glamsterdam geth.
All sixteen original report bundles are retained; their coverage summaries show
zero missing, extra or duplicated assigned identities. See the compact
[first-failure index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-acceptance-evidence/first-failure.json).
Explicit verified binary paths and geth installation address those gaps; no test
expectations or skip rules were changed. The corrected hosted run and same-revision comparison passed before advancing
the inventory.

The corrected clean native run at `236df44cad` passed all six producer packages,
both discoveries and all sixteen fresh verdict workers:
[RWX run 92831490](https://cloud.rwx.com/optimism/runs/9283149078084d80a1cbaba1eb917b69).
Same-revision [Circle job 5632121](https://circleci.com/gh/ethereum-optimism/optimism/5632121)
(Kona) and [5632123](https://circleci.com/gh/ethereum-optimism/optimism/5632123)
(op-node) passed too. Complete original-report comparison verifies the same 80
packages, 243 top-level identities and 780 reported case identities per variant.
All initial assignments occur exactly once. Both providers report 770 pass / 10
skip for op-node and 742 pass / 38 skip for Kona, with zero failures or retries.
There are no missing, extra or unknown skipped cases, unhealthy sources or
incomplete evidence. All sixteen native verdict tasks executed; none reused a
verdict result.

Strict skip-text comparison remains `different`: nine op-node and 37 Kona
messages contain different structured logger timestamps. Each was checked after
replacing only that recognized timestamp field: severity, message, scope and test
identity match exactly. The unchanged originals, both reasons and each resolution
are retained in [the compact parity index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-acceptance-evidence/parity.json).
Other skips preserve existing client-support limits, the disabled batcher test
and full SP1 ELF opt-in behavior; real guest ELFs remain outside this stage.
All four required Circle gates and the five optional RWX shadows passed on the
comparison revision. Full JSON, JUnit, selection, provenance and per-test logs
remain in provider artifacts and downloaded evidence archives.

The final closeout synchronizes `develop`, including its newly added ZK
acceptance case. Exhaustive discovery includes that case automatically. Final
combined-revision execution, original-report comparison and required checks are
tracked in PR #23151; the immutable comparison above remains tied to its stated
source revision. No performance matrix or speed claim is part of this stage.

The final develop synchronization follows current Circle dependencies: the
proposer package/binary is `op-zk-proposer`, and the removed standalone
`op-reth-sdm-fixture` is no longer built, exported or restored. Producer settings,
artifact validation, the op-reth shadow and runtime fixtures follow those changes.
The immutable earlier comparison above predates that upstream rename/removal;
final same-revision evidence is recorded in the PR.

The current-source validation exposed Cargo reusing older workspace dependency
metadata from restored targets when checkout timestamps did not advance. Circle
compiled op-reth successfully on the same SHA. The live two-crate Cargo fixture
reproduces the missing-trait failure, then passes after the timestamp correction
and proves unchanged targets remain reusable. All Rust producers now fingerprint
source content, restore a stable timestamp for each content version and commit
that fingerprint only after successful compilation. Registry targets and sccache
remain intact. See the [failure attribution](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-acceptance-evidence/cargo-freshness-failure.json).
The expanded pinned Linux helper suite passes all 153 tests.
