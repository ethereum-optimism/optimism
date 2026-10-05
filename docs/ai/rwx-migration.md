# CircleCI to RWX migration

CircleCI remains the merge-gating CI provider. The RWX pilot is for comparing
execution, caching, and feedback before moving required checks. It does not change
GitHub rulesets, fork authorization, schedules, or publishing credentials.

Track the full PR workload and remaining validation in
[rwx-parity-todos.md](rwx-parity-todos.md). Implementation and evidence updates
stay in the single draft
[PR #23151](https://github.com/ethereum-optimism/optimism/pull/23151)
against `develop`.

## Run the pilot

`.rwx/pilot.yml` runs shared routing tests plus policy-selected Go lint (with the
superchain bundle) and Rust formatting/upstream-mirror checks. Verdict tasks use
`cache: false`; tool setup can be reused. This is partial coverage, with the
optional push status `RWX: optimism-pilot`. No fork PR trigger or vault/token is
configured. Authorized `external-fork/*` pushes follow the existing Bailiff path.
Terminal verdict tasks disable filesystem output to avoid uploading unused state;
logs remain available. Export any future test reports as explicit artifacts.

Install repo tools using [dev-workflow.md](dev-workflow.md). RWX is initially a
standalone pinned CLI, outside the mise toolset. On Linux x86_64, install v3.32.1
into `~/.local/bin` following the [official CLI installation](https://www.rwx.com/docs/cli#pinning-a-version-for-scripts):

```bash
set -euo pipefail
RWX_CLI_DOWNLOAD=$(mktemp)
curl -fsSL --retry 5 --retry-delay 2 -o "$RWX_CLI_DOWNLOAD" https://github.com/rwx-cloud/rwx/releases/download/v3.32.1/rwx-linux-x86_64
printf '%s  %s\n' b67326b892b301f6c0abaaa69287fb9f53869c766216cb568e861131311991ef "$RWX_CLI_DOWNLOAD" | sha256sum -c - && {
  mkdir -p "$HOME/.local/bin"
  install -m 0755 "$RWX_CLI_DOWNLOAD" "$HOME/.local/bin/rwx"
}
rm -f "$RWX_CLI_DOWNLOAD"
export PATH="$HOME/.local/bin:$PATH"
```

The [linter](https://www.rwx.com/docs/cli-reference/rwx-lint) needs an installed
Node.js on `PATH` (22 or newer recommended). From the repository root:

```bash
mise exec -- bash ops/ci/test-decision-tree.sh
mise exec -- bash .circleci/scripts/test-decision-tree.sh
mise exec -- python -m unittest ops/ci/test_rwx_metadata.py
rwx --version
mise exec -- rwx lint .rwx/pilot.yml --warnings-as-errors
rwx login
rwx whoami
mise exec -- rwx run .rwx/pilot.yml --wait
```

Lint does not need an RWX login. A remote run requires an RWX organization and
authenticated CLI. [CLI runs](https://www.rwx.com/docs/cli-reference/rwx-run)
include local changes through Git patching; use a clean, pushed commit when
comparing providers. For automatic pushes, an organization administrator must
connect the repository using the [RWX GitHub App](https://www.rwx.com/docs/getting-started/github).
Account/app setup and remote execution must be verified separately from local lint.

## Go rollup shadow

`.rwx/go-rollup.yml` adds the optional GitHub push status
`RWX: optimism-go-rollup-shadow` and accepts authenticated CLI execution, which
does not post a VCS status. Both paths use the existing `run-main` routing value.
It runs every package under `./op-node/rollup/...` with `-tags=ci`, without `-short` or a
test-name filter. This is a complete component workload, not a replacement for
the aggregate Go gate or its dependent acceptance, Cannon, contract and Rust jobs.

`ops/ci/go-rollup-tests.sh prepare` builds one authoritative package manifest.
Four native RWX shards use longest-package-first placement from the recorded
package durations in `ops/ci/go-rollup-timings.json`. New packages receive a
median estimate; timing data only affects placement. Discovery and every shard
validate complete, duplicate-free assignment, including packages without tests.
The timing seed names the original RWX run and commit; refresh it from retained
Go JSON when package durations change.

A content-cached producer compiles each package with `go test -c -p=4 -tags=ci`.
Its native [tool cache](https://www.rwx.com/docs/tool-caches) retains downloaded
modules and Go compiler objects on source changes. Fresh verdict tasks consume
compiled binaries and source/fixtures as [artifact dependencies](https://www.rwx.com/docs/artifacts),
without inheriting the compiler object cache or Git history. They preserve the
package working directory, `-count=1`, at most four concurrent packages,
`-parallel=nproc`, `-timeout=40m`, and Go's panic-on-exit-zero behavior.
The existing gotestsum wrapper retains three failure retries, original Go JSON,
JUnit and per-test logs. Its raw-command reruns append `-test.run` plus a package;
only those diagnostic retry calls may select individual tests.

The import guards execute `go/packages` at runtime, so verdicts retain the pinned
Go toolchain and an artifact with the root package's production module sources
and downloaded module graph metadata. They do not bypass these tests or replace
them with a precomputed result. Runtime source, reporter and binaries are bound
to the compiler's authoritative package manifest and tested commit. `cache: false`
keeps verdicts fresh; terminal tasks only publish reports and test results.

The shared bootstrap installs Just, JQ, YQ and Python; Go, Go lint, Rust and
Foundry add separate pinned tool layers. Small bootstrap artifacts avoid passing
the checkout's Git history into tool preparation. Full source/Git state remain
compiler and routing inputs. This scope needs no RPC credentials, publishing
credentials, Docker, contract artifacts or Rust binaries. Mutable tool caches
use the existing cache-only vault, writable by `develop` and the temporary pilot
branch, with read-only access for other branches.

From a trusted checkout, validate and run it with:

```bash
mise exec -- python ops/ci/test_go_package_shards.py
mise exec -- rwx lint .rwx/go-rollup.yml --warnings-as-errors
mise exec -- rwx run .rwx/go-rollup.yml --wait
```

## Standard contracts shadow

`.rwx/contracts.yml` adds the optional GitHub push status
`RWX: optimism-contracts-shadow` and accepts authenticated CLI execution without
posting a VCS status. Its standard and changed-file feature matrices now use the
shared contract-suite adapter described in [rwx-contract-suites.md](rwx-contract-suites.md).
All four variants use the shared `c-run_contracts_feature_tests` route. Circle's
complete test-file discovery and actual timing split are retained; native selects
each file exactly once. Standard PR/develop profiles remain `liteci`/`ci`, with
128 fuzz runs and 64 invariant runs at depth 32. Changed-file tests retain
`ciheavy`: 20,000 fuzz runs, 128 invariant runs, depth 512 and 300-second fuzz
and invariant timeouts.

Compilation and fresh verdicts use full Git/source/submodule state, the Mise
Go/Forge/Cast pins, and the same four solc versions as Circle. Each suite/feature
has its own native Foundry/compiler cache. Artifacts bind complete tracked inputs,
source SHA, branch/profile, effective configuration, compiler signatures, bytecode
and file hashes. Runtime validates every input before consuming compilation,
keeps Go available for the original convention checker and fixtures, and uses a
separate Go compiler cache. Initial JUnit, diagnostic reruns, source selection,
commands, settings, logs and generated fixtures remain explicit evidence.

This definition uses the cache-only vault with no RPC or publishing credentials.
Its protected compiler warming executes zero verdicts. [Coverage](rwx-contract-coverage.md),
[L2 fork](rwx-contract-l2-fork.md), L1 upgrades and contract-fast checks have
their own completed definitions and evidence. The [Contracts aggregate](rwx-pr-gates.md)
now verifies all 21 exact prerequisites. Circle continues to own the required gate.

From a trusted checkout:

```bash
RWX_LIVE_CONTRACT_SUITE_FIXTURE=1 mise exec -- python ops/ci/test_contract_suites.py
mise exec -- python ops/ci/test_compare_contract_suites.py
mise exec -- rwx lint .rwx/contracts.yml --warnings-as-errors
mise exec -- rwx run .rwx/contracts.yml --wait
```

## Cache warming and resource trials

The producer definitions configure [cache-rebuild triggers](https://www.rwx.com/docs/cache-rebuild-triggers)
restricted to `develop`. Go/Foundry/op-reth triggers target preparation and
compiler tasks only; the pilot warms its independent language tool layers.
Warm-only routing validates the checkout and emits false verdict flags without
running PR change detection. Keep this routing task successful: RWX automatically
skips a task when a referenced dependency was skipped, even if an `if` expression
could bypass its value. Compiler tasks opt in explicitly through the warm flag.
Warming does not run verdicts, publish check successes, or invoke publishers.
A CLI `--init cache-warm=true --target <compiler>` rehearses task selection;
it is not proof that the native protected-branch cache-rebuild event fired.
The definitions must reach `develop` before protected-branch warming is active.
RWX rebuild events reset the native cache's initial layer and 48-hour TTL.

Use `--init build-probe=<label>` to force a compiler content-cache miss while
retaining its native cache, or a new `--init cache-epoch=<label>` for cold mutable
caches. Do not disable caching on a compiler to measure incremental tool reuse:
RWX disables its tool cache too. Tests remain `cache: false` in every trial.
The current organization accepts at most 16 CPUs per task: a 32-CPU/64-GB
trial was rejected by the hosted service despite the broader public runner
catalog. Use the supported 2/4/8/16-CPU combinations until availability changes.
Record actual resources, all task preparation/execution/post-processing times,
whole-run elapsed time, source revision/patch and retained reports. An old
execution duration attached to a content-cache hit is historical, not time spent
compiling during that run. Keep cold and warm measurements separate.

### Speed implementation validation (October 2)

CLI trials used `9e863c1546` with explicitly uploaded, uncommitted CI changes;
they are configuration experiments, not a new same-SHA CircleCI comparison.
Whole-run figures below are `CompletedAt - StartedAt`, including setup and
waiting within the run. The API's `CompletedRuntimeSeconds` omits some waiting
and must not be substituted for this wall-clock metric.

| Workload | Earlier native run | Updated CLI observations | Retained outcomes |
| --- | ---: | ---: | --- |
| Go rollup | 140.8s | [121.3s](https://cloud.rwx.com/optimism/runs/bf4a4ca09cdf41d6aa8bde83e480b2eb), [107.2s](https://cloud.rwx.com/optimism/runs/0e86b64ca7df4af1a0bc994a0cfe7193) | 1,245 pass / 2 skip |
| Standard contracts | 592.1s | [360.0s](https://cloud.rwx.com/optimism/runs/67de4a0549374b4482bf0c9f32afee02), [263.0s](https://cloud.rwx.com/optimism/runs/2b74a989bc2f49e7ad98e48f5c8e0362) | 9,503 pass / 638 skip |

Both compiler cache state and runner-local layers differed between observations;
collect repeated timings at the final pushed SHA before interpreting medians as
an expected PR latency. The Rust release producer executed in 230s on 8 CPUs
and 137s on the supported 16-CPU/32-GB runner. The latter
[trial](https://cloud.rwx.com/optimism/runs/27513cd6d1c1428cbebd7ef5d3b1a962)
verified both binaries; release/codec producers now share the integration
runner specification. This is a producer measurement, not a whole-workload
provider speed claim.

The [Go source-change probe](https://cloud.rwx.com/optimism/runs/0520c39b016e44fc986399830a8ec2bf)
missed the compiler content cache, restored its native cache, and reported an
intentional new test failure through the initial attempt plus all three retries.
All 1,247 original cases remained present; the probe was removed without being
committed. Original Go JSON retained all four failing attempts. The
[warm-only rehearsal](https://cloud.rwx.com/optimism/runs/1c7d233c758f4afeb8a185be9976cb0e)
executed routing and compilation with zero tests and false verdict flags. Earlier
warm rehearsals that skipped the compiler are excluded from this evidence.

The native GitHub push at `6eecce7cb4` passed all four optional RWX checks:

| Workload | Run-start to completion | Retained outcomes |
| --- | ---: | --- |
| [Pilot](https://cloud.rwx.com/optimism/runs/1b92ee67b3f34c838ec9ff335aff7172) | 189.7s | Go lint and Rust formatting passed |
| [Go rollup](https://cloud.rwx.com/optimism/runs/773d558c04fb4659a9583098c3f0ad58) | 124.9s | 1,247 outcomes, no failures |
| [Standard contracts](https://cloud.rwx.com/optimism/runs/ffcb42f9f4d04931ba1b31ec36a8fe4f) | 260.8s | 10,141 outcomes, no failures |
| [Op-reth](https://cloud.rwx.com/optimism/runs/e14f9ddf23234958b3b34dda5015a981) | 289.1s | Both binaries, 50 runnable tests, codec and snapshots passed |

The earliest run start to the last shadow completion was 289.2s (4m49s),
compared with the earlier 592.1s (9m52s) observation. These are individual
observations with different cache/local-layer states. The codec baseline resolved
to develop `055562c9186c70fa01bb548e83fe6b06f47e7b29`; retain this SHA when
repeating the workload. Release compilation executed in 121s; its original
metadata reported 16 CPUs, no CPU quota, and a 30-GiB cgroup memory limit for
the requested 32-GB runner. Push/queue time before run start is unavailable.
Required CircleCI gates and current warm-run medians are refreshed in PR #23151;
these RWX observations do not establish a provider speed comparison.

## Compare retained test evidence

Use [ci-comparison.md](ci-comparison.md) to collect reports from both providers
on the same pushed commit, compare test identities, outcomes and skips, and report
observed retry history. The report distinguishes missing or different cases
from evidence that is incomplete or belongs to a different revision, profile or
routing context.

An initial automatic push on October 1, 2026 completed all three optional checks
at commit `02d29aa13f9da3d8e51a1f52a987eb69e1b6bda0`: [pilot](https://cloud.rwx.com/optimism/runs/f8d83016615a4dcabd89b0e79c055610),
[Go rollup](https://cloud.rwx.com/optimism/runs/fbd271ef52184767a0bf8f2c03e1b393),
and [standard contracts](https://cloud.rwx.com/optimism/runs/d10bf0a7eb65404aac0782d41d386ea7). CircleCI's project setting
`build-prs-only` is enabled: a branch-only push starts RWX, while an open PR is
needed for the CircleCI baseline. Keep the actual provider trigger in the
collection; compare the verified shared routing context rather than relabeling
an event. GitHub status details link to the corresponding native RWX run.

Cached RWX tasks may expose execution timestamps and durations from the run that
created the cache entry. Exclude those historical execution values from current
compute totals. Retain cache hits separately, use run start/end timestamps for
elapsed time, and leave billed time and price unknown when the provider does not
return them. A warm run alone does not establish cold-cache performance.

## Op-reth shadow and cache measurements

`.rwx/op-reth.yml` adds `RWX: optimism-op-reth-shadow`. The release producer uses
`run-main`; integration, compact-codec and superchain-snapshot checks use
`run-rust-ci`. The four workloads retain CircleCI's package selection and features:

| CircleCI job | RWX producer and fresh verdict |
| --- | --- |
| `rust-op-reth-binary` | Release `op-reth`, default features; export and verify the binary (the standalone SDM fixture was removed upstream) |
| `op-reth-integration-tests` | Archive every `reth-optimism-node` test with nextest; run the archive using the committed default nextest configuration |
| `op-reth-compact-codec` | Compile pinned `develop` and the PR with `dev` features in parallel; generate fresh baseline vectors and read them with the PR binary |
| `op-reth-superchain-snapshot-check` | Compile chainspec with `superchain-configs` and sync flag `0`; switch to `1` to regenerate and compare both committed snapshots |

Builds use `--locked`, the repository's Rust/mold/protoc/nextest pins, and
sccache 0.18.0 with a verified release checksum. `CARGO_INCREMENTAL=0` keeps
sccache enabled. Build helpers live in the reusable tool layer so the baseline
checkout can use them without receiving PR Rust source. Public
`superchain-registry` initialization uses each checkout's own gitlink; compilers
retain the generated tar only when its checksum matches the checkout's committed
pin. A mismatching tar is removed so `build.rs` regenerates it. Keeping the matching
tar and its timestamp avoids rebuilding chainspec and relinking its consumers.

Release and each codec compiler use 8 CPUs / 16 GB; integration compilation and
execution use 16 CPUs / 32 GB; snapshot compilation and regeneration use
4 CPUs / 8 GB. These match CircleCI's job allocations. Codec's two compilers run
concurrently, so its peak allocation is twice that of the serial CircleCI job;
retain total usage as well as elapsed time when comparing it.

The cache layers serve different purposes:

- Content caching reuses an identical compiler output. Full source, Git state,
  pins, profiles, features, runner and commands remain inputs.
- A shared Rust dependency download layer feeds all producers. It retains the
  complete Rust manifest/source tree for Cargo discovery and uses `cargo fetch
  --locked`; the base can download any additional dependencies its lock requires.
- Separate native [tool caches](https://www.rwx.com/docs/tool-caches) preserve
  Cargo registry/git downloads, `rust/target`, and local sccache entries across
  compiler input changes. Each producer has its own cache to avoid concurrent
  writers. Filesystem output filters exclude unrelated source and system changes.
- Nextest archives transfer compiled tests, metadata, dynamic libraries and build
  outputs. Running them does not recompile or cache test verdicts.
- [Artifact mounts](https://www.rwx.com/docs/artifacts) deliver the release/codec
  binaries, nextest archive and fresh vectors to consumers independently of
  compiler filesystem layers. Consumers use the PR checkout and tool layer;
  artifact checksums and source revisions remain mandatory before execution.
  Input file filters change cache keys; they do not remove inherited layer
  downloads. Only snapshot regeneration needs the producer's Cargo targets.

The cache-only vault `optimism-op-reth-shadow` has no secrets, OIDC credentials
or production GCS writer. Its write permissions currently cover `develop` and
the temporary `codex/rwx-ci-pilot` branch. Remove the pilot permission and evict
its test caches before promoting this shadow or sharing protected cache entries.
Other branches can read the tool caches but cannot write them. These tool caches
expire after 48 hours; content caching is independent of that lifetime.
The vault must also opt into access from public repositories: repository/branch
permissions alone do not enable writes for this public monorepo. The pilot has
that toggle enabled and one-day access for the authenticated benchmark user.
CLI runs need a user grant; setting an init `branch` does not establish native
GitHub trigger identity or grant vault write access.

Every verdict and baseline vector generation has `cache: false`. This also
disables tool caching on that task, so compilation lives in separate cached
producers. The snapshot producer records sync flag `0`; `build.rs` declares the
flag as an environment input, so the uncached `1` invocation regenerates even
when the target directory is restored. Baseline vectors retain CircleCI's random
100 values per type on every run. Only the immutable baseline executable is
eligible for reuse; its cache inputs omit the unrelated PR SHA.

Artifacts include SHA-bound binary/archive checksums, the pinned base revision,
vector checksums, original nextest JUnit, full discovery, explicit ignored-test identities/reasons, per-case coverage and
retry evidence, logs, compiler cache statistics and elapsed time. Missing or
duplicate verdicts, unexpected filters/skips, empty discovery, stale revisions
and corrupted build outputs fail verification. Diagnostic failures preserve the
original test/build failure. Nextest keeps its existing single-test retry
exception; RWX records retries in the report rather than sending Slack messages.

Validate with:

```bash
python3 ops/ci/test_op_reth_shadow.py
shellcheck ops/ci/op-reth-shadow.sh ops/ci/rwx-rust-prepare.sh
rwx lint .rwx/op-reth.yml --warnings-as-errors
rwx run .rwx/op-reth.yml --wait
```

For measurements, use a clean pushed checkout and pin the same `develop` SHA
for every sample. CLI runs remain separate from native GitHub check evidence.
The following examples target the release path; omit `--target release` to run
all four workloads:

```bash
# Cold compiler caches, with tool installation measured separately.
rwx run .rwx/op-reth.yml --target release --init cache-epoch=measurement-1 \
  --init codec-base-sha=<develop-sha> --title 'op-reth cold' --wait
# Identical inputs: content-cache reuse, while the binary verdict executes.
rwx run .rwx/op-reth.yml --target release --init cache-epoch=measurement-1 \
  --init codec-base-sha=<develop-sha> --title 'op-reth content warm' --wait
# Force a compiler content miss; retain Cargo targets and sccache entries.
rwx run .rwx/op-reth.yml --target release --init cache-epoch=measurement-1 \
  --init build-probe=target-warm-1 --title 'op-reth target warm' --wait
# Clear only targets; restored sccache entries must carry the compilation.
rwx run .rwx/op-reth.yml --target release --init cache-epoch=measurement-1 \
  --init build-probe=sccache-warm-1 --init target-cache-mode=sccache-only \
  --title 'op-reth sccache warm' --wait
```

`build-probe` changes a compiler input, rather than setting `cache: false`, which
would disable the tool cache under measurement. A new epoch creates fresh tool
cache entries without evicting other workloads. These knobs measure compiler
cache state; they do not make the image/tool layer cold. Preserve task preparation,
execution and output-upload time separately. Verify actual tool-cache reads and
writes from RWX task metadata, not the requested epoch alone. Retain source-change
and failing-verdict probes before claiming cache correctness across revisions.
The pilot optimizes push-to-final-verdict wall time, including queueing, setup,
cache restoration, transfers, builds and fresh tests. Label run-start timings when
earlier timestamps are unavailable. Compare repeated same-SHA, same-workload
CircleCI samples and retain test counts, actual CPU/memory allocations and cache
state. Runner sizes may differ: choose resources for wall-clock speed. Cost and
billed-usage analysis are deferred.

### First hosted measurements

At `d2b9024161341a61fa48dcfa2f177e92651d220d`, all four workloads passed in
[native run b2a03116](https://cloud.rwx.com/optimism/runs/b2a0311680d04079ac335e1439e75a7e).
The original integration JUnit matched
[CircleCI job 5625702](https://circleci.com/gh/ethereum-optimism/optimism/5625702)
for all 50 reported cases, with no missing, extra or changed verdicts. RWX's
discovery retained the additional ignored `p2p::can_sync` case. RWX parsed all
50 JUnit results in its UI. Both release binaries, fresh codec vectors at baseline
`ea9cce9b1c5d86336da52e2346e666d6a21a956b`, and both regenerated snapshots passed.

The first complete native run took 13m10s. The
[complete warm CLI run](https://cloud.rwx.com/optimism/runs/2c7e09f8233541869ff6fa8771b3d372)
took 5m58s with all verdicts and random vector generation executing again.
Tool installation and dependency layers were reusable in both runs. The warm
sample restored native tool caches; all five compiler producers still executed.
These are single observations with different provider trigger contexts.

| Build phase | Initial compiler state | Restored Cargo targets and sccache |
| --- | ---: | ---: |
| Release, both binaries | 517s | 237s |
| Integration archive | 354s | 126s |
| Codec baseline | 381s | 47s |
| Codec PR | 379s | 48s |
| Snapshot preparation | 65s | 24s |

Separate snapshot probes forced compiler content misses. The
[target-retaining probe](https://cloud.rwx.com/optimism/runs/8c3069f390034f8682b2695cbe01f927)
compiled in 24s; the
[empty-target/sccache probe](https://cloud.rwx.com/optimism/runs/8552f40db3314cf282583313806e0be6)
compiled in 36s with 263 Rust hits, 72 C/C++ hits, two assembler hits and no cache
misses. The initial 65s snapshot build had zero hits and 263 Rust misses.
These are build-phase times, excluding preparation, output upload and the fresh
snapshot verdict. Requested snapshot resources were 4 CPUs / 8 GB; metadata
reported four CPUs and a 6 GiB container memory limit. Capture actual limits from
both providers before treating declared resource classes as identical hardware.

Same-SHA CircleCI jobs also passed: release
[5625647](https://circleci.com/gh/ethereum-optimism/optimism/5625647) took 9m50s,
integration took 5m55s, codec
[5625720](https://circleci.com/gh/ethereum-optimism/optimism/5625720) took 7m39s,
and snapshot
[5625721](https://circleci.com/gh/ethereum-optimism/optimism/5625721) took 1m32s.
Those are whole-job durations, including their own cache and setup behavior;
they are not directly comparable to the RWX compiler-phase table. The cold RWX
run still has substantial preparation and transfer overhead.

An [intentional CLI-only verdict fault](https://cloud.rwx.com/optimism/runs/add6576e7432497a8a1e3f85a40a8313)
failed one selected integration test after archive extraction and discovery. The
task and run failed, retained original JUnit and coverage, and preserved nextest's
exit 100 despite incomplete coverage caused by fail-fast cancellation. The
committed Rust tests were unchanged. Successful warm execution subsequently
passed all 50 cases.

These samples exposed a missing generated-bundle cache output. Losing or deleting
the tar forced chainspec regeneration and relinking even with restored targets.
The producer now retains a checksum-matching tar, with a regression check for
preserving its timestamp and invalidating it when the committed pin changes.
At `99ccc76a`, all four workloads passed in
[native run 442bd7b8](https://cloud.rwx.com/optimism/runs/442bd7b81c0044e1b41d0178a8c63a25),
and the 50 original JUnit identities/outcomes matched CircleCI job 5625810.
A [forced target-cache probe](https://cloud.rwx.com/optimism/runs/8545eb684d094e13abc4e91fdab645fe)
executed snapshot preparation in 7s, with six Rust sccache hits and no misses;
fresh regeneration still executed for 24s. A separate
[identical-input release rerun](https://cloud.rwx.com/optimism/runs/be0b0006cd5044c88739ec46cd81f2d6)
at `d2b90241` hit the compiler content cache and completed the release path in
8s, including fresh binary verification.

The `99ccc76a` native run took 8m30s. Its codec verifier spent 171s preparing,
including downloading two 11,516 MiB compiler cache layers, before executing in
9s. The consumers now mount artifacts instead of inheriting compiler caches.
At `0b7935e7`, [native run 0d4989f8](https://cloud.rwx.com/optimism/runs/0d4989f8fc044b48888ba7cdda9e9b2a)
passed all four workloads in 6m20s. The codec verifier prepared in 2s and executed
in 5s without inheriting op-reth compiler layers. These are single observations
with different runner-local layer states.

At the same revision, three cold-compiler snapshot probes and three restored-target
probes passed fresh regeneration. Median whole-run wall time was **126.3s versus
55.6s**; median compiler execution was 65s versus 6s. Images, tools and dependency
layers were warm, and cold samples 2/3 ran concurrently. The
[empty-target/sccache sample](https://cloud.rwx.com/optimism/runs/636452ced9274d0d9b7c211bd6611cb3)
took 63.6s for the whole run, including 17s compiling with 263 Rust and 74 C/assembly hits
and no misses. All samples reported 4 CPUs and a 6 GiB container limit. These
measurements show cache benefits within RWX; a full provider comparison remains open.

An [uploaded Rust assertion mutation](https://cloud.rwx.com/optimism/runs/11d82d12b2cf4635a8c8b2c22cac734b)
invalidated compiler content reuse, executed with restored targets and failed the
intended case with nextest exit 100. Original reports, discovery and the actual
source patch/hash remained downloadable. A
[clean recovery](https://cloud.rwx.com/optimism/runs/198aad232de64004af4938d485336d6a)
reused the clean compiler output and ran a fresh verdict: all 50 cases passed in a
48.8s whole run. For local source probes, omit an explicit `--init commit-sha`,
which makes the CLI skip the local patch, and verify retained source hashes.

The October 2 implementation above trims inherited layers, reuses Go/Foundry
compilation, and increases the release producer to the measured 16-CPU runner.
Continue collecting repeated whole-run latency on pushed revisions and extend
source-change invalidation probes as shared producers and concurrent consumers
are added for aggregate Go and acceptance tests. Required checks remain on
CircleCI.

## Migration contract

Preserve the existing routing and test coverage before tuning performance:

- PRs run the main workflow, with contract and Rust suites selected by changed
  paths. Only changes entirely inside `docs/public-docs/` take the docs fast path.
- Merge queue runs the main and contract suites, with Rust selected by changed
  paths. Docs-only merge groups still emit all required gates.
- `develop` runs the full post-merge set unconditionally. Diffing `develop`
  against itself cannot select post-merge suites correctly.
- Tags retain component-specific release filters. Schedules and manual dispatches
  retain their current workload selection.
- Changes to `.circleci/`, `.rwx/`, and `ops/ci/` select CI, contract, and Rust
  validation. A new or otherwise unclassified code path must not take a docs skip.

`ops/ci/routing.yml` and `ops/ci/compute-workflow-conditions.sh` hold shared policy;
CircleCI retains adapters in `.circleci/scripts/` and a routing-data symlink.
`ops/ci/rwx-metadata.sh` maps push/CLI metadata into that policy and exports
`run-main`/`run-rust-ci` values. Extend the shared routing tests for new selection
rules, retaining adapter parity and real changed-file fixtures.
CircleCI's per-job `ops/check-changed/main.py` rebuild policy also includes `.rwx/`
and `ops/ci/`, so selected contract jobs execute on CI changes.

## Inventory before cutover

For each workload, capture commands, dependencies, shards, resources, timeouts,
caches, output files/results, credentials and merge-blocking status. Compare
coverage, cold/warm duration, critical-path time, cost and failures on the same
commit/event.

Checked-in configuration does not contain all operational state. Export and
review these separately:

- GitHub ruleset contexts, provider identities, branch patterns, and bypass rules.
- CircleCI context variable names and restrictions, project settings, concurrency,
  scheduled trigger cadence/timezone/branch, and API dispatch clients.
- GCP workload identity providers, claim conditions, service accounts and bucket
  permissions; Bailiff's deployed version, team configuration, and webhook setup.

Inventory secret names and permissions without exporting their values into the
repository or run artifacts.

### Historical CircleCI baseline

A read-only API snapshot collected on October 1, 2026 sampled five successful
`develop` pipelines created between `2026-09-30T16:23:15.165Z` and
`2026-10-01T13:43:39.973Z` (UTC). All four core workflows
succeeded in each sample; 435 job records include executor classes and parallelism.

| Scope | Median elapsed time | CircleCI executor / shards |
| --- | --- | --- |
| Main workflow | 25.77 min | Multiple |
| Rust CI workflow | 17.48 min | Multiple |
| Rust E2E workflow | 19.78 min | Multiple |
| Contracts workflow | 16.50 min | Multiple |
| Setup workflow | 47 sec | Single setup job |
| Go lint job | 5.07 min | Docker `large-gen2` / 1 |
| Rust formatting job | 1.37 min | Docker `medium-gen2` / 1 |

Sampled commit SHAs, oldest first:

```text
5686e957e7456831d8891356e6a8ab145f0439c4
031adbfb268906396306a41ffde0d78f0120bf81
777fc39a9220814391ee1fdf99f56b923ad487ca
da6d3252491754837a778061db0cc47236ec13c6
a271fd6b3feb76e49f3453e5a27be4b33a3c7211
```

The main execution path consistently passed through op-reth compilation, sysgo
binary aggregation, and acceptance tests. These successful-run medians are
historical orientation, not performance targets or failure-rate estimates. RWX
uses different CPUs, caches, and checkout history; compare equivalent workloads
on the same commit/event before attributing timing differences to the provider.
Job elapsed time is not total billed time across shards.

The live schedule inventory had five definitions targeting `develop`, while
routing mapped three. `build_sunday_early` (Sunday hour 01 UTC) and `build_mon_thu`
(Monday/Thursday hour 07 UTC) had no routing entry and therefore selected no
continuation workflows under the current policy. Confirm their owners and intent
before carrying them over or removing them. The mapped schedules used four-hourly
hours 00/04/08/12/16/20, daily hour 04, and Sunday hour 00. CircleCI schedules run
within the specified UTC hour rather than guaranteeing its exact start. See
[scheduled trigger timing](https://circleci.com/docs/guides/orchestrate/schedule-triggers/).

All nine referenced contexts were accessible through the restriction API. Six
exposed only the organization-wide All members group. The repository write-token
context additionally had project and branch-expression restrictions; the sccache
context restricted access to the two Optimism repositories; the release context
required the release-managers group. Preserve these restrictions when transferring
credentials. GCP federation and bucket authorization still need a separate audit;
CircleCI context restrictions alone do not establish those permissions.

## Required checks and fork authorization

The `develop` ruleset currently requires CircleCI contexts for `ci-gate`,
`required-contracts-ci`, `required-rust-ci`, and `required-rust-e2e`.
`dependency-review` is a separate GitHub Actions requirement. Match both the
context and its provider when reviewing a proposed ruleset change; preserving a
display name alone does not transfer a requirement to RWX.

The CircleCI gates use terminal dependencies and the private
`ethereum-optimism/circleci-utils` orb to inspect upstream results. RWX replacements
must report a failure after a failed/canceled build or test, even when descendants
never ran. An intentional safe skip must still produce every required context.
Gate coverage includes exact matrix names, not just job templates.

Test reporting on actual PR/merge-group commits, superseded runs and cancellation
before changing requirements. Keep distinct, non-required RWX checks meanwhile.

Legacy rulesets `enforce-circleci-check-old-backports` and
`enforce-circleci-check-porposal-v3` cover `backports/op-deployer/v*` and
`proposal/op-contracts/v*` with older CircleCI and image checks. Backport replacement
config/requirements or retain CircleCI for those branches. Re-read live rulesets
before cutover; these names describe the audited baseline.

Keep human authorization of exact fork commits through Bailiff's internal
`external-fork/*` pushes. New commits require new authorization. Verify check
association with the approved SHA and prevent privileged direct fork execution,
including access to secrets or writable trusted caches. See
[PR authorization](../../ops/book/src/ci/pr-authorization.md) and
[the human-only authorization rule](../handbook/pr-guidelines.md#triggering-ci-on-prs-from-external-forks).

## Integration blockers

| Area | Current dependency | Required replacement |
| --- | --- | --- |
| Toolchain and checkout | Private orb installs mise, warms tools, and handles checkout | Explicit pinned toolchain and checkout tasks, including submodules and required Git history |
| Shards | `circleci tests split`, `CIRCLE_NODE_*`, and CircleCI JUnit timing history | RWX sharding with complete test-set coverage and provider-neutral shard inputs |
| Metadata | Branch, PR, repository, workflow ID, and run URL in CircleCI variables | Explicit shared metadata; preserve true PR base lookup and merge-group SHA |
| Workspaces | Contract outputs, binaries, prestates, and gitignored `superchain-configs.zip` | Explicit producer/consumer dependencies and retained outputs |
| Reporting | JUnit uploads, log/artifact paths, CircleCI Insights flake API | Accessible failed-run artifacts and a replacement for flake history/reporting |
| Cloud identity | `CIRCLE_OIDC_TOKEN` and GCP workload identity | Separate RWX vault subjects and exact-sub GCP bindings; prove equivalent ref restrictions before writer cutover |
| Rust compile cache | sccache GCS reader for all refs; writer only for `develop` | Enforced reader/writer identity split or a separately validated RWX cache design |
| Release and publish | GoReleaser, contract artifacts, Cannon/SP1 prestates | One active publisher per destination, preserving component tag/ref filters |
| Maintenance | TODO/Cannon every four hours; daily suites; weekly nightly PR; labels and stale automation | Explicit ownership, cadence, permissions, and one active trigger |

Audit `justfile`, `ops/scripts/shard-tests.sh`, acceptance/Kona justfiles, and contract
target-branch/semver scripts. Compatibility variables do not replace Circle's CLI
or timing history. `CIRCLECI` also changes dependency builds and Docker pruning.

[RWX OIDC](https://www.rwx.com/docs/oidc) documents issuer
`https://cloud.rwx.com/mint`, a vault-identifying `sub`, audience and run/task IDs;
the documented claims contain no signed repository, ref or trigger identity.
Reader and writer identities need separate vault subjects and exact-sub GCP
bindings. These enforce vault identity without independently verifying GitHub refs.
[Locked vaults](https://www.rwx.com/docs/vaults) restrict repository/ref access but
also permit user/service-account grants. Keep the current CircleCI sccache writer
until denial tests prove CLI runs, local patches and unauthorized refs cannot
access the writer vault, or signed VCS claims/external attestation restores that
policy.

Preserve op-deployer's release settings for fresh tools/modules and disabled caches.

Tests must execute on comparison runs; retain retries/flaky-result reporting.
Dependency/build cache reuse must include its real inputs.

## Rollout and rollback

1. **Pilot:** run the small workload above on an exact commit; compare results.
2. **Shadow:** add Go/contract/Rust suites with equivalent shards and dependencies,
   then compare PRs and merge groups using distinct, non-required RWX checks.
3. **Gates:** inject failures, cancellations and safe skips; exercise internal PRs,
   authorized forks and merge groups. Verify all gates and legacy branches before
   changing requirements.
4. **Post-merge:** validate full `develop`, schedules, dispatches and governance
   writes. Transfer each trigger to one provider.
5. **Publishers:** validate release-candidate artifacts/provenance and identity
   restrictions without publishing, then transfer each destination to one provider.
6. **Retirement:** remove obsolete config/tokens/identities after the rollback window
   and legacy branch obligations. Keep unrelated GitHub Actions workflows.

Keep the CircleCI config runnable, preserve exported rulesets/trigger settings, and
retain its required identities during comparison. A rollback restores previous
required contexts and transfers trigger/publisher ownership back; it must not
activate duplicate publishing or maintenance runs.

Before retirement, update `ci-ops.md`, `ci-config-review.md`, the CI reviewer agent,
the TODO repair skill, authorization/runner runbooks, feature-matrix documentation,
and Kona SP1 publishing guidance. The image-provenance GitHub Action currently
parses the Rust image pin from `.circleci/config.yml`; move its source and path
triggers together if that pin moves.

The completed aggregate Go stage and deferred follow-ups are tracked in
[rwx-go-parity.md](rwx-go-parity.md). It expands coverage to Circle's complete
`go-tests` selector while preserving required gates and fresh-test evidence.

Full fresh Go parity is verified at `cf7f3f2d51ea5e75b9eb21adcb4cf7a860a15bce`: 459 packages, 11,490 passes, 123 skips and zero retries on both providers. The selected configuration is 12 shards / parallel 8, with 16 CPU / 32 GiB compile and 8 CPU / 16 GiB verdict workers. Native Go routing now uses the full suite; rollup remains CLI-only. Repeated performance tuning is deferred, and single-run timing observations do not establish a speed win. Circle retains all required gates.
