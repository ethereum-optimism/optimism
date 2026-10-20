# RWX CI operator guide

Maintain this guide, the [86-job checklist](rwx-parity-todos.md), and
[evidence retrieval instructions](rwx-evidence-index.md). Historical closeouts,
benchmark samples and earlier inventories are archived outside Git. Implementation
stays in [PR #23151](https://github.com/ethereum-optimism/optimism/pull/23151)
on `codex/rwx-ci-pilot`, targeting `develop`.

## Authority and rollout

CircleCI owns `ci-gate`, `required-contracts-ci`, `required-rust-ci` and
`required-rust-e2e`. Keep dependency review and existing rulesets. Optional RWX
checks supply shadow evidence; they do not authorize required-gate migration.
Production publication, release/tag jobs and scheduled maintenance remain with
Circle. The private selector registry exercises publication without writing to
production. Only the pilot branch and `develop` trigger the native coordinator.
Fork authorization remains Bailiff's human-approved `external-fork/*` path;
fork, merge-queue and denied-cache-access rehearsals are still cutover work.

Shared selection is [routing.yml](../../ops/ci/runtime/routing.yml), interpreted by
[compute-workflow-conditions.sh](../../ops/ci/runtime/compute-workflow-conditions.sh).
Circle scripts adapt Circle metadata; RWX metadata adapts native events. Unknown
or mixed code paths cannot take the docs-only shortcut. Safe skips still emit
Circle's exact required contexts. CI source changes select contracts and Rust
validation too. See [CI review](ci-config-review.md) for the merged continuation
config and gate review procedure.

## Reusing feature compilation on PRs

Each `features-build-N` task runs the original two Cargo Hack phases with the
source SHA, pinned tools, flags, incremental mode and one of ten partitions bound
to its inputs. Its compiler filesystem can be reused without granting PRs writes
to the protected tool-cache vault. `features-N` then inherits the current source
and that compiler state, reruns both original phases with `cache: false`, and
publishes only its fresh report. The Rust gate still depends on all ten fresh
consumers. Both stages use 4 CPUs and 8 GB; preparation and filesystem transfer
must be included in benchmark wall time and cost. A cold producer is additional
work, so warm results must be reported separately from warm-up.

RWX WASM checks also link native Cargo build scripts. Restored GCC/glibc inputs
have intermittently returned `ETIMEDOUT` from file opens (`crti.o`,
`crtbeginS.o`, `libgcc.a`). Before these checks compile, `rwx-materialize-linker.py`
copies the image's host linker inputs into fresh local inodes, preserving their
bytes, ownership, mode and timestamps. It retries only timed-out input reads,
at most three times, and fails on exhaustion or any other error. The sealed
`host-linker` stage retains hashes, inode changes, timing and timeout counts.
Compilation and verdicts are never retried by this helper. This is a local
mitigation; the provider-side cause of the original reads remains unconfirmed.

Develop refresh on October 19, 2026 adds required `rust-deny` and
`opgeth-ratchet` checks, bringing the current inventory to 88 PR occurrences.
The historical 86-occurrence parity snapshot remains dated evidence. SP1 now
also includes the log-commitment ELF and standalone vkey crate discovered from
its original recipes. Current-source validation must include those additions.

## Helper ownership and removable boundaries

| Directory | Owner and lifetime |
| --- | --- |
| `ops/ci/runtime/` | Permanent runners, report/source libraries, routing, native gate policy, timing seeds and build inputs |
| `ops/ci/tests/` | Permanent execution/configuration tests, routing scenarios and shared fixture construction |
| `ops/ci/migration/` | Offline comparers, Circle gate mappings/alignment, Insights, schedule/RPC synchronization, private publication rehearsal and their tests/fixtures |

Runtime imports only runtime. Permanent tests consume runtime and permanent test
support. Migration tooling may consume both. Moved commands retain their basenames,
arguments and artifact destinations; there are no old-path forwarding helpers.
The existing Circle apt installer is the sole installer adapter; RWX uses
`runtime/apt-install.sh` directly. Toolchain snapshots exclude migration inputs.

The native gate manifest is version 4. It contains native assignments, routing,
observers and check names. `migration/circle-gates.json` owns the Circle workflow
mapping; `migration/circle-alignment.py` verifies expanded Circle prerequisites.
Native execution does not read Circle YAML. Receipts use `native_policy_sha256`,
a hash of native policy, workload definitions/packages and the gate implementation.
The gate also verifies checked-out SHA, clean tracked source, current run/attempt,
complete assignments, engine states and sealed artifacts. Workload producers and
consumers own source/toolchain/settings binding and binary integrity.

Shared runners still accept `--provider circleci`, `record-circle`, and
`CI_CHECK_PROVIDER`, `CI_CONTRACT_PROVIDER`, `CI_RUST_PROVIDER` and
`CI_E2E_PROVIDER=circleci`. Retained metadata aliases are `CIRCLE_SHA1`,
`CIRCLE_BRANCH`, `CIRCLE_NODE_INDEX`, `CIRCLE_NODE_TOTAL` and `CIRCLE_WORKFLOW_ID`.
Acceptance/Rust dependencies retain `.circleci-cache/rust-binaries`; Circle
selection paths retain their CLI partitioning adapter where applicable. These are caller compatibility inputs, not native
gate dependencies. Their tests stay with the execution contract until those
callers retire. Insights and the schedule/RPC synchronization remain transitional
workloads in both providers; the production publisher is unchanged.

Offline `migration/report-evidence.py` centralizes original byte verification,
explicit empty-file recovery with audit records, and common path normalization.
Suite comparers still own discovery, required reports, settings, retries and
verdict interpretation. Runtime verification is read-only and rejects missing
empty files. Comparison helpers must not repair runtime artifacts.

## Chosen execution graph

[pr-gates.yml](../../.rwx/pr-gates.yml) coordinates full workloads and exact
engine-bound receipts. Workload definitions retain CLI entrypoints and protected
`develop` cache warming. The definitions and their local package parameters own
current resources, eligibility and settings; change them with their regression
fixtures. The 24-shard Go experiment and standalone package-transfer probe have
been retired.

| Workload | Definition / contract |
| --- | --- |
| Full Go | `go-tests.yml`: 12 duration-balanced package shards; compile 8 CPU / 16 GiB, verdict 8 CPU / 16 GiB; `-p=4`, `-parallel=8`, `40m` package timeout, three gotestsum retries / 50-failure ceiling |
| Acceptance | `acceptance.yml`: eight shards each for op-node and Kona, Fusaka / op-reth; separate discovery/compilation and fresh verdicts, per-variant memory |
| Contracts | `contracts.yml`: standard and modified-file matrices for four features; PR standard `liteci`, protected standard `ci`, modified `ciheavy` |
| Coverage / upgrades / L2 fork | Separate definitions and compile/verdict packages; preserve `cicoverage`, chain/feature matrices, pinned fork inputs and reporting |
| Rust | `rust.yml`, `op-reth.yml`, `rust-e2e.yml`; preserve package/features/partitions and original command authorities |
| Other PR jobs | `pilot.yml`, `pr-checks.yml`, `cannon-go.yml`, `nut-prefork.yml`, `nut-provenance.yml`, `sp1-guest.yml`, `kontrol-build.yml`, `fetcher-artifacts.yml`, `flaky-report.yml`, `selector-upload.yml` |
| Rollup compatibility | `go-rollup.yml` remains CLI-only; shared compiled-Go helpers retain `--suite go-rollup` alongside `go-tests`; it adds no coverage credit |

`just list-test-packages` owns full Go selection, including packages without tests.
Its acceptance, Cannon, Rust and deployer Forge exclusions remain. Discovery
rejects errors with the `ci` tag. Every selected package has one initial shard.
Timing seeds affect placement only; unknown packages stay included. Captain has
no Go partitioning support here; the shared package balancer assigns packages.

Compile packages and verdict packages are separate. Go compilation starts after
Go preparation and contracts, while Rust/prestates continue. A verdict consumes
its own compiler leaf and runtime dependencies; it does not wait for other shard
compilers. Use `package.use` for file inputs, and caller leaves such as
`compile-0.build`. Artifact mount and vault expressions are not package value
arguments. Output filesystem filters use static family roots: native RWX omitted
parameterized output paths despite successful lint.

## Cache, artifacts and credentials

Keep native Go module/build/runtime, Foundry, Cargo target, sccache and
Docker/BuildKit caches isolated by producer, profile and toolchain. A content miss
restores compiler state and recompiles changed inputs. `cache-epoch`, build-probe
and the empty-target sccache mode remain explicit cache diagnostics; normal runs
use their chosen defaults. Retain probe evidence outside Git rather than adding
alternative task graphs. Compiler warming targets producers and executes zero
tests, RPC verdicts or helper fixtures.

Rust target restoration records a content hash and timestamp for each source
file. Unchanged files recover their prior timestamps; changed/new files get new
timestamps. Legacy manifests and uncommitted prior builds refresh all inputs
once. The caller commits the map only after a successful build. Cargo owns
dependency freshness, including embedded bundles and compiler/toolchain changes.

The RWX Rust feature checks and test build/runtime tasks trial incremental host
compilation with `cargo-incremental: "1"` (CLI override: `--init
cargo-incremental=0`). The shared runner enables the `dev`, `test`, and
`fast-build` profiles; it unsets the global incremental switch because sccache
0.18.0 rejects `CARGO_INCREMENTAL=1`. Local incremental invocations pass through
sccache; registry dependencies remain eligible for compiler caching. Reports and
test archives bind the effective profile settings. Test build/runtime cache names
include the incremental mode: combining old nonincremental targets with the new
outputs exceeded RWX's 100 GiB filesystem-layer limit in the first hosted trial.
Only test build/runtime workers use 150 GB writable disks. A compiler-cold run
exhausted the default 50 GB disk; a later warm verdict required 150 GB before
execution because its dependencies occupied 88.37 GB and RWX requires 20 GB of
scratch space. This does not increase the separate 100 GiB filesystem-layer
limit. CPU/memory resources stay unchanged. Feature cache names, shards and
selected commands remain. Circle, release/prestate builds
and other Rust jobs retain their current incremental policy. Compare complete
compile/cache-transfer timings before claiming a speed improvement.

Verdicts execute freshly. Most use `cache: false`; Rust and contract fork verdicts
include run and attempt identity in their cache keys so tool-cache reads remain
available. RWX's `cache: false` disables tool caches as well as result reuse.
Exclude test results, logs and failure caches from
reusable outputs. Go/acceptance runtime snapshots exclude build caches and Cargo
targets, then validate and restore sealed dependency archives. Preserve fixtures,
working directories, source paths, submodules and branch/commit metadata. Go,
Forge, Cast, Anvil and Docker remain available when runtime fixtures require them.

Each producer binds source revision, toolchain, effective settings and file hashes.
Consumers reject missing/corrupt binaries or stale bindings. The latest Cannon
build owns its Docker image identity; rebuilding can replace the initial image.
Gate receipts bind actual selected caller states, attempts and source. Failed,
canceled or missing selected verdicts fail; successful diagnostic retries do not
erase original failures.

The shared cache-only vault permits writes only from `refs/heads/develop`.
PRs, including the pilot branch, restore this baseline read-only and compile their
changes locally. The pilot cache grant was revoked on 2026-10-07. Verify the
repository grants with `rwx vaults access list --vault optimism-op-reth-shadow
--json`; the only repository grant must be `ethereum-optimism/optimism` on
`refs/heads/develop`. New namespaces start cold until a protected `develop` run
warms them. The separate test-only archive-RPC vault
supplies Go runtime and contract archive inputs. Availability checks precede
expensive builds. Values stay in preflight/runtime tasks, outside tool setup,
compilation, package arguments and retained evidence. No production publishing
credentials are introduced into RWX.

Contract archive calls use the shared HTTP relay for preflight and Forge runtime.
It allows one upstream request at a time, at most 20 requests/second per verdict,
and five attempts for HTTP 429/500/502/503/504 with 2/4/8/16-second backoff.
JSON-RPC errors returned with HTTP 200, permanent HTTP errors and exhausted
transport failures still reach Forge unchanged. This is a per-verdict limit,
not a provider-wide quota. Each attempt retains redacted request/response bytes
and status; the outer report seals the relay's final manifest too. Nested Bash
must not reload `BASH_ENV` and replace the loopback RPC environment with the
original upstream. Diagnostic suite retries never replace an original failure.

For a resource-matched benchmark, freeze one commit and warm both providers
before measured pairs. Use the committed RWX definitions and protected cache
vault without private writable namespaces or permission changes. Set Circle's
`c-go_fresh_tests` and `c-contract_coverage_replay` to true to match RWX's fresh
Go verdicts and deterministic coverage settings. Record every failed sample.
Compare actual provider allocations, including producers and shards: the pilot's
OP Node acceptance and coverage jobs use Circle machines at 16 CPU / 64 GB to
match RWX without reducing memory below the observed acceptance peak. Circle's
Docker runner sizes cannot express that allocation. Other branches retain their
existing Docker executor for those jobs. Equal allocations do not imply equal
available RAM: RWX reserves 2 GB for its own processes. Report executor differences
and unmatched bootstrap/control tasks separately, but include them in elapsed
time and cost. Keep preparation, execution and output processing visible; cache
phase worker-seconds overlap and must not be added to critical-path time.
Cold preparation killed SP1 at 4 CPU / 8 GB on RWX; its matched pilot build and
check allocations use 8 CPU / 16 GB on both providers. Non-pilot Circle jobs keep
their existing allocation. Do not infer adequate cold-build headroom from a
successful cached build.
The pilot NUT verdict likewise matches 2 CPU / 8 GB: its cold Go build was
killed at 2 CPU / 4 GB on RWX. Circle uses a machine for this pilot job while
other branches retain the existing Docker executor.
Clippy and udeps use 4 CPU / 8 GB on both pilot providers after both cold
checks were killed at 2 CPU / 4 GB. Circle retains its Rust Docker image and
normal compiler-cache setup; non-pilot jobs keep their existing allocation.

The general Go contract producer matches Circle's `ci` profile and
`--deny-warnings --skip test` arguments, sealed in dependency settings. Forge's
`test` alias skips `.t.sol` files, retaining helpers such as `MockSP1Verifier.sol`
under `test/`. Rust E2E retains its original `--skip test` arguments. Compiling
the extra executable Solidity test graph exhausted 4 CPU / 8 GB and one
8 CPU / 16 GB cold producer; those failed preparations are separate evidence.
The corrected general producer uses the normal 4 CPU / 8 GB allocation and a
production `source-scripts` tool-cache namespace so old full-build artifacts
cannot enter its dependency archive. This namespace has the same protected
`develop` write grant; it is not a private writable benchmark cache.

Runtime scripts import `ci-report.py` for JSON, hashes, manifest byte verification
and one subprocess invocation, and `ci-test-results.py` for original Go/JUnit
parsing. Suite owners choose discovery, settings, required reports, retries and
verdicts. The shared process helper forwards cancellation and preserves exit
status; it never retries. Contract callers redact before storing or hashing.
Only offline collectors may explicitly recover a manifest-declared empty file,
with an audit record. Runtime recovery is forbidden. Every runtime import belongs
in the producer's source/input seal and filtered snapshot. Test imports from the
actual filtered files, including SP1's toolchain snapshot.

Rust test target-cache names include a namespace of tracked Cargo manifests,
`Cargo.lock`, Cargo configuration, the pinned Superchain bundle checksum,
compiler flags, build/cache helper settings,
and actual Rust/Cargo/sccache versions. Source-only edits keep the namespace and
use per-file timestamp restoration. Dependency/toolchain changes seed a new
initial layer. This fixes the observed `develop` dependency update: compilation
passed, but retaining the old 64.9 GiB layers plus 54.2 GiB of new outputs exceeded
RWX's 100 GiB filesystem-layer cap. Runner disk size cannot raise that cap. The
failed producer's original archive and logs remain in external evidence.

Timestamp preparation also tracks the ignored Superchain tar declared by Cargo's
build script. Its producer still owns checksum verification. A cold build may
materialize it after preparation; successful compilation records its actual
timestamp. Later restores preserve that timestamp when its bytes are unchanged.

Raw target restores can quantize mtimes to whole seconds. Preparation rounds
unchanged input timestamps down to the same resolution and stamps changed inputs
with the next whole second. It waits until the wall clock is strictly past every
prepared input timestamp before returning to Cargo. Unchanged inputs normally
return immediately; the clock check also handles an early wake or a backwards
clock adjustment. Newly materialized generated assets keep their producer-owned
timestamps until their next restore.

CLI validation of this helper on `fd722f0900` with uploaded CI changes passed the
[full Rust test workload](https://cloud.rwx.com/optimism/runs/01a13ae1ed257f0a9061db6cfe987fd5):
3,314 passes, 17 existing doctest skips and 10 policy-excluded nextest cases.
The [following cache reuse check](https://cloud.rwx.com/optimism/runs/01a13b0aacc373e795a9328169eeacc6)
restored 1,346 source files and the generated tar with no refreshed inputs and
zero crate recompilations. A [whole-second target regression](https://cloud.rwx.com/optimism/runs/01a13ae6f4987ee69ca2c04698ee113b)
also verified source-change invalidation and reuse after the changed build.
These trials used isolated default-vault cache names; they do not establish
protected-branch cache warming or change the production cache grants.

Full-test compiler producers retain individual `rust/target` files. Keep compiler
binaries and fingerprints;
remove nextest reports before publication. RWX can reuse unchanged filesystem
files without retaining another full compressed snapshot on every update.
The complete producer artifact at `4047caccc5` measured 54.24 GiB of regular-file
payload (59.26 GiB counting hardlink aliases), versus 11.44 GiB compressed.
This is a current-tree measurement, not the complete RWX layer total.
Successful workspace runs retain `cache-size.json` with regular-file bytes,
allocated bytes, hardlink counts and directory breakdowns for targets, Cargo
and sccache. These sizes describe the current tree; the engine's layer accounting
also includes inherited and superseded layers.

The Rust test handoff uses RWX's immutable filesystem directly. The current-source
`tests-build` producer publishes compiler files plus a small `compiler.json`
artifact. The compiler-dependent worker inherits those files and checks the
manifest's revision, toolchain, flags and source-fingerprint digest before Cargo
runs. A pending fingerprint is rejected. Neither task packs a target archive;
compiler and unit verdict workers publish reports only, avoiding another full
compiler-layer publication after tests. The nextest binary archive remains the
handoff to the execution-only worker.

Compiler preparation uses a source ancestor selected by the last change to build
inputs: manifests, lockfile, toolchain, Cargo/Just configuration and CI helpers.
Its checkout, dependency fetch and compiler outputs are immutable cached tasks, reusable across
source-only PR edits without shared-cache writer permission. Current-source
workers reset/clean the Rust checkout to HEAD and always invoke Cargo; source fingerprint preparation
restores unchanged timestamps and refreshes edited inputs. Feature partitions use
the Cargo.lock digest on both services, keeping assignments stable across source
edits. Both providers default to `CARGO_INCREMENTAL=0` so sccache can cache local
crate compilations. The explicit incremental trial remains available on RWX.

The test compiler seed runs a second compilation discovery after its cold build,
so generated watched inputs (the superchain archive) precede Cargo's final
build-script fingerprints. Fresh HEAD workers still create their own test archive.

An intermediate ancestor need not build. Optional preparation retains its original
failure and discards partial targets before fresh HEAD compilation. Such a fallback
is not evidence of a warm cache. Changes to `cache-epoch` invalidate the preparation
tasks. Benchmark evidence must verify the actual restored targets, current-source
recompilation, cache hits and fresh coverage rather than assuming warmth from a
successful previous run.

The full Rust test workload has two fresh verdict workers. `tests-unit` executes
nextest from the checksum-verified binary archive. It receives no compiler archive;
its unpacked execution binaries and runtime fixture outputs are excluded from
filesystem publication. `tests-compiler` runs the beacon Cargo test
and doctests using the current producer's inherited compiler filesystem.
Both workers retain the selected 16 CPU / 32 GiB resources and may execute in
parallel. The `tests` task collects both artifacts after execution, including
ordinary failures. It verifies source, tools, settings and run identity, retains
each child's original settings and attempt, consumes its validated coverage, and
emits the existing
`.ci/rust-workspace/tests` full-suite report. Native gates observe both workers and
the aggregate, including cancellation or absent artifacts. Warm-only runs select
neither verdict worker nor the aggregate.

For an isolated execution fixture, run
`RWX_LIVE_RUST_FIXTURE=1 python3 -m unittest discover -s ops/ci/tests -p 'test_rust_workspace.py'`
on Linux with the pinned Rust, nextest, Just, mold and sccache tools. This exercises
an empty-target nextest worker, native compiler handoff validation, aggregate coverage and
original failure retention. It also copies the real Lokahi CLI tests and verifies
the runtime `CARGO_BIN_EXE_lokahi` path under nextest and ordinary Cargo. Rust 1.95
and nextest 0.9.132 provide this input; compile-time executable paths are not
portable to an extracted target directory. The shared `tests` CLI mode remains available for
Circle and earlier collected commands. The split changes scheduling and transport;
it does not remove the independently scheduled `doctest` workload. Compare hosted
end-to-end timings before claiming a speed improvement. The compiler-dependent
worker inherits compiler state and publishes only reports; hosted cache transport
and filesystem restoration performance remain part of the benchmark.

Raw target layers exceeded the cap even after executable pruning: at
`d233b6c244`, 73.5 GiB inherited plus 26.6 GiB added failed publication after
successful compilation. Keep those original failure records outside Git.
Cache preparation
separates host `deps/*.rmeta` hardlinks from incremental metadata while preserving
bytes, permissions and timestamps; retain `cache-publication.json`. The engine
result, rather than this preparation record, proves filesystem publication. A restored
shared entry returned `ESTALE` twice, but the filesystem root cause is unproven.
These changes need cold publication and untouched warm-update validation; a
breakpoint run that touched cached files is diagnostic evidence only.

Modified-contract runs capture `develop` once in `modified-baseline` and retain
`.ci/contract-baseline.txt`. Compile/verdict packages receive that full SHA through
`target-sha` / `CI_CONTRACT_TARGET_SHA`; originals record `target_binding:
run-pinned`, `target_sha`, and `merge_base_sha`. Discovery keeps Circle's three-dot
changed-file rule but names the pinned commit. Verdicts verify the same pin and
selection without fetching the moving branch. Standard-contract cache inputs do
not inherit this baseline. Circle's standalone commands retain their existing
fetch/selection behavior; comparison requires both providers to use the same
recorded target and merge base.

The bounded changed-crate experiment on `3df2ffe919` used three samples per mode
for `kona-providers-alloy`, 4 CPU/8 GiB checks and 16 CPU/32 GiB test compilation.
Median compile seconds were 1.79/0.60 (off/on) for checks and 2.42/0.81 for tests.
Including local tar archive/restore, medians were 3.42/3.03 and 3.14/1.75 seconds.
Target contents grew from 996 MB to 1,173 MB for checks and from 1,053 MB to
1,417 MB for tests. Keep the selected incremental configuration. This experiment
measures a small crate edit, not full-workspace runtime or cost; local tar timings
do not measure RWX cache-network transfer. Cold seeds are excluded from samples.

The historical pilot-readiness rehearsal allowed the pilot branch and `develop`
in both vaults. Cache writes are now restricted to `develop`; the test-only RPC
vault still permits the pilot branch. A patched CLI run with `branch: develop` was denied
the RPC vault before execution; a harmless cache-write probe ran but declared no
protected tool-cache version. A separate running task was canceled and returned
`aborted/cancelled`, `failed`. Routing and gate fixtures cover safe skips,
unknown/mixed paths, queue metadata and failed/canceled/missing prerequisites.
Actual fork and merge-queue events remain required before gate cutover. GitHub
currently requires one approving review plus two contract-team approvals for this
PR's contract paths; CI success does not satisfy these reviews.

## Run and validate

Use repository tool pins via mise. RWX CLI v3.32.1 was used for retained pilot
validation; install/authenticate it separately and retain the actual version in
new evidence. Do not copy authentication wrappers or secret values into reports.
An authenticated operator can run a workload from a clean pushed checkout:

```bash
rwx whoami
rwx run .rwx/pr-gates.yml --wait
# A CLI workload run does not establish automatic push/check association.
rwx results RUN_ID --json
rwx logs TASK_ID
```

Validate changes as one batch, then observe one combined hosted push. Do not run
Circle per edit or restart performance tuning. Additional runs follow diagnosed
failures or material fixes. On macOS use Bash 4+ for routing and a Python with
PyYAML; the pinned Python environment may require its dependencies installed.

```bash
mise exec yq jq -- bash ops/ci/tests/test-decision-tree.sh
mise exec yq jq -- bash .circleci/scripts/test-decision-tree.sh
python3 -m unittest discover -s ops/ci/tests -p 'test_*.py'
python3 -m unittest discover -s ops/ci/migration/tests -p 'test_*.py'
python3 ops/ci/migration/circle-alignment.py
rwx lint .rwx/*.yml .rwx/packages/*.yml --warnings-as-errors
# Merge and validate both setup and activated continuation configs as described
# in ci-config-review.md, with the authenticated Circle CLI.
```

Run ShellCheck on changed scripts using the native task's Bash context. Live
Go/Forge/Docker fixtures are opt-in locally and execute in their hosted helper
tasks; report local skips honestly. Preserve failure originals before retrying.
Watch the exact pushed SHA's four Circle gates, dependency review and all optional
checks to terminal states. Git conflict status and local tests are not hosted
readiness.

## Retained changes outside CI

This audit covers every non-CI source/config change against the pilot's upstream
base, plus the Circle adapters below. Archived pre-cleanup diffs retain exact
bytes. The table distinguishes required reporting from workload changes.

| Retained path(s) | Necessity and behavior | Regression rationale / evidence |
| --- | --- | --- |
| `.dockerignore` | Exclude `.ci` caches/reports and generated offline witnesses from image contexts; these are not image inputs | Prestate hashes and Cannon guest/image validation catch changed build inputs; avoid multi-GB context transfers |
| `.github/CODEOWNERS` | Give CI/security owners review of `.rwx` and shared `ops/ci` policy | Both providers now execute these files, so existing CI review ownership must cover them |
| `.gitignore`, `rust/kona/.gitignore` | Ignore generated `.ci` data and offline Cannon output | Keep caches/evidence out of commits; checked-in witness specification remains an input |
| Root `justfile` Go runner | Retain complete package/discovery/settings reports; opt-in fresh Circle tests, default false | Discovery errors, new/no-test packages, duplicates, fresh execution and retry ceilings are covered by `test_go_suite`, `test_go_package_shards`, `test_go_compiled_tests`; no default selection/count reduction |
| Root `justfile` pre-fork recipe | Delegate original fresh Go command to `nut-prefork-test.sh` so originals survive failure | `test_nut_prefork` exercises production Just loops, both forks, fixture paths, cancellation and original failure retention |
| `op-acceptance-tests/justfile` | Provider-neutral shard inputs and retained discovery/assignments/JSON/JUnit; valid empty shards still emit reports; quote checkout paths | `test_acceptance_manifest`, `test_acceptance_runner`, `test_acceptance_report` verify exact selection, errors, empty shards and retries; same-SHA full Fusaka parity retained |
| `op-acceptance-tests/scripts/generate-flaky-tests-report.sh` | Public API may run anonymously; reject HTTP/schema errors, retain each attempt, escape report HTML, preserve acceptance-only interface | `test_flaky_report` / comparer fixtures cover full public response, malformed data, transient/permanent errors and output drift; branch label identifies requesting branch, not an API filter |
| `op-fetcher/justfile` | Separate `compile-contracts` from explicit developer `build-contracts` export; CI must compare untouched committed inputs | Old command overwrote its comparison inputs. `test_fetcher_artifacts` proves committed drift fails, full future artifact discovery, failed exports remain atomic, missing/corrupt data and cancellation |
| `packages/contracts-bedrock/scripts/FetchChainInfo.s.sol` | Pin 0.8.30; restored compiler sets otherwise selected 0.8.28 | Both first hosted builds failed the strict artifact diff; the pin resolves compiler selection without relaxing comparison |
| Four `op-fetcher/pkg/fetcher/fetch/forge-artifacts/FetchChainInfo.s.sol/*.json` | Actual compiler-generated metadata refresh and portable remappings; required by the repaired strict check | Baseline/current ABI, method identifiers, creation/runtime bytecode and link references agree. Source maps, compiler node IDs and documentation metadata changed; they remain checked, not ignored; same-SHA parity covers all 729 compiler outputs and four embedded artifacts |
| `ops/check-changed/main.py` | Treat `.rwx` and `ops/ci` as CI inputs for existing changed-pattern jobs | `test_check_changed` and real routing fixtures prevent CI-only changes silently skipping validation |
| `ops/scripts/nut-provenance-verify-changed.sh` | Shared selection/report adapter; full inspection remains opt-in and does not alter normal changed-lock selection | `test_nut_provenance` covers missing refs, unchanged/changed locks, failed generation and retained attempts; historical worktree remains authority |
| `ops/scripts/nut-provenance-verify/{main.go,report.go,main_test.go}` | Optional reports copy actual source/tool/submodule and generated bytes before temporary worktree removal; failed generation cannot report an old snapshot as fresh | Real Git tests cover matching/mismatched bytes, failed generator, dual report/generator errors and worktree cleanup; plain verifier wrapper keeps reporting disabled |
| `ops/scripts/test-interop-deposits-diff.sh` | Optional stdout/stderr/exit retention; ordinary invocations still clean temporary files | Existing differential command/result remains authority; Rust extra fixtures reject failed or changed originals |
| `packages/contracts-bedrock/scripts/go-ffi/{trie.go,trie_replay.go,trie_replay_test.go,README.md}` | Explicit CI coverage replay only; OS randomness made identical seeded Solidity runs produce different LCOV hits | Replay requires `CI=true`, `cicoverage`, valid 256-bit seed, and binds complete argv; unset seed keeps `crypto/rand`. Reader tests and nine real CLI variants prove reproducibility and changed-seed/variant sampling; full LCOV/attribution comparison stays strict |
| `packages/contracts-bedrock/test/{setup/Setup.sol,L1/ResourceMetering.t.sol,L2/L1Block.t.sol}` | Attach reasons to existing `vm.skip` calls; log after skip was unreachable | Conditions, assertions and skip counts unchanged. Coverage/suite fixtures verify reason evidence and initial failures. No Solidity production contract is changed |
| `packages/contracts-bedrock/test/kontrol/scripts/make-summary-deployment.sh` | Capture deployment, loader and generated inputs only when report directory is set | `test_kontrol_build` validates real summary/proof phases, corrupt/stale inputs and failures; ordinary generation unchanged |
| `rust/lokahi/tests/cli.rs` | Read Cargo/nextest's runtime executable path so archived tests use the remapped Lokahi binary; assertions and CLI behavior stay intact | The pinned Linux fixture copies the real three CLI tests, reproduces their missing compile-time path without producer targets, then verifies archive-only nextest and ordinary `cargo test` execution |
| `rust/justfile` | Optional cargo-hack command-list reporting, default false, retains actual partitions | Workspace fixtures verify full packages and exactly-once feature partitions; original checks still run |
| `rust/kona/bin/client/scripts/fetch-witness-tar.sh` | Remove a stale ETag when the archive is missing; bounded transfer retries | Otherwise a bodyless 304 left no runtime witness. Cannon fixtures cover actual archive/ELF/image handoff and retained transport failures |
| `docs/ai/ci-ops.md`, `docs/ai/ci-config-review.md`, three RWX guides and checksum index | Describe actual shared routing, required gate ownership and private retrieval | Local links and the 86 exact baseline names are checked; historical source and hashes survive consolidation |

Removed: the unrelated `Bytes.t.sol` comment, three standalone native package
probe definitions, three isolated Circle replay workflows and their API routing
options. The selector workflow remains because normal pilot pushes require a
private publication target. Historical probes/experiments are reproducible from
the archived source. Offline comparers, timing seeds, bounded captured regression
fixtures and cache diagnostic modes remain: active parity verification and
package placement consume them. They are not generated run evidence.

## Circle change audit

| Surface | Retained reason and regression boundary |
| --- | --- |
| `.circleci/config.yml` and continuation parameters | Keep default-false `c-go_fresh_tests`, `c-contract_coverage_replay`, `c-nut_provenance_full`, with default-false forwarding. These alter explicit verification inputs, not normal Circle test settings. Retired API replay and block-override parameters are no longer forwarded |
| `.circleci/routing.yml`, `scripts/{collect-params.sh,compute-workflow-conditions.sh,workflow-helpers.sh}` | One policy and path authority for both providers; Circle entrypoints translate event/base/branch and preserve schedules, tags and authorized forks. The retired issue-close automation is no longer routed. Real changed-file and Circle-adapter fixtures cover unknown paths and safe gates |
| `continue/main.yml` workload wrappers | Retain original recipes via shared suite runners and complete discovery/settings/failure reports, including dependency/prestate hashes. The fetcher check now detects committed drift. Gate names and dependency lists remain Circle-owned |
| Main contract matrices | Preserve profile, feature, changed-file selection, timeout and fuzz/invariant settings. Reasons/reporting support strict parity; seeded coverage is opt-in. L2 pilot relay/pinned-block checks retain originals without reloading `BASH_ENV` over runtime overrides |
| Private selector workflow | Pilot pushes only, original publisher against official private service/database; no production context. No API replay dispatch remains. Non-pilot production uploader retains its original branch filtering and destination |
| `continue/rust-ci.yml` | Shared reporters retain original workspace, WASM, registry, Cannon and interop commands; manifests bind complete selection and native artifacts. Original profiles/features and required fan-in remain |
| `continue/rust-e2e.yml` | Original devnet, restart, proof and SP1 guest runners retain complete runtime/compiler originals; profile, feature and suite selection remain. Rust E2E comparer/regressions cover original failures, retries and runtime dependencies |
| `.circleci/scripts/{test-schedule-triggers.js,test-decision-tree.sh}` and shared fixtures | Read the shared routing file, exercise the same Circle entrypoints, and verify retired options cannot dispatch isolated workflows. Setup/merged/activated config validation supplements routing tests |

Same-SHA workload evidence established the 86-job parity baseline; this cleanup
uses existing regression ownership rather than creating new workload benchmarks.
Cutover, required-check ownership, production identity and full `develop` parity
remain open in the checklist.

## Migration tooling retirement

Retire migration tooling only after approved RWX required-gate cutover and the
rollback window closes. First archive final original reports and the matching
source revision in checksum-verified storage outside Git. Replace or retire
Insights reporting, Circle schedule synchronization and the private publication
rehearsal. Then remove `ops/ci/migration`, its mappings/tests, helper-task
invocations and migration-only filters. Finally remove Circle adapters and unused
provider/metadata compatibility modes, and run permanent tests plus native
configuration validation. This PR separates these owners; it does not cut over
required gates or retire these workloads.

Verification is reported by scenarios: complete/new/unknown package discovery;
omission on either provider; duplicate assignment; stale SHA, toolchain or settings;
corrupt artifacts; real runtime fixture builds; retries; cancellation; retained
failure reports; legitimate safe skips; and altered Circle prerequisites. The
permanent import test and isolated native gate fixture run without Circle or
migration files. Historical reports must be reproduced with their archived
source, not interpreted with the version 4 receipt schema.

The SP1 guest shadow follows Circle's `test-config-fallback` feature selection.
Both providers record the feature and the `-test` ELF source marker, require the
unsafe-test marker in the super-range guest, and discover guest tests with
`--all-features`. Production publication retains its upstream marker rejection.
SP1 runtime report verification is read-only; only the migration comparer can
recover manifest-declared empty Circle logs, with an audit record.
