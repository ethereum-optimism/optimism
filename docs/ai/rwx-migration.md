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

`ops/ci/go-rollup-tests.sh prepare` builds one sorted package manifest and two
round-robin shards. `go-package-shards.py` rejects empty discovery, Go package or
dependency errors, invalid shard inputs, and any assignment that omits or repeats
a package. Each shard validates that manifest before running the existing
gotestsum wrapper, retaining its three failure retries, JUnit, JSON and per-test
logs. The log artifact also includes the file logger's output under `tmp/testlogs`.
`-count=1` and `cache: false` ensure comparison runs execute tests; terminal
tasks disable filesystem output and publish reports as explicit
[artifacts](https://www.rwx.com/docs/artifacts) and
[test results](https://www.rwx.com/docs/test-results). Go JSON workflows should set
parser options `language: Go` and `framework: go test`, as this shadow does.
Each isolated shard uses a literal JSON report path; RWX associates it with the
parallel task. Test-result paths do not expand filename expressions.
Parallel tasks translate
[RWX shard metadata](https://www.rwx.com/docs/parallelism) into provider-neutral
`CI_SHARD_INDEX` and `CI_SHARD_TOTAL`.

The new run reuses the pilot's pinned checkout, full Git history, tool bootstrap
and routing adapter. A component-specific task installs the existing gotestsum
pin; a lockfile-filtered module download task feeds both shards. Source inputs
remain unfiltered because tests also import other Go components and embed NUT
bundle files. This scope needs no RPC credentials, cloud identity, publishers,
Docker, contract artifacts or Rust binaries. GitHub App setup remains necessary
for automatic pushes; authenticated CLI runs can exercise the workload beforehand.

From a trusted checkout, validate and run it with:

```bash
mise exec -- python ops/ci/test_go_package_shards.py
mise exec -- rwx lint .rwx/go-rollup.yml --warnings-as-errors
mise exec -- rwx run .rwx/go-rollup.yml --wait
```

## Standard contracts shadow

`.rwx/contracts.yml` adds the optional GitHub push status
`RWX: optimism-contracts-shadow` and accepts authenticated CLI execution without
posting a VCS status. The complete standard `contracts-bedrock-tests` workload
runs once without feature overrides and in the existing `CUSTOM_GAS_TOKEN`,
`OPTIMISM_PORTAL_INTEROP` and `ZK_DISPUTE_GAME` configurations. The main variant
uses `run-main`; the feature matrix uses shared `c-run_contracts_feature_tests`.
There is no path, test-name or shard filter. `develop` uses `ci`, other branches
use `liteci`; both retain 128 fuzz runs and 64 invariant runs at depth 32.
Fork and activation test flags remain disabled, retaining their existing
conditional skips.

A component bootstrap installs the repository's Forge/Cast 1.2.3 and svm-rs
0.5.19 pins, with solc 0.8.15, 0.8.19, 0.8.25 and 0.8.28. An unfiltered source
producer explicitly initializes public submodules, downloads Go modules and
builds Go FFI through Just. Keep its full source and Git state until narrower
cache inputs have been proved safe. Contract tool, compiler, submodule and Go
module downloads retry.

Each uncached verdict runs `just test` and the existing Go test-convention check.
The runner validates effective Foundry settings and a complete test-file
inventory, removes inherited filters/feature overrides, and preserves the first
test failure while collecting `just test-rerun` traces. Terminal tasks disable
filesystem output and export literal-path JUnit with `Solidity`/`Foundry` parser
labels, compiler output, configuration, inventory, traces and generated
counterexamples/file reports. Verify native parsed counts and failed-run artifact
collection in hosted runs before promoting any check.
This shadow needs no RPC credentials, vault, publisher, Docker or Rust producer.
Coverage, upgrade/fork, heavy-fuzz, snapshot and semver jobs remain outside this
bounded workload; the existing required contracts gate stays in CircleCI.

From a trusted checkout:

```bash
mise exec -- python ops/ci/test_contracts_shadow.py
mise exec -- rwx lint .rwx/contracts.yml --warnings-as-errors
mise exec -- rwx run .rwx/contracts.yml --wait
```

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
| `rust-op-reth-binary` | Release `op-reth` and `op-reth-sdm-fixture`, default features; export and verify both binaries |
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
Compare repeated same-SHA CircleCI samples, test counts, CPU/memory allocations
and billed usage before making a provider-wide speed or cost claim.

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
Repeat the measurements for that change. Further work includes reducing large
compiler-layer transfers, avoiding volatile Git inputs in compiler content keys
while preserving version identity, representative Rust source-change probes,
repeated samples and actual billed usage. Required checks remain on CircleCI.

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
